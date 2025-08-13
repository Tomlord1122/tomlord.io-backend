-- Remove the foreign key constraint
ALTER TABLE messages DROP CONSTRAINT IF EXISTS fk_messages_post_slug; 