-- name: GetAllLinks :many
SELECT * FROM links;

-- name: GetLink :one
SELECT * FROM links where id = ($1);

-- name: GetLinkByOriginUrl :one
SELECT * FROM links where original_url = ($1);

-- name: GetLinkByShortName :one
SELECT * FROM links where short_name = ($1);


 