-- name: InsertMovie :one 
INSERT INTO movies (title, year, runtime, genres)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at, version;

-- name: GetMovieById :one
SELECT id, created_at, title, year, runtime, genres, version
FROM movies
WHERE id = $1;

-- name: UpdateMovie :one
UPDATE movies
SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
WHERE id = $5 AND version = $6
RETURNING version;

-- name: DeleteMovie :execrows
DELETE from movies
WHERE id = $1;

-- name: GetAllMovies :many
SELECT id, created_at, title, year, runtime, genres, version
FROM movies
ORDER BY id;

-- name: InsertUser :one
INSERT INTO users (name, email, password_hash, activated)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at, version;

-- name: GetUserByEmail :one
SELECT id, created_at, name, email, password_hash, activated, version
FROM users
WHERE email = $1;

-- name: UpdateUser :one
UPDATE users
SET name = $1, email = $2, password_hash = $3, activated = $4, version = version + 1
WHERE id = $5 AND version = $6
RETURNING version;

-- name: GetUserForToken :one
SELECT users.id, users.created_at, users.name, users.email, users.password_hash, users.activated, users.version
FROM users
INNER JOIN tokens
ON users.id = tokens.user_id
WHERE tokens.hash = $1
AND tokens.scope = $2
AND tokens.expiry > $3;

-- name: InsertToken :exec
INSERT INTO tokens (hash, user_id, expiry, scope)
VALUES ($1, $2, $3, $4);

-- name: DeleteAllTokensForUser :exec
DELETE FROM tokens
WHERE scope = $1 AND user_id = $2;

-- name: GetUsersPermissions :many
SELECT permissions.code 
FROM permissions 
INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
INNER JOIN users ON users_permissions.user_id = users.id
WHERE users.id = $1;

-- name: AddPermissionForUser :exec
INSERT INTO users_permissions (user_id, permission_id)
SELECT sqlc.arg(user_id)::bigint, permissions.id
FROM permissions
WHERE permissions.code = ANY(sqlc.arg(codes)::text[]);
