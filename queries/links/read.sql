-- name: GetAllLinks :many
SELECT * FROM links;

-- name: GetLink :one
SELECT * FROM links where id = ($1);


 