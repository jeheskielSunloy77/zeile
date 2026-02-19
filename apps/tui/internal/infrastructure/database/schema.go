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
}
