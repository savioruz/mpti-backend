-- name: CreateField :one
INSERT INTO fields (location_id, name, type, price, description, images)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: GetFieldById :one
SELECT * FROM fields WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetFields :many
SELECT * FROM fields
WHERE deleted_at IS NULL
  AND ($1::text = '' OR name ILIKE '%' || $1 || '%')
    LIMIT $2 OFFSET $3;

-- name: CountFields :one
SELECT COUNT(*) FROM fields
WHERE deleted_at IS NULL
  AND ($1::text = '' OR name ILIKE '%' || $1 || '%');

-- name: GetFieldsByLocationID :many
SELECT * FROM fields
WHERE deleted_at IS NULL
  AND location_id = $2
  AND ($1::text = '' OR name ILIKE '%' || $1 || '%')
    LIMIT $3 OFFSET $4;

-- name: CountFieldsByLocationID :one
SELECT COUNT(*) FROM fields
WHERE deleted_at IS NULL
  AND location_id = $2
  AND ($1::text = '' OR name ILIKE '%' || $1 || '%');

-- name: UpdateField :one
UPDATE fields SET
    location_id = $2,
    name = $3,
    type = $4,
    price = $5,
    description = $6,
    images = $7,
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id;

-- name: DeleteField :exec
DELETE FROM fields WHERE id = $1 AND deleted_at IS NULL;
