package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/files"
	"github.com/zeile/tui/internal/infrastructure/repository"
	"github.com/zeile/tui/internal/infrastructure/storage"
	"github.com/zeile/tui/internal/preprocessing"
)

type ImportProgressFunc func(stage string, percent float64)

type LibraryService struct {
	books      BookRepository
	states     ReadingStateRepository
	processors *preprocessing.Registry
	paths      storage.Paths
}

func NewLibraryService(
	books BookRepository,
	states ReadingStateRepository,
	processors *preprocessing.Registry,
	paths storage.Paths,
) *LibraryService {
	return &LibraryService{
		books:      books,
		states:     states,
		processors: processors,
		paths:      paths,
	}
}

func (s *LibraryService) ImportBook(ctx context.Context, sourcePath string, managedCopy bool, onProgress ImportProgressFunc) (domain.Book, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(sourcePath))
	if cleanPath == "" {
		return domain.Book{}, errors.New("book path is required")
	}

	if onProgress != nil {
		onProgress("Checking file", 0.05)
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return domain.Book{}, fmt.Errorf("stat source file: %w", err)
	}
	if info.IsDir() {
		return domain.Book{}, errors.New("directories are not supported; select a single EPUB or PDF file")
	}

	format, err := domain.DetectFormat(cleanPath)
	if err != nil {
		return domain.Book{}, err
	}

	if onProgress != nil {
		onProgress("Calculating fingerprint", 0.15)
	}

	fingerprint, err := files.FingerprintSHA256(cleanPath)
	if err != nil {
		return domain.Book{}, err
	}

	existing, err := s.books.GetByFingerprint(ctx, fingerprint)
	if err == nil {
		return existing, ErrBookAlreadyImported
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return domain.Book{}, err
	}

	bookID := uuid.NewString()
	managedPath := cleanPath
	if managedCopy {
		managedPath = s.paths.ManagedBookPath(bookID, format)
		if onProgress != nil {
			onProgress("Copying to managed library", 0.3)
		}
		if err := files.CopyFile(cleanPath, managedPath); err != nil {
			return domain.Book{}, err
		}
	}

	cleanupOnError := func() {
		if managedCopy {
			_ = os.Remove(managedPath)
		}
		_ = os.RemoveAll(s.paths.BookCacheDir(bookID))
	}

	if onProgress != nil {
		onProgress("Preprocessing", 0.45)
	}

	processor, err := s.processors.ForFormat(format)
	if err != nil {
		cleanupOnError()
		return domain.Book{}, err
	}

	result, err := processor.Process(ctx, preprocessing.Input{
		BookID:      bookID,
		SourcePath:  cleanPath,
		ManagedPath: managedPath,
		CacheDir:    s.paths.BookCacheDir(bookID),
	}, onProgress)
	if err != nil {
		cleanupOnError()
		return domain.Book{}, err
	}

	now := time.Now().UTC()
	book := domain.Book{
		ID:          bookID,
		Fingerprint: fingerprint,
		Title:       defaultString(result.Title, strings.TrimSuffix(filepath.Base(cleanPath), filepath.Ext(cleanPath))),
		Author:      defaultString(result.Author, "Unknown"),
		Format:      format,
		AddedAt:     now,
		SourcePath:  cleanPath,
		ManagedPath: managedPath,
		Metadata:    result.Metadata,
		SizeBytes:   info.Size(),
	}

	if err := s.books.Create(ctx, book); err != nil {
		cleanupOnError()
		return domain.Book{}, err
	}

	defaultMode := domain.DefaultModeForFormat(book.Format)
	if err := s.states.Upsert(ctx, domain.ReadingState{
		BookID:          book.ID,
		Mode:            defaultMode,
		Locator:         domain.Locator{Offset: 0, PageIndex: 0},
		ProgressPercent: 0,
		UpdatedAt:       now,
		IsFinished:      false,
	}); err != nil {
		cleanupOnError()
		return domain.Book{}, err
	}

	if onProgress != nil {
		onProgress("Imported", 1)
	}

	return book, nil
}

func (s *LibraryService) ListBooks(ctx context.Context) ([]domain.Book, error) {
	return s.books.List(ctx)
}

