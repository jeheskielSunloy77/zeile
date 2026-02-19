package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zeile/tui/internal/domain"
)

type ReadingStateRepository struct {
	db *sql.DB
}

func NewReadingStateRepository(db *sql.DB) *ReadingStateRepository {
	return &ReadingStateRepository{db: db}
}

func (r *ReadingStateRepository) Upsert(ctx context.Context, state domain.ReadingState) error {
	locatorBytes, err := json.Marshal(state.Locator)
	if err != nil {
		return fmt.Errorf("marshal locator: %w", err)
	}

	query := `INSERT INTO reading_state (
		book_id, mode, locator_json, progress_percent, updated_at, is_finished
	) VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(book_id, mode) DO UPDATE SET
		locator_json = excluded.locator_json,
		progress_percent = excluded.progress_percent,
		updated_at = excluded.updated_at,
		is_finished = excluded.is_finished`

	_, err = r.db.ExecContext(
		ctx,
		query,
		state.BookID,
		state.Mode,
		string(locatorBytes),
		state.ProgressPercent,
		state.UpdatedAt.Unix(),
		boolToInt(state.IsFinished),
	)
	if err != nil {
		return fmt.Errorf("upsert reading state: %w", err)
	}
	return nil
}

func (r *ReadingStateRepository) GetByBookAndMode(ctx context.Context, bookID string, mode domain.ReadingMode) (domain.ReadingState, error) {
	query := `SELECT book_id, mode, locator_json, progress_percent, updated_at, is_finished
		FROM reading_state WHERE book_id = ? AND mode = ? LIMIT 1`

	var (
		state       domain.ReadingState
		modeValue   string
		locatorJSON string
		updatedAt   int64
		isFinished  int
	)

	err := r.db.QueryRowContext(ctx, query, bookID, mode).Scan(
		&state.BookID,
		&modeValue,
		&locatorJSON,
		&state.ProgressPercent,
		&updatedAt,
		&isFinished,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ReadingState{}, ErrNotFound
		}
		return domain.ReadingState{}, fmt.Errorf("query reading state: %w", err)
	}

	state.Mode = domain.ReadingMode(modeValue)
	state.UpdatedAt = time.Unix(updatedAt, 0)
	state.IsFinished = isFinished == 1
	if err := json.Unmarshal([]byte(locatorJSON), &state.Locator); err != nil {
		return domain.ReadingState{}, fmt.Errorf("decode locator: %w", err)
	}

	return state, nil
}

func (r *ReadingStateRepository) ListByBook(ctx context.Context, bookID string) ([]domain.ReadingState, error) {
	query := `SELECT book_id, mode, locator_json, progress_percent, updated_at, is_finished
		FROM reading_state WHERE book_id = ?`

	rows, err := r.db.QueryContext(ctx, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("list reading state: %w", err)
	}
	defer rows.Close()

	states := make([]domain.ReadingState, 0)
	for rows.Next() {
		var (
			state       domain.ReadingState
			modeValue   string
			locatorJSON string
			updatedAt   int64
			isFinished  int
		)

		if err := rows.Scan(
			&state.BookID,
			&modeValue,
			&locatorJSON,
			&state.ProgressPercent,
			&updatedAt,
			&isFinished,
		); err != nil {
			return nil, fmt.Errorf("scan reading state: %w", err)
		}

		state.Mode = domain.ReadingMode(modeValue)
		state.UpdatedAt = time.Unix(updatedAt, 0)
		state.IsFinished = isFinished == 1
		if err := json.Unmarshal([]byte(locatorJSON), &state.Locator); err != nil {
			return nil, fmt.Errorf("decode locator: %w", err)
		}

		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reading states: %w", err)
	}

	return states, nil
}

func (r *ReadingStateRepository) SetFinishedForBook(ctx context.Context, bookID string, isFinished bool, updatedAt time.Time) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE reading_state SET is_finished = ?, updated_at = ? WHERE book_id = ?`,
		boolToInt(isFinished),
		updatedAt.Unix(),
		bookID,
	)
	if err != nil {
		return fmt.Errorf("update finished state: %w", err)
	}
	return nil
}

func (r *ReadingStateRepository) MostRecentUnfinishedBookID(ctx context.Context) (string, error) {
	query := `SELECT book_id
		FROM reading_state
		WHERE is_finished = 0
		ORDER BY updated_at DESC
		LIMIT 1`

	var bookID string
	if err := r.db.QueryRowContext(ctx, query).Scan(&bookID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("query most recent unfinished book: %w", err)
	}
	return bookID, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
