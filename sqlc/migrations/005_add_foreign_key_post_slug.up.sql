-- Add foreign key constraint to make post_slug reference blogs.slug
ALTER TABLE messages 
ADD CONSTRAINT fk_messages_post_slug 
FOREIGN KEY (post_slug) REFERENCES blogs(slug) ON DELETE CASCADE;

-- Note: This assumes that all existing post_slug values in messages 
-- correspond to valid blog slugs. If there are orphaned references,
-- this migration will fail and you'll need to clean up the data first. 