func (s *LibraryService) SearchBooks(ctx context.Context, query string) ([]domain.Book, error) {
	books, err := s.books.List(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return books, nil
	}

	tokens := strings.Fields(query)
	type scoredBook struct {
		book  domain.Book
		score int
	}
	scored := make([]scoredBook, 0, len(books))

	for _, book := range books {
		fields := []string{strings.ToLower(book.Title), strings.ToLower(book.Author), strings.ToLower(filepath.Base(book.ManagedPath))}
		score := 0
		for _, token := range tokens {
			tokenMatched := false
			for _, field := range fields {
				switch {
				case field == token:
					score += 120
					tokenMatched = true
				case strings.HasPrefix(field, token):
					score += 80
					tokenMatched = true
				case strings.Contains(field, token):
					score += 40
					tokenMatched = true
				}
			}
			if !tokenMatched {
				score = 0
				break
			}
		}

		if book.LastOpened != nil {
			score += 15
		}

		states, err := s.states.ListByBook(ctx, book.ID)
		if err == nil {
			unfinished := false
			for _, state := range states {
				if !state.IsFinished {
					unfinished = true
					break
				}
			}
			if unfinished {
				score += 20
			}
		}

		if score > 0 {
			scored = append(scored, scoredBook{book: book, score: score})
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			left := scored[i].book.LastOpened
			right := scored[j].book.LastOpened
			switch {
			case left == nil && right == nil:
				return scored[i].book.AddedAt.After(scored[j].book.AddedAt)
			case left == nil:
				return false
			case right == nil:
				return true
			default:
				return left.After(*right)
			}
		}
		return scored[i].score > scored[j].score
	})

	result := make([]domain.Book, 0, len(scored))
	for _, item := range scored {
		result = append(result, item.book)
	}
	return result, nil
}

func (s *LibraryService) RemoveFromLibrary(ctx context.Context, bookID string) error {
	if _, err := s.books.GetByID(ctx, bookID); err != nil {
		return err
	}

	if err := s.books.DeleteByID(ctx, bookID); err != nil {
		return err
	}
	_ = os.RemoveAll(s.paths.BookCacheDir(bookID))
	return nil
}

func (s *LibraryService) DeleteFromDisk(ctx context.Context, bookID string) error {
	book, err := s.books.GetByID(ctx, bookID)
	if err != nil {
		return err
	}

	if err := s.books.DeleteByID(ctx, bookID); err != nil {
		return err
	}

	if book.ManagedPath != "" {
		_ = os.Remove(book.ManagedPath)
	}
	_ = os.RemoveAll(s.paths.BookCacheDir(bookID))
	return nil
}

func (s *LibraryService) MarkOpened(ctx context.Context, bookID string, when time.Time) error {
	return s.books.UpdateLastOpened(ctx, bookID, when)
}

func (s *LibraryService) UpdateReadingState(ctx context.Context, state domain.ReadingState) error {
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = time.Now().UTC()
	}
	return s.states.Upsert(ctx, state)
}

func (s *LibraryService) ReadingStateForMode(ctx context.Context, bookID string, mode domain.ReadingMode) (domain.ReadingState, error) {
	return s.states.GetByBookAndMode(ctx, bookID, mode)
}

func (s *LibraryService) StatesForBook(ctx context.Context, bookID string) ([]domain.ReadingState, error) {
	return s.states.ListByBook(ctx, bookID)
}

func (s *LibraryService) BookByID(ctx context.Context, bookID string) (domain.Book, error) {
	return s.books.GetByID(ctx, bookID)
}

func (s *LibraryService) MostRecentUnfinishedBook(ctx context.Context) (domain.Book, error) {
	bookID, err := s.states.MostRecentUnfinishedBookID(ctx)
	if err != nil {
		return domain.Book{}, err
	}
	return s.books.GetByID(ctx, bookID)
}

func (s *LibraryService) SetFinished(ctx context.Context, bookID string, finished bool) error {
	now := time.Now().UTC()
	if err := s.states.SetFinishedForBook(ctx, bookID, finished, now); err != nil {
		return err
	}
	return nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
