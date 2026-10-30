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

-- name: DeleteMovie :exec
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
