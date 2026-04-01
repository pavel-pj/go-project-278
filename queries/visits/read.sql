-- name: GetVisits :many
SELECT * FROM link_visits
LIMIT $1 OFFSET $2;

-- name: GetVisitsCount :one
SELECT count(id) FROM link_visits;