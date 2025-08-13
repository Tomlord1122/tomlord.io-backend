CREATE TABLE message_thumbs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure a user can only thumb a message once
    UNIQUE(message_id, user_id)
);

-- Create indexes for performance
CREATE INDEX idx_message_thumbs_message_id ON message_thumbs(message_id);
CREATE INDEX idx_message_thumbs_user_id ON message_thumbs(user_id); 