-- Short posts for the public /thread timeline.
-- Cursor pagination uses (created_at, id), so both stay unique and indexed together.
CREATE TABLE threads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT threads_content_not_blank CHECK (char_length(btrim(content)) > 0),
    CONSTRAINT threads_content_length CHECK (char_length(content) <= 2000)
);

CREATE UNIQUE INDEX idx_threads_created_at_id ON threads (created_at DESC, id DESC);
