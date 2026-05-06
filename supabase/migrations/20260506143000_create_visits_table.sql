CREATE TABLE daily_visits (
    date DATE PRIMARY KEY,
    visit_count INT NOT NULL DEFAULT 0
);

CREATE TABLE visitor_hashes (
    date DATE NOT NULL,
    hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (date, hash)
);

CREATE INDEX idx_visitor_hashes_date ON visitor_hashes(date);
