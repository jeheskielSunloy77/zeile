package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zeile/tui/internal/domain"
)

type SyncRepository struct {
	db *sql.DB
}

func NewSyncRepository(db *sql.DB) *SyncRepository {
	return &SyncRepository{db: db}
}

func (r *SyncRepository) Upsert(ctx context.Context, account domain.SyncAccount) error {
	query := `INSERT INTO sync_accounts (
		user_id, email, username, last_reconciled_at, updated_at
	) VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(user_id) DO UPDATE SET
		email = excluded.email,
		username = excluded.username,
		last_reconciled_at = excluded.last_reconciled_at,
		updated_at = excluded.updated_at`

	var lastReconciled any
	if account.LastReconciledAt != nil {
		lastReconciled = account.LastReconciledAt.UTC().Unix()
	}

	updatedAt := account.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		account.UserID,
		account.Email,
		account.Username,
		lastReconciled,
		updatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("upsert sync account: %w", err)
	}
	return nil
}

func (r *SyncRepository) GetByLocalBookID(ctx context.Context, localBookID string) (domain.SyncBookLink, error) {
	query := `SELECT local_book_id, local_fingerprint, remote_catalog_book_id, remote_library_book_id, updated_at
		FROM sync_book_links
		WHERE local_book_id = ?
		LIMIT 1`

	var (
		link      domain.SyncBookLink
		updatedAt int64
	)

	err := r.db.QueryRowContext(ctx, query, localBookID).Scan(
		&link.LocalBookID,
		&link.LocalFingerprint,
		&link.RemoteCatalogBookID,
		&link.RemoteLibraryBookID,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.SyncBookLink{}, ErrNotFound
		}
		return domain.SyncBookLink{}, fmt.Errorf("get sync link by local book id: %w", err)
	}
	link.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return link, nil
}

func (r *SyncRepository) UpsertBookLink(ctx context.Context, link domain.SyncBookLink) error {
	query := `INSERT INTO sync_book_links (
		local_book_id, local_fingerprint, remote_catalog_book_id, remote_library_book_id, updated_at
	) VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(local_book_id) DO UPDATE SET
		local_fingerprint = excluded.local_fingerprint,
		remote_catalog_book_id = excluded.remote_catalog_book_id,
		remote_library_book_id = excluded.remote_library_book_id,
		updated_at = excluded.updated_at`

	updatedAt := link.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		link.LocalBookID,
		link.LocalFingerprint,
		link.RemoteCatalogBookID,
		link.RemoteLibraryBookID,
		updatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("upsert sync book link: %w", err)
	}
	return nil
}
