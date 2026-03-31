-- name: CreateVisit :one
INSERT INTO link_visits (link_id , ip , user_agent,status)
values($1,$2,$3,$4)
RETURNING *;