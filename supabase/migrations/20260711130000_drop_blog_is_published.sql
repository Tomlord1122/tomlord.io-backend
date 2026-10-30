-- is_published was never exposed as a draft workflow. Every live row is true.
-- Drop it only after the API stops reading and writing the column.
DROP INDEX IF EXISTS idx_blogs_published;
ALTER TABLE blogs DROP COLUMN IF EXISTS is_published;
