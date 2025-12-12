-- Add content column to blogs table to store full markdown content
ALTER TABLE blogs ADD COLUMN content TEXT;
