-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS link_visits;
CREATE TABLE link_visits (
    id SERIAL PRIMARY KEY,
    link_id INTEGER REFERENCES links(id) ON DELETE CASCADE NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()  NOT NULL, 
    ip VARCHAR(45) NOT NULL,
    user_agent TEXT NOT NULL,
    status INTEGER NOT NULL,
    referer TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS link_visits;
-- +goose StatementEnd

