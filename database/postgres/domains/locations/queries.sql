-- name: CreateLocation :one
INSERT INTO locations (name, latitude, longitude, description)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetLocationById :one
SELECT * FROM locations WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetLocationsWithFilter :many
SELECT * FROM locations
WHERE deleted_at IS NULL
  AND ($1::text = '' OR name ILIKE '%' || $1 || '%')
ORDER BY created_at DESC
    LIMIT $2 OFFSET $3;

-- name: CountLocationsWithFilter :one
SELECT COUNT(*) FROM locations
WHERE deleted_at IS NULL
  AND ($1::text = '' OR name ILIKE '%' || $1 || '%');

-- name: UpdateLocation :one
UPDATE locations SET name = $1, latitude = $2, longitude = $3, description = $4, updated_at = now()
    WHERE id = $5 AND deleted_at IS NULL RETURNING *;

-- name: DeleteLocation :exec
DELETE FROM locations WHERE id = $1 AND deleted_at IS NULL;
