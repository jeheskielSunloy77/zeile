CREATE TABLE IF NOT EXISTS books_catalog (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    authors TEXT NOT NULL DEFAULT '',
    identifiers JSONB NOT NULL DEFAULT '{}'::jsonb,
    language TEXT,
    verification_status TEXT NOT NULL DEFAULT 'pending',
    source_type TEXT NOT NULL DEFAULT 'user_upload',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (verification_status IN ('pending', 'verified_public_domain', 'rejected'))
);

CREATE INDEX IF NOT EXISTS idx_books_catalog_verification_status ON books_catalog (verification_status);
CREATE INDEX IF NOT EXISTS idx_books_catalog_deleted_at ON books_catalog (deleted_at);

CREATE TABLE IF NOT EXISTS book_assets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    catalog_book_id UUID NOT NULL REFERENCES books_catalog(id) ON DELETE CASCADE,
    uploader_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
CREATE INDEX IF NOT EXISTS idx_book_assets_checksum ON book_assets (checksum);
CREATE INDEX IF NOT EXISTS idx_book_assets_deleted_at ON book_assets (deleted_at);

CREATE TABLE IF NOT EXISTS user_library_books (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    catalog_book_id UUID NOT NULL REFERENCES books_catalog(id) ON DELETE CASCADE,
    preferred_asset_id UUID REFERENCES book_assets(id) ON DELETE SET NULL,
    state TEXT NOT NULL DEFAULT 'active',
    visibility_in_profile BOOLEAN NOT NULL DEFAULT TRUE,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (state IN ('active', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_library_books_user_book_active
    ON user_library_books (user_id, catalog_book_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_library_books_user_id ON user_library_books (user_id);
CREATE INDEX IF NOT EXISTS idx_user_library_books_catalog_book_id ON user_library_books (catalog_book_id);
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

CREATE TABLE IF NOT EXISTS share_lists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    visibility TEXT NOT NULL DEFAULT 'private',
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (visibility IN ('private', 'authenticated'))
);

CREATE INDEX IF NOT EXISTS idx_share_lists_user_id ON share_lists (user_id);
CREATE INDEX IF NOT EXISTS idx_share_lists_deleted_at ON share_lists (deleted_at);

CREATE TABLE IF NOT EXISTS highlights (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_library_book_id UUID NOT NULL REFERENCES user_library_books(id) ON DELETE CASCADE,
    mode TEXT NOT NULL,
    locator_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    excerpt TEXT,
    visibility TEXT NOT NULL DEFAULT 'private',
    list_id UUID REFERENCES share_lists(id) ON DELETE SET NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (mode IN ('epub', 'pdf_text', 'pdf_layout')),
    CHECK (visibility IN ('private', 'authenticated'))
);

CREATE INDEX IF NOT EXISTS idx_highlights_user_id ON highlights (user_id);
CREATE INDEX IF NOT EXISTS idx_highlights_user_library_book_id ON highlights (user_library_book_id);
CREATE INDEX IF NOT EXISTS idx_highlights_list_id ON highlights (list_id);
CREATE INDEX IF NOT EXISTS idx_highlights_deleted_at ON highlights (deleted_at);

CREATE TABLE IF NOT EXISTS share_list_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    list_id UUID NOT NULL REFERENCES share_lists(id) ON DELETE CASCADE,
    item_type TEXT NOT NULL,
    user_library_book_id UUID REFERENCES user_library_books(id) ON DELETE CASCADE,
    highlight_id UUID REFERENCES highlights(id) ON DELETE CASCADE,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (item_type = 'book' AND user_library_book_id IS NOT NULL AND highlight_id IS NULL)
        OR
        (item_type = 'highlight' AND highlight_id IS NOT NULL AND user_library_book_id IS NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_share_list_items_list_position ON share_list_items (list_id, position);
CREATE INDEX IF NOT EXISTS idx_share_list_items_list_id ON share_list_items (list_id);
CREATE INDEX IF NOT EXISTS idx_share_list_items_book_id ON share_list_items (user_library_book_id);
CREATE INDEX IF NOT EXISTS idx_share_list_items_highlight_id ON share_list_items (highlight_id);

CREATE TABLE IF NOT EXISTS book_share_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_library_book_id UUID NOT NULL REFERENCES user_library_books(id) ON DELETE CASCADE,
    raw_file_sharing TEXT NOT NULL DEFAULT 'private',
    allow_metadata_sharing BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (raw_file_sharing IN ('private', 'public_link'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_book_share_policies_user_book ON book_share_policies (user_id, user_library_book_id);
CREATE INDEX IF NOT EXISTS idx_book_share_policies_user_id ON book_share_policies (user_id);

CREATE TABLE IF NOT EXISTS share_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL,
    resource_id UUID NOT NULL,
    token TEXT NOT NULL,
    requires_auth BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (resource_type IN ('list', 'highlight', 'book_file'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_share_links_token ON share_links (token);
CREATE INDEX IF NOT EXISTS idx_share_links_user_id ON share_links (user_id);
CREATE INDEX IF NOT EXISTS idx_share_links_resource ON share_links (resource_type, resource_id);

CREATE TABLE IF NOT EXISTS community_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT,
    bio TEXT,
    avatar_url TEXT,
    show_reading_activity BOOLEAN NOT NULL DEFAULT TRUE,
    show_highlights BOOLEAN NOT NULL DEFAULT TRUE,
    show_lists BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS activity_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id UUID,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    visibility TEXT NOT NULL DEFAULT 'authenticated',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (visibility IN ('private', 'authenticated'))
);

CREATE INDEX IF NOT EXISTS idx_activity_events_user_id ON activity_events (user_id);
CREATE INDEX IF NOT EXISTS idx_activity_events_created_at ON activity_events (created_at DESC);

CREATE TABLE IF NOT EXISTS moderation_reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    catalog_book_id UUID NOT NULL REFERENCES books_catalog(id) ON DELETE CASCADE,
    submitted_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    decision TEXT,
    evidence_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    reviewer_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (status IN ('pending', 'approved', 'rejected')),
    CHECK (decision IS NULL OR decision IN ('approved', 'rejected'))
);

CREATE INDEX IF NOT EXISTS idx_moderation_reviews_catalog_book_id ON moderation_reviews (catalog_book_id);
CREATE INDEX IF NOT EXISTS idx_moderation_reviews_status ON moderation_reviews (status);

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
