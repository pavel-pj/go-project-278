CREATE TABLE links(
    id SERIAL PRIMARY KEY  ,
    original_url TEXT NOT NULL UNIQUE,
    short_name TEXT , 
    short_url TEXT NOT NULL
);