package database

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS books (
		id TEXT PRIMARY KEY,
		fingerprint TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		author TEXT,
		format TEXT NOT NULL,
		added_at INTEGER NOT NULL,
		last_opened_at INTEGER,
		source_path TEXT,
		managed_path TEXT NOT NULL,
		metadata_json TEXT,
		size_bytes INTEGER NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_books_last_opened ON books(last_opened_at DESC);`,
	`CREATE TABLE IF NOT EXISTS reading_state (
		book_id TEXT NOT NULL,
		mode TEXT NOT NULL,
		locator_json TEXT NOT NULL,
		progress_percent REAL NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL,
		is_finished INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (book_id, mode),
		FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
	);`,
	`CREATE INDEX IF NOT EXISTS idx_reading_state_updated_at ON reading_state(updated_at DESC);`,
	`CREATE TABLE IF NOT EXISTS sync_accounts (
		user_id TEXT PRIMARY KEY,
		email TEXT,
		username TEXT,
		last_reconciled_at INTEGER,
		updated_at INTEGER NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS sync_book_links (
		local_book_id TEXT PRIMARY KEY,
		local_fingerprint TEXT NOT NULL,
		remote_catalog_book_id TEXT NOT NULL,
		remote_library_book_id TEXT NOT NULL,
		updated_at INTEGER NOT NULL,
		FOREIGN KEY (local_book_id) REFERENCES books(id) ON DELETE CASCADE
	);`,
	`CREATE TABLE IF NOT EXISTS sync_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		operation TEXT NOT NULL,
		local_book_id TEXT,
		payload_json TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		retry_count INTEGER NOT NULL DEFAULT 0,
		next_attempt_at INTEGER NOT NULL,
		last_error TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_sync_queue_status_next ON sync_queue(status, next_attempt_at ASC);`,
}
