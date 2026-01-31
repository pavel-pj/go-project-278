-- name: CreateLink :one
INSERT INTO links (original_url,short_name ,short_url)
values($1,$2,$3)
RETURNING id,original_url,short_name ,short_url;

 