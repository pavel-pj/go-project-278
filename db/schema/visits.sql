CREATE TABLE link_visits (
    id SERIAL PRIMARY KEY,
    link_id INTEGER REFERENCES links(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    ip VARCHAR(45),
    user_agent TEXT,
    status INTEGER,
    referer TEXT
);