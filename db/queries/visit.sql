-- name: CreateVisit :one
INSERT INTO link_visits (link_id , ip , user_agent,status, referer)
values($1,$2,$3,$4,$5)
RETURNING *;

-- name: GetVisits :many
SELECT * FROM link_visits
LIMIT $1 OFFSET $2;

-- name: GetVisitsCount :one
SELECT count(id) FROM link_visits;