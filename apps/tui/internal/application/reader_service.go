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
	ChapterStarts []int
	TokenStyles   map[int]reader.TextStyle
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
	chapterStarts := []int(nil)
	tokenStyles := map[int]reader.TextStyle(nil)
	switch mode {
	case domain.ReadingModeEPUB:
		cache, err := s.readEPUBCache(book.ID)
		if err != nil {
			return TextSession{}, err
		}
		text = strings.Join(cache.Sections, "\n\n")
		sectionStarts = computeSectionTokenStarts(cache.Sections)
		chapterStarts = computeChapterTokenStarts(cache.Sections, cache.SectionChapterLineIndexes)
		tokenStyles = computeTokenStyles(cache.Sections, cache.SectionInlineStyles)
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
		Document:      reader.NewTextDocumentWithStyles(text, tokenStyles),
		SectionStarts: sectionStarts,
		ChapterStarts: chapterStarts,
		TokenStyles:   tokenStyles,
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

func computeChapterTokenStarts(sections []string, sectionChapterLineIndexes [][]int) []int {
	if len(sections) == 0 || len(sectionChapterLineIndexes) == 0 {
		return nil
	}

	starts := make([]int, 0, 16)
	sectionOffset := 0
	for sectionIdx, section := range sections {
		if sectionIdx > 0 {
			sectionOffset += 2
		}

		lineTokenStarts := lineTokenStartsForSection(section)

		if sectionIdx < len(sectionChapterLineIndexes) {
			for _, lineIdx := range sectionChapterLineIndexes[sectionIdx] {
				if lineIdx < 0 || lineIdx >= len(lineTokenStarts) {
					continue
				}
				starts = append(starts, sectionOffset+lineTokenStarts[lineIdx])
			}
		}

		sectionOffset += reader.NewTextDocument(section).TokenCount()
	}

	if len(starts) == 0 {
		return nil
	}
	return starts
}

func lineTokenStartsForSection(section string) []int {
	lineCount := len(strings.Split(section, "\n"))
	if lineCount == 0 {
		return nil
	}

	starts := make([]int, 1, lineCount)
	starts[0] = 0
	document := reader.NewTextDocument(section)
	for tokenIdx, token := range document.Tokens {
		if token.Type == reader.TokenNewline && len(starts) < lineCount {
			starts = append(starts, tokenIdx+1)
		}
	}

	for len(starts) < lineCount {
		starts = append(starts, document.TokenCount()-1)
	}
	return starts
}

func computeTokenStyles(sections []string, sectionInlineStyles [][]domain.InlineStyleSpan) map[int]reader.TextStyle {
	if len(sections) == 0 || len(sectionInlineStyles) == 0 {
		return nil
	}

	styles := make(map[int]reader.TextStyle)
	sectionOffset := 0
	for sectionIdx, section := range sections {
		if sectionIdx > 0 {
			sectionOffset += 2
		}

		lineWordTokenIndexes := sectionLineWordTokenIndexes(section)
		if sectionIdx < len(sectionInlineStyles) {
			for _, span := range sectionInlineStyles[sectionIdx] {
				if span.LineIndex < 0 || span.LineIndex >= len(lineWordTokenIndexes) {
					continue
				}
				lineTokens := lineWordTokenIndexes[span.LineIndex]
				if len(lineTokens) == 0 {
					continue
				}
				startWord := span.StartWord
				if startWord < 0 {
					startWord = 0
				}
				endWord := span.EndWord
				if endWord > len(lineTokens) {
					endWord = len(lineTokens)
				}
				if endWord <= startWord {
					continue
				}
				mappedStyle := mapInlineStyle(span.Style)
				if mappedStyle == 0 {
					continue
				}
				for wordIdx := startWord; wordIdx < endWord; wordIdx++ {
					globalToken := sectionOffset + lineTokens[wordIdx]
					styles[globalToken] |= mappedStyle
				}
			}
		}

		sectionOffset += reader.NewTextDocument(section).TokenCount()
	}

	if len(styles) == 0 {
		return nil
	}
	return styles
}

func mapInlineStyle(style domain.InlineStyle) reader.TextStyle {
	var result reader.TextStyle
	if style&domain.InlineStyleBold != 0 {
		result |= reader.TextStyleBold
	}
	if style&domain.InlineStyleItalic != 0 {
		result |= reader.TextStyleItalic
	}
	if style&domain.InlineStyleUnderline != 0 {
		result |= reader.TextStyleUnderline
	}
	if style&domain.InlineStyleMark != 0 {
		result |= reader.TextStyleMark
	}
	if style&domain.InlineStyleSmall != 0 {
		result |= reader.TextStyleSmall
	}
	if style&domain.InlineStyleSub != 0 {
		result |= reader.TextStyleSub
	}
	if style&domain.InlineStyleSup != 0 {
		result |= reader.TextStyleSup
	}
	if style&domain.InlineStyleCode != 0 {
		result |= reader.TextStyleCode
	}
	return result
}

func sectionLineWordTokenIndexes(section string) [][]int {
	lines := strings.Split(section, "\n")
	indexes := make([][]int, len(lines))
	tokenIdx := 0
	for lineIdx, line := range lines {
		words := strings.Fields(line)
		lineIndexes := make([]int, len(words))
		for wordIdx := range words {
			lineIndexes[wordIdx] = tokenIdx
			tokenIdx++
		}
		indexes[lineIdx] = lineIndexes
		if lineIdx < len(lines)-1 {
			tokenIdx++
		}
	}
	return indexes
}
