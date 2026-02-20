package reader

import (
	"strings"
	"testing"
)

func TestSearchTokenOffsets(t *testing.T) {
	doc := NewTextDocument("Go makes systems programming fun. Go keeps tooling simple.")
	offsets := doc.SearchTokenOffsets("Go")
	if len(offsets) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(offsets))
	}
	if offsets[0] == offsets[1] {
		t.Fatalf("expected unique token offsets, got duplicates: %v", offsets)
	}
}

func TestPaginatePreservesAnchorAcrossReflow(t *testing.T) {
	text := strings.Repeat("alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu ", 20)
	doc := NewTextDocument(text)

	wide := doc.Paginate(48, 8)
	if len(wide.Pages) < 3 {
		t.Fatalf("expected at least 3 pages in wide pagination, got %d", len(wide.Pages))
	}

	anchor := wide.OffsetForPage(2)
	narrow := doc.Paginate(24, 8)
	page := narrow.PageForOffset(anchor)

	if page < 1 {
		t.Fatalf("expected anchor to remain in later content after reflow, got page index %d", page)
	}
}

func TestPaginateWithForcedPageStartsMovesChapterToTop(t *testing.T) {
	doc := NewTextDocument("chapter one\nalpha beta gamma\ndelta epsilon\nchapter two\nzeta eta theta")
	offsets := doc.SearchTokenOffsets("chapter two")
	if len(offsets) == 0 {
		t.Fatalf("expected chapter offset")
	}
	forced := map[int]struct{}{offsets[0]: {}}

	pagination := doc.PaginateWithForcedPageStarts(20, 3, forced)
	if len(pagination.Pages) < 2 {
		t.Fatalf("expected at least 2 pages, got %d", len(pagination.Pages))
	}
	secondPage := strings.Split(pagination.Pages[1], "\n")
	if len(secondPage) == 0 || secondPage[0] != "chapter two" {
		t.Fatalf("expected chapter two on top of next page, got %q", pagination.Pages[1])
	}
}

func TestNewTextDocumentWithStylesAppliesTokenStyles(t *testing.T) {
	doc := NewTextDocumentWithStyles("alpha beta gamma", map[int]TextStyle{
		1: TextStyleBold | TextStyleItalic,
	})

	if len(doc.Tokens) < 3 {
		t.Fatalf("expected word tokens, got %d", len(doc.Tokens))
	}
	if doc.Tokens[1].Style != (TextStyleBold | TextStyleItalic) {
		t.Fatalf("expected style on token 1, got %v", doc.Tokens[1].Style)
	}
}
