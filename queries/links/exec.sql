-- name: CreateLink :exec
INSERT INTO links (original_url,short_name ,short_url)
values($1,$2,$3);

 