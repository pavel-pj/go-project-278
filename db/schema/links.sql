CREATE TABLE links(
    id SERIAL PRIMARY KEY  ,
    original_url TEXT NOT NULL UNIQUE,
    short_name TEXT NOT NULL UNIQUE,
    short_url TEXT NOT NULL UNIQUE
);