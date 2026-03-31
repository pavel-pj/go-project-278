-- +goose Up
-- +goose StatementBegin
CREATE TABLE link_visits (
    id SERIAL PRIMARY KEY,
    link_id INTEGER REFERENCES links(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    ip VARCHAR(45),
    user_agent TEXT,
    status INTEGER
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS link_visits;
-- +goose StatementEnd

