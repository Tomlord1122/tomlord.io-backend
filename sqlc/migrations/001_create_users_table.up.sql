-- uuid-ossp if for the uuid_generate_v4()
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- This table will store the user info from Google Oauth:
-- 1. google_id: The unique identifier for the user in Google
-- 2. email: The email of the user
-- 3. name: The name of the user
-- 4. picture_url: The URL of the user's profile picture
-- 5. created_at: The timestamp when the user was created
-- 6. updated_at: The timestamp when the user was updated
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    google_id VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    picture_url TEXT,  
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create index for faster lookups
-- the convention is to use idx_<table_name>_<column_name>
CREATE INDEX idx_users_google_id ON users(google_id);
CREATE INDEX idx_users_email ON users(email); 
