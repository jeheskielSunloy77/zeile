DROP INDEX IF EXISTS idx_idempotency_keys_user_operation_key;
DROP TABLE IF EXISTS idempotency_keys;

DROP INDEX IF EXISTS idx_moderation_reviews_status;
DROP INDEX IF EXISTS idx_moderation_reviews_catalog_book_id;
DROP TABLE IF EXISTS moderation_reviews;

DROP INDEX IF EXISTS idx_activity_events_created_at;
DROP INDEX IF EXISTS idx_activity_events_user_id;
DROP TABLE IF EXISTS activity_events;

DROP TABLE IF EXISTS community_profiles;

DROP INDEX IF EXISTS idx_share_links_resource;
DROP INDEX IF EXISTS idx_share_links_user_id;
DROP INDEX IF EXISTS idx_share_links_token;
DROP TABLE IF EXISTS share_links;

DROP INDEX IF EXISTS idx_book_share_policies_user_id;
DROP INDEX IF EXISTS idx_book_share_policies_user_book;
DROP TABLE IF EXISTS book_share_policies;

DROP INDEX IF EXISTS idx_share_list_items_highlight_id;
DROP INDEX IF EXISTS idx_share_list_items_book_id;
DROP INDEX IF EXISTS idx_share_list_items_list_id;
DROP INDEX IF EXISTS idx_share_list_items_list_position;
DROP TABLE IF EXISTS share_list_items;

DROP INDEX IF EXISTS idx_highlights_deleted_at;
DROP INDEX IF EXISTS idx_highlights_list_id;
DROP INDEX IF EXISTS idx_highlights_user_library_book_id;
DROP INDEX IF EXISTS idx_highlights_user_id;
DROP TABLE IF EXISTS highlights;

DROP INDEX IF EXISTS idx_share_lists_deleted_at;
DROP INDEX IF EXISTS idx_share_lists_user_id;
DROP TABLE IF EXISTS share_lists;

DROP INDEX IF EXISTS idx_reading_states_deleted_at;
DROP INDEX IF EXISTS idx_reading_states_user_library_book_id;
DROP INDEX IF EXISTS idx_reading_states_user_id;
DROP INDEX IF EXISTS idx_reading_states_user_book_mode_active;
DROP TABLE IF EXISTS reading_states;

DROP INDEX IF EXISTS idx_user_library_books_deleted_at;
DROP INDEX IF EXISTS idx_user_library_books_catalog_book_id;
DROP INDEX IF EXISTS idx_user_library_books_user_id;
DROP INDEX IF EXISTS idx_user_library_books_user_book_active;
DROP TABLE IF EXISTS user_library_books;

DROP INDEX IF EXISTS idx_book_assets_deleted_at;
DROP INDEX IF EXISTS idx_book_assets_checksum;
DROP INDEX IF EXISTS idx_book_assets_uploader_user_id;
DROP INDEX IF EXISTS idx_book_assets_catalog_book_id;
DROP TABLE IF EXISTS book_assets;

DROP INDEX IF EXISTS idx_books_catalog_deleted_at;
DROP INDEX IF EXISTS idx_books_catalog_verification_status;
DROP TABLE IF EXISTS books_catalog;
