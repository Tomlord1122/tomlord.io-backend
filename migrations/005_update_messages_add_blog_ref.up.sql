-- Add blog_id column to messages table
ALTER TABLE messages ADD COLUMN blog_id UUID REFERENCES blogs(id) ON DELETE CASCADE;

-- Create index for blog_id
CREATE INDEX idx_messages_blog_id ON messages(blog_id);

-- Create a composite index for efficient queries by blog and time  
CREATE INDEX idx_messages_blog_created ON messages(blog_id, created_at DESC);

-- Note: We keep post_slug for backward compatibility
-- but blog_id will be the primary reference going forward 