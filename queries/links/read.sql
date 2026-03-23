-- name: GetAllLinks :many
SELECT * FROM links;

-- name: GetLink :one
SELECT * FROM links where id = ($1);

-- name: GetLinkByOriginURL :one
SELECT * FROM links where original_url = ($1);

-- name: GetLinkByShortName :one
SELECT * FROM links where short_name = ($1);

-- name: GetLinkByOriginURLExludedID :one
SELECT * FROM links 
where original_url = ($1)
and id != $2;

-- name: GetLinkByShortNameExcluedeID :one
SELECT * FROM links 
where short_name = ($1)
and id != $2;



 