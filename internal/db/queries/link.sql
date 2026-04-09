-- name: GetAllLinks :many
SELECT * FROM links
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: GetLinksCount :one
SELECT count(id) FROM links;


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

-- name: CreateLink :one
INSERT INTO links (original_url,short_name,short_url)
values($1,$2,$3)
RETURNING *;

-- name: DeleteLink :exec
DELETE from links where id =$1;

-- name: UpdateLink :one
UPDATE links set 
    original_url = COALESCE (sqlc.narg(original_url),original_url ),
    short_name = COALESCE(sqlc.narg(short_name),short_name),
    short_url = COALESCE(sqlc.narg(short_url), short_url)
    where id = sqlc.arg(id)
RETURNING *;


 