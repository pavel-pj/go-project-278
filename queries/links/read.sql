-- name: GetAllLinks :many
SELECT * FROM links;

-- name: GetLink :one
SELECT * FROM links where id = ($1);

-- name: GetLinkByOriginUrl :one
SELECT * FROM links where original_url = ($1);

-- name: GetLinkByShortName :one
SELECT * FROM links where short_name = ($1);

-- name: GetLinkByOriginUrlExludedId :one
SELECT * FROM links 
where original_url = ($1)
and id != $2;

-- name: GetLinkByShortNameExcluedeId :one
SELECT * FROM links 
where short_name = ($1)
and id != $2;



 