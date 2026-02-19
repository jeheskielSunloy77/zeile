package application

import (
	"context"
	"testing"
	"time"

	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/storage"
	"github.com/zeile/tui/internal/preprocessing"
)

type fakeBookRepo struct {
	books []domain.Book
}

func (f *fakeBookRepo) Create(context.Context, domain.Book) error { return nil }
func (f *fakeBookRepo) GetByFingerprint(context.Context, string) (domain.Book, error) {
	return domain.Book{}, nil
}
func (f *fakeBookRepo) GetByID(context.Context, string) (domain.Book, error) {
	return domain.Book{}, nil
}
func (f *fakeBookRepo) List(context.Context) ([]domain.Book, error) { return f.books, nil }
func (f *fakeBookRepo) DeleteByID(context.Context, string) error    { return nil }
func (f *fakeBookRepo) UpdateLastOpened(context.Context, string, time.Time) error {
	return nil
}

type fakeStateRepo struct {
	states map[string][]domain.ReadingState
}

func (f *fakeStateRepo) Upsert(context.Context, domain.ReadingState) error { return nil }
func (f *fakeStateRepo) GetByBookAndMode(context.Context, string, domain.ReadingMode) (domain.ReadingState, error) {
	return domain.ReadingState{}, nil
}
func (f *fakeStateRepo) ListByBook(_ context.Context, bookID string) ([]domain.ReadingState, error) {
	return f.states[bookID], nil
}
func (f *fakeStateRepo) SetFinishedForBook(context.Context, string, bool, time.Time) error {
	return nil
}
func (f *fakeStateRepo) MostRecentUnfinishedBookID(context.Context) (string, error) { return "", nil }

func TestSearchBooksRankingExactPrefixSubstring(t *testing.T) {
	books := []domain.Book{
		{ID: "1", Title: "Go", Author: "A"},
		{ID: "2", Title: "Golang Team Guide", Author: "B"},
		{ID: "3", Title: "A practical guide to go", Author: "C"},
	}

	service := NewLibraryService(
		&fakeBookRepo{books: books},
		&fakeStateRepo{states: map[string][]domain.ReadingState{}},
		preprocessing.NewRegistry(),
		storage.Paths{},
	)

	result, err := service.SearchBooks(context.Background(), "go")
	if err != nil {
		t.Fatalf("search returned error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	if result[0].ID != "1" {
		t.Fatalf("expected exact match first, got %s", result[0].ID)
	}
	if result[1].ID != "2" {
		t.Fatalf("expected prefix match second, got %s", result[1].ID)
	}
	if result[2].ID != "3" {
		t.Fatalf("expected substring match third, got %s", result[2].ID)
	}
}

func TestSearchBooksBoostsRecencyAndUnfinished(t *testing.T) {
	now := time.Now().UTC()
	older := now.Add(-24 * time.Hour)
	books := []domain.Book{
		{ID: "a", Title: "Go patterns", Author: "Author One", LastOpened: &older},
		{ID: "b", Title: "Go patterns", Author: "Author Two", LastOpened: &now},
	}

	stateRepo := &fakeStateRepo{states: map[string][]domain.ReadingState{
		"a": {{BookID: "a", IsFinished: true}},
		"b": {{BookID: "b", IsFinished: false}},
	}}

	service := NewLibraryService(
		&fakeBookRepo{books: books},
		stateRepo,
		preprocessing.NewRegistry(),
		storage.Paths{},
	)

	result, err := service.SearchBooks(context.Background(), "go")
	if err != nil {
		t.Fatalf("search returned error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0].ID != "b" {
		t.Fatalf("expected recently opened unfinished book first, got %s", result[0].ID)
	}
}
