CREATE TABLE IF NOT EXISTS books_catalog (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    authors TEXT NOT NULL DEFAULT '',
    identifiers JSONB NOT NULL DEFAULT '{}'::jsonb,
    language TEXT,
    source_type TEXT NOT NULL DEFAULT 'user_upload',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_books_catalog_deleted_at ON books_catalog (deleted_at);

CREATE TABLE IF NOT EXISTS book_assets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    catalog_book_id UUID NOT NULL REFERENCES books_catalog(id) ON DELETE CASCADE,
    uploader_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_asset_id UUID REFERENCES book_assets(id) ON DELETE SET NULL,
    storage_path TEXT NOT NULL,
    public_url TEXT,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    checksum TEXT NOT NULL,
    ingest_status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (size_bytes >= 0),
    CHECK (ingest_status IN ('pending', 'completed', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_book_assets_catalog_book_id ON book_assets (catalog_book_id);
CREATE INDEX IF NOT EXISTS idx_book_assets_uploader_user_id ON book_assets (uploader_user_id);
CREATE INDEX IF NOT EXISTS idx_book_assets_source_asset_id ON book_assets (source_asset_id);
CREATE INDEX IF NOT EXISTS idx_book_assets_checksum ON book_assets (checksum);
CREATE INDEX IF NOT EXISTS idx_book_assets_deleted_at ON book_assets (deleted_at);

CREATE TABLE IF NOT EXISTS user_library_books (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    catalog_book_id UUID NOT NULL REFERENCES books_catalog(id) ON DELETE CASCADE,
    preferred_asset_id UUID REFERENCES book_assets(id) ON DELETE SET NULL,
    source_library_book_id UUID REFERENCES user_library_books(id) ON DELETE SET NULL,
    state TEXT NOT NULL DEFAULT 'active',
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (state IN ('active', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_library_books_user_catalog_source_active
    ON user_library_books (
        user_id,
        catalog_book_id,
        COALESCE(source_library_book_id, '00000000-0000-0000-0000-000000000000'::uuid)
    )
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_library_books_user_id ON user_library_books (user_id);
CREATE INDEX IF NOT EXISTS idx_user_library_books_catalog_book_id ON user_library_books (catalog_book_id);
CREATE INDEX IF NOT EXISTS idx_user_library_books_source_library_book_id ON user_library_books (source_library_book_id);
CREATE INDEX IF NOT EXISTS idx_user_library_books_is_public ON user_library_books (is_public);
CREATE INDEX IF NOT EXISTS idx_user_library_books_deleted_at ON user_library_books (deleted_at);

CREATE TABLE IF NOT EXISTS reading_states (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_library_book_id UUID NOT NULL REFERENCES user_library_books(id) ON DELETE CASCADE,
    mode TEXT NOT NULL,
    locator_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    progress_percent NUMERIC(5,2) NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (mode IN ('epub', 'pdf_text', 'pdf_layout')),
    CHECK (progress_percent >= 0 AND progress_percent <= 100),
    CHECK (version >= 1)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_reading_states_user_book_mode_active
    ON reading_states (user_id, user_library_book_id, mode)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_reading_states_user_id ON reading_states (user_id);
CREATE INDEX IF NOT EXISTS idx_reading_states_user_library_book_id ON reading_states (user_library_book_id);
CREATE INDEX IF NOT EXISTS idx_reading_states_deleted_at ON reading_states (deleted_at);

CREATE TABLE IF NOT EXISTS highlights (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_library_book_id UUID NOT NULL REFERENCES user_library_books(id) ON DELETE CASCADE,
    mode TEXT NOT NULL,
    locator_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    excerpt TEXT,
    visibility TEXT NOT NULL DEFAULT 'private',
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (mode IN ('epub', 'pdf_text', 'pdf_layout')),
    CHECK (visibility IN ('private', 'authenticated'))
);

CREATE INDEX IF NOT EXISTS idx_highlights_user_id ON highlights (user_id);
CREATE INDEX IF NOT EXISTS idx_highlights_user_library_book_id ON highlights (user_library_book_id);
CREATE INDEX IF NOT EXISTS idx_highlights_deleted_at ON highlights (deleted_at);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation TEXT NOT NULL,
    key TEXT NOT NULL,
    request_hash TEXT,
    response_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_idempotency_keys_user_operation_key ON idempotency_keys (user_id, operation, key);
