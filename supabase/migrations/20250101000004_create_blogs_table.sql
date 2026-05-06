CREATE TABLE blogs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(500) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL, -- SEO friendly URL slug
    date DATE NOT NULL, -- Publication date
    lang VARCHAR(10) NOT NULL DEFAULT 'zh-tw', -- Language (zh-tw, en, etc.)
    duration VARCHAR(20) NOT NULL DEFAULT '5min', -- Reading duration
    tags TEXT[] DEFAULT '{}', -- Array of tags
    description TEXT, -- Optional meta description for SEO
    is_published BOOLEAN DEFAULT false, -- Whether the blog is published
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX idx_blogs_slug ON blogs(slug);
CREATE INDEX idx_blogs_date ON blogs(date DESC);
CREATE INDEX idx_blogs_published ON blogs(is_published);
CREATE INDEX idx_blogs_lang ON blogs(lang);
CREATE INDEX idx_blogs_tags ON blogs USING GIN(tags); -- GIN index for array search
