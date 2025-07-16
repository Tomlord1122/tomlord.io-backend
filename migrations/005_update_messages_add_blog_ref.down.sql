-- Remove indexes
DROP INDEX IF EXISTS idx_messages_blog_created;
DROP INDEX IF EXISTS idx_messages_blog_id;

-- Remove blog_id column
ALTER TABLE messages DROP COLUMN IF EXISTS blog_id; 