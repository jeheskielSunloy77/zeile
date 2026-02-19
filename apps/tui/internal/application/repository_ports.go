package application

import (
	"context"
	"time"

	"github.com/zeile/tui/internal/domain"
)

type BookRepository interface {
	Create(ctx context.Context, book domain.Book) error
	GetByFingerprint(ctx context.Context, fingerprint string) (domain.Book, error)
	GetByID(ctx context.Context, id string) (domain.Book, error)
	List(ctx context.Context) ([]domain.Book, error)
	DeleteByID(ctx context.Context, id string) error
	UpdateLastOpened(ctx context.Context, id string, when time.Time) error
}

type ReadingStateRepository interface {
	Upsert(ctx context.Context, state domain.ReadingState) error
	GetByBookAndMode(ctx context.Context, bookID string, mode domain.ReadingMode) (domain.ReadingState, error)
	ListByBook(ctx context.Context, bookID string) ([]domain.ReadingState, error)
	SetFinishedForBook(ctx context.Context, bookID string, isFinished bool, updatedAt time.Time) error
	MostRecentUnfinishedBookID(ctx context.Context) (string, error)
}
