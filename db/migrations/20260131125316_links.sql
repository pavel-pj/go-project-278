-- +goose Up
-- +goose StatementBegin
CREATE TABLE links(
    id SERIAL PRIMARY KEY  ,
    original_url TEXT NOT NULL UNIQUE,
    short_name TEXT , 
    short_url TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS links;
-- +goose StatementEnd
