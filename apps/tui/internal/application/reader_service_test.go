package application

import (
	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/reader"
	"reflect"
	"testing"
)

func TestComputeSectionTokenStarts(t *testing.T) {
	sections := []string{
		"alpha beta",
		"gamma",
		"delta epsilon zeta",
	}

	starts := computeSectionTokenStarts(sections)
	if len(starts) != 3 {
		t.Fatalf("expected 3 starts, got %d", len(starts))
	}
	if starts[0] != 0 {
		t.Fatalf("expected first section to start at 0, got %d", starts[0])
	}
	if !(starts[0] < starts[1] && starts[1] < starts[2]) {
		t.Fatalf("expected monotonic starts, got %v", starts)
	}
}

func TestComputeChapterTokenStarts(t *testing.T) {
	sections := []string{
		"CHAPTER ONE\nalpha beta\n\ngamma",
		"delta\nCHAPTER TWO\nepsilon zeta",
	}
	sectionChapterLineIndexes := [][]int{
		{0},
		{1},
	}

	starts := computeChapterTokenStarts(sections, sectionChapterLineIndexes)
	expected := []int{0, 12}
	if !reflect.DeepEqual(starts, expected) {
		t.Fatalf("expected chapter starts %v, got %v", expected, starts)
	}
}

func TestLineTokenStartsForSection(t *testing.T) {
	section := "title\n\nbody line"
	starts := lineTokenStartsForSection(section)
	expected := []int{0, 2, 3}
	if !reflect.DeepEqual(starts, expected) {
		t.Fatalf("expected line starts %v, got %v", expected, starts)
	}
}

func TestComputeTokenStyles(t *testing.T) {
	sections := []string{
		"plain bold\nitalic",
		"code mark",
	}
	sectionStyles := [][]domain.InlineStyleSpan{
		{
			{LineIndex: 0, StartWord: 1, EndWord: 2, Style: domain.InlineStyleBold},
			{LineIndex: 1, StartWord: 0, EndWord: 1, Style: domain.InlineStyleItalic},
		},
		{
			{LineIndex: 0, StartWord: 0, EndWord: 1, Style: domain.InlineStyleCode},
			{LineIndex: 0, StartWord: 1, EndWord: 2, Style: domain.InlineStyleMark},
		},
	}

	got := computeTokenStyles(sections, sectionStyles)
	expected := map[int]reader.TextStyle{
		1: reader.TextStyleBold,
		3: reader.TextStyleItalic,
		6: reader.TextStyleCode,
		7: reader.TextStyleMark,
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected styles %v, got %v", expected, got)
	}
}
