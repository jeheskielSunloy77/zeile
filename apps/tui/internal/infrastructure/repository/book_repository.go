package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zeile/tui/internal/domain"
)

type BookRepository struct {
	db *sql.DB
}

func NewBookRepository(db *sql.DB) *BookRepository {
	return &BookRepository{db: db}
}

func (r *BookRepository) Create(ctx context.Context, book domain.Book) error {
	query := `INSERT INTO books (
		id, fingerprint, title, author, format, added_at, last_opened_at,
		source_path, managed_path, metadata_json, size_bytes
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var lastOpened any
	if book.LastOpened != nil {
		lastOpened = book.LastOpened.Unix()
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		book.ID,
		book.Fingerprint,
		book.Title,
		book.Author,
		book.Format,
		book.AddedAt.Unix(),
		lastOpened,
		book.SourcePath,
		book.ManagedPath,
		book.Metadata,
		book.SizeBytes,
	)
	if err != nil {
		return fmt.Errorf("insert book: %w", err)
	}
	return nil
}

func (r *BookRepository) GetByFingerprint(ctx context.Context, fingerprint string) (domain.Book, error) {
	query := `SELECT id, fingerprint, title, author, format, added_at, last_opened_at,
		source_path, managed_path, metadata_json, size_bytes
		FROM books WHERE fingerprint = ? LIMIT 1`
	book, err := scanBook(r.db.QueryRowContext(ctx, query, fingerprint))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Book{}, ErrNotFound
		}
		return domain.Book{}, fmt.Errorf("query book by fingerprint: %w", err)
	}
	return book, nil
}

func (r *BookRepository) GetByID(ctx context.Context, id string) (domain.Book, error) {
	query := `SELECT id, fingerprint, title, author, format, added_at, last_opened_at,
		source_path, managed_path, metadata_json, size_bytes
		FROM books WHERE id = ? LIMIT 1`
	book, err := scanBook(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Book{}, ErrNotFound
		}
		return domain.Book{}, fmt.Errorf("query book by id: %w", err)
	}
	return book, nil
}

func (r *BookRepository) List(ctx context.Context) ([]domain.Book, error) {
	query := `SELECT id, fingerprint, title, author, format, added_at, last_opened_at,
		source_path, managed_path, metadata_json, size_bytes
		FROM books
		ORDER BY COALESCE(last_opened_at, 0) DESC, added_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list books: %w", err)
	}
	defer rows.Close()

	books := make([]domain.Book, 0)
	for rows.Next() {
		book, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate books: %w", err)
	}

	return books, nil
}

func (r *BookRepository) DeleteByID(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM books WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete book: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete book rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *BookRepository) UpdateLastOpened(ctx context.Context, id string, when time.Time) error {
	res, err := r.db.ExecContext(ctx, `UPDATE books SET last_opened_at = ? WHERE id = ?`, when.Unix(), id)
	if err != nil {
		return fmt.Errorf("update last_opened: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update last_opened rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanBook(row scanner) (domain.Book, error) {
	var (
		book          domain.Book
		format        string
		addedAt       int64
		lastOpenedRaw sql.NullInt64
	)

	err := row.Scan(
		&book.ID,
		&book.Fingerprint,
		&book.Title,
		&book.Author,
		&format,
		&addedAt,
		&lastOpenedRaw,
		&book.SourcePath,
		&book.ManagedPath,
		&book.Metadata,
		&book.SizeBytes,
	)
	if err != nil {
		return domain.Book{}, err
	}

	book.Format = domain.BookFormat(format)
	book.AddedAt = time.Unix(addedAt, 0)
	if lastOpenedRaw.Valid {
		lastOpened := time.Unix(lastOpenedRaw.Int64, 0)
		book.LastOpened = &lastOpened
	}
	return book, nil
}
