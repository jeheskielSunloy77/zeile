package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/repository"
	"github.com/zeile/tui/internal/infrastructure/storage"
	"github.com/zeile/tui/internal/reader"
)

type TextSession struct {
	Book          domain.Book
	Mode          domain.ReadingMode
	Document      reader.TextDocument
	SectionStarts []int
	State         domain.ReadingState
}

type LayoutSession struct {
	Book  domain.Book
	Pages []string
	State domain.ReadingState
}

type ReaderService struct {
	books  BookRepository
	states ReadingStateRepository
	paths  storage.Paths
}

func NewReaderService(books BookRepository, states ReadingStateRepository, paths storage.Paths) *ReaderService {
	return &ReaderService{books: books, states: states, paths: paths}
}

func (s *ReaderService) LoadTextSession(ctx context.Context, bookID string, mode domain.ReadingMode) (TextSession, error) {
	book, err := s.books.GetByID(ctx, bookID)
	if err != nil {
		return TextSession{}, err
	}

	var text string
	sectionStarts := []int(nil)
	switch mode {
	case domain.ReadingModeEPUB:
		cache, err := s.readEPUBCache(book.ID)
		if err != nil {
			return TextSession{}, err
		}
		text = strings.Join(cache.Sections, "\n\n")
		sectionStarts = computeSectionTokenStarts(cache.Sections)
	case domain.ReadingModePDFText:
		cache, err := s.readPDFCache(book.ID)
		if err != nil {
			return TextSession{}, err
		}
		text = strings.Join(cache.Pages, "\n\n")
	default:
		return TextSession{}, fmt.Errorf("mode %s is not a text mode", mode)
	}

	state, err := s.getOrInitState(ctx, book.ID, mode)
	if err != nil {
		return TextSession{}, err
	}

	return TextSession{
		Book:          book,
		Mode:          mode,
		Document:      reader.NewTextDocument(text),
		SectionStarts: sectionStarts,
		State:         state,
	}, nil
}

func (s *ReaderService) LoadLayoutSession(ctx context.Context, bookID string) (LayoutSession, error) {
	book, err := s.books.GetByID(ctx, bookID)
	if err != nil {
		return LayoutSession{}, err
	}
	if book.Format != domain.BookFormatPDF {
		return LayoutSession{}, fmt.Errorf("layout mode is only available for PDF books")
	}

	cache, err := s.readPDFCache(book.ID)
	if err != nil {
		return LayoutSession{}, err
	}
	pages := cache.LayoutPages
	if len(pages) == 0 {
		pages = cache.Pages
	}

	state, err := s.getOrInitState(ctx, book.ID, domain.ReadingModePDFLayout)
	if err != nil {
		return LayoutSession{}, err
	}

	return LayoutSession{
		Book:  book,
		Pages: pages,
		State: state,
	}, nil
}

func (s *ReaderService) SaveState(ctx context.Context, state domain.ReadingState) error {
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = time.Now().UTC()
	}
	if err := s.states.Upsert(ctx, state); err != nil {
		return err
	}
	if err := s.books.UpdateLastOpened(ctx, state.BookID, state.UpdatedAt); err != nil {
		return err
	}
	return nil
}

func (s *ReaderService) SetFinished(ctx context.Context, bookID string, finished bool) error {
	return s.states.SetFinishedForBook(ctx, bookID, finished, time.Now().UTC())
}

func (s *ReaderService) readEPUBCache(bookID string) (domain.EPUBCache, error) {
	cachePath := filepath.Join(s.paths.BookCacheDir(bookID), "epub_cache.json")
	content, err := os.ReadFile(cachePath)
	if err != nil {
		return domain.EPUBCache{}, fmt.Errorf("read epub cache: %w", err)
	}

	var cache domain.EPUBCache
	if err := json.Unmarshal(content, &cache); err != nil {
		return domain.EPUBCache{}, fmt.Errorf("decode epub cache: %w", err)
	}
	if len(cache.Sections) == 0 {
		return domain.EPUBCache{}, fmt.Errorf("epub cache has no sections")
	}
	return cache, nil
}

func (s *ReaderService) readPDFCache(bookID string) (domain.PDFCache, error) {
	cachePath := filepath.Join(s.paths.BookCacheDir(bookID), "pdf_cache.json")
	content, err := os.ReadFile(cachePath)
	if err != nil {
		return domain.PDFCache{}, fmt.Errorf("read pdf cache: %w", err)
	}

	var cache domain.PDFCache
	if err := json.Unmarshal(content, &cache); err != nil {
		return domain.PDFCache{}, fmt.Errorf("decode pdf cache: %w", err)
	}
	if len(cache.Pages) == 0 {
		return domain.PDFCache{}, fmt.Errorf("pdf cache has no pages")
	}
	return cache, nil
}

func (s *ReaderService) getOrInitState(ctx context.Context, bookID string, mode domain.ReadingMode) (domain.ReadingState, error) {
	state, err := s.states.GetByBookAndMode(ctx, bookID, mode)
	if err == nil {
		return state, nil
	}
	if err != nil && !errorsIsNotFound(err) {
		return domain.ReadingState{}, err
	}

	state = domain.ReadingState{
		BookID:          bookID,
		Mode:            mode,
		Locator:         domain.Locator{Offset: 0, PageIndex: 0},
		ProgressPercent: 0,
		UpdatedAt:       time.Now().UTC(),
		IsFinished:      false,
	}
	if err := s.states.Upsert(ctx, state); err != nil {
		return domain.ReadingState{}, err
	}
	return state, nil
}

func errorsIsNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound)
}

func computeSectionTokenStarts(sections []string) []int {
	if len(sections) == 0 {
		return nil
	}

	starts := make([]int, len(sections))
	offset := 0
	for idx, section := range sections {
		starts[idx] = offset
		offset += reader.NewTextDocument(section).TokenCount()
		if idx < len(sections)-1 {
			offset += 2
		}
	}
	return starts
}
