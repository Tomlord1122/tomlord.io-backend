CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_slug VARCHAR(255) NOT NULL, -- Reference to the blog post
    message TEXT NOT NULL,
    thumb_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX idx_messages_user_id ON messages(user_id);
CREATE INDEX idx_messages_post_slug ON messages(post_slug);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);

-- Create a composite index for efficient queries by post and time
CREATE INDEX idx_messages_post_created ON messages(post_slug, created_at DESC); 