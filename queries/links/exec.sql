-- name: CreateLink :exec
INSERT INTO links (original_url,short_name )
values($1,$2);

-- name: DeleteLink :exec
DELETE from links where id =$1;

-- name: UpdateLink :one
UPDATE links set(original_url,short_name ,short_url) values($1, $2, $3)
where id = $4
RETURNING *;

 