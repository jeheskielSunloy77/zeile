DROP INDEX IF EXISTS idx_idempotency_keys_user_operation_key;
DROP TABLE IF EXISTS idempotency_keys;

DROP INDEX IF EXISTS idx_highlights_deleted_at;
DROP INDEX IF EXISTS idx_highlights_user_library_book_id;
DROP INDEX IF EXISTS idx_highlights_user_id;
DROP TABLE IF EXISTS highlights;

DROP INDEX IF EXISTS idx_reading_states_deleted_at;
DROP INDEX IF EXISTS idx_reading_states_user_library_book_id;
DROP INDEX IF EXISTS idx_reading_states_user_id;
DROP INDEX IF EXISTS idx_reading_states_user_book_mode_active;
DROP TABLE IF EXISTS reading_states;

DROP INDEX IF EXISTS idx_user_library_books_deleted_at;
DROP INDEX IF EXISTS idx_user_library_books_is_public;
DROP INDEX IF EXISTS idx_user_library_books_source_library_book_id;
DROP INDEX IF EXISTS idx_user_library_books_catalog_book_id;
DROP INDEX IF EXISTS idx_user_library_books_user_id;
DROP INDEX IF EXISTS idx_user_library_books_user_catalog_source_active;
DROP TABLE IF EXISTS user_library_books;

DROP INDEX IF EXISTS idx_book_assets_deleted_at;
DROP INDEX IF EXISTS idx_book_assets_checksum;
DROP INDEX IF EXISTS idx_book_assets_source_asset_id;
DROP INDEX IF EXISTS idx_book_assets_uploader_user_id;
DROP INDEX IF EXISTS idx_book_assets_catalog_book_id;
DROP TABLE IF EXISTS book_assets;

DROP INDEX IF EXISTS idx_books_catalog_deleted_at;
DROP TABLE IF EXISTS books_catalog;
