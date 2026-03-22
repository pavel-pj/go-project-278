-- name: CreateLink :one
INSERT INTO links (original_url,short_name,short_url)
values($1,$2,$3)
RETURNING *;

-- name: DeleteLink :exec
DELETE from links where id =$1;

-- name: UpdateLink :one
UPDATE links set 
    original_url = COALESCE (sqlc.narg(original_url),original_url ),
    short_name = COALESCE(sqlc.narg(short_name),short_name)
    where id = sqlc.arg(id)
RETURNING *;


 