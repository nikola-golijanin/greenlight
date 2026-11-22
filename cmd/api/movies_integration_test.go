package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMovieUser registers, activates, and logs in a fresh user with both
// movies:read and movies:write permissions, returning the bearer token.
func setupMovieUser(t *testing.T) string {
	t.Helper()
	email := fmt.Sprintf("movies_%s@example.com", t.Name())
	token := registerUser(t, "MovieUser", email, "pa55word!")
	activateUser(t, token)

	// Retrieve the user's ID to grant write permission.
	var userID int
	err := testDB.QueryRow(context.Background(),
		`SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
	require.NoError(t, err)
	grantWritePermission(t, userID)

	return loginUser(t, email, "pa55word!")
}

func TestGetMovies(t *testing.T) {
	cleanDB(t)

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		rr := doRequest(t, http.MethodGet, "/v1/movies", nil, nil)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("authenticated user with read permission returns 200", func(t *testing.T) {
		authToken := setupMovieUser(t)
		rr := doRequest(t, http.MethodGet, "/v1/movies", nil, bearerHeader(authToken))
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestCreateMovie(t *testing.T) {
	cleanDB(t)

	t.Run("creates movie and returns 201 with Location header", func(t *testing.T) {
		authToken := setupMovieUser(t)
		rr := doRequest(t, http.MethodPost, "/v1/movies", jsonBody(t, map[string]any{
			"title":   "Inception",
			"year":    2010,
			"runtime": "148 mins",
			"genres":  []string{"sci-fi", "thriller"},
		}), bearerHeader(authToken))

		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.NotEmpty(t, rr.Header().Get("Location"))

		var body struct {
			ID      int    `json:"id"`
			Title   string `json:"title"`
			Version int    `json:"version"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
		assert.Equal(t, "Inception", body.Title)
		assert.Greater(t, body.ID, 0)
		assert.Equal(t, 1, body.Version)
	})

	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		rr := doRequest(t, http.MethodPost, "/v1/movies", jsonBody(t, map[string]any{
			"title":   "Inception",
			"year":    2010,
			"runtime": "148 mins",
			"genres":  []string{"sci-fi"},
		}), nil)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestGetMovieByID(t *testing.T) {
	cleanDB(t)

	authToken := setupMovieUser(t)

	// Create a movie to retrieve.
	createRR := doRequest(t, http.MethodPost, "/v1/movies", jsonBody(t, map[string]any{
		"title":   "The Matrix",
		"year":    1999,
		"runtime": "136 mins",
		"genres":  []string{"action", "sci-fi"},
	}), bearerHeader(authToken))
	require.Equal(t, http.StatusCreated, createRR.Code)

	var created struct {
		ID int `json:"id"`
	}
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))

	t.Run("existing id returns 200 with movie", func(t *testing.T) {
		rr := doRequest(t, http.MethodGet, fmt.Sprintf("/v1/movies/%d", created.ID),
			nil, bearerHeader(authToken))

		assert.Equal(t, http.StatusOK, rr.Code)

		var body struct {
			Title string `json:"title"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
		assert.Equal(t, "The Matrix", body.Title)
	})

	t.Run("non-existent id returns 404", func(t *testing.T) {
		rr := doRequest(t, http.MethodGet, "/v1/movies/9999999",
			nil, bearerHeader(authToken))
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestUpdateMovie(t *testing.T) {
	cleanDB(t)

	authToken := setupMovieUser(t)

	createRR := doRequest(t, http.MethodPost, "/v1/movies", jsonBody(t, map[string]any{
		"title":   "Interstellar",
		"year":    2014,
		"runtime": "169 mins",
		"genres":  []string{"sci-fi", "drama"},
	}), bearerHeader(authToken))
	require.Equal(t, http.StatusCreated, createRR.Code)

	var created struct {
		ID      int `json:"id"`
		Version int `json:"version"`
	}
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))

	t.Run("partial update returns 200 with updated fields", func(t *testing.T) {
		rr := doRequest(t, http.MethodPatch, fmt.Sprintf("/v1/movies/%d", created.ID),
			jsonBody(t, map[string]any{"title": "Interstellar (Director's Cut)"}),
			bearerHeader(authToken))

		assert.Equal(t, http.StatusOK, rr.Code)

		var body struct {
			Title   string `json:"title"`
			Version int    `json:"version"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
		assert.Equal(t, "Interstellar (Director's Cut)", body.Title)
		assert.Equal(t, created.Version+1, body.Version)
	})
}

func TestDeleteMovie(t *testing.T) {
	cleanDB(t)

	authToken := setupMovieUser(t)

	createRR := doRequest(t, http.MethodPost, "/v1/movies", jsonBody(t, map[string]any{
		"title":   "Parasite",
		"year":    2019,
		"runtime": "132 mins",
		"genres":  []string{"thriller", "drama"},
	}), bearerHeader(authToken))
	require.Equal(t, http.StatusCreated, createRR.Code)

	var created struct {
		ID int `json:"id"`
	}
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))

	t.Run("deletes existing movie and returns 204", func(t *testing.T) {
		rr := doRequest(t, http.MethodDelete, fmt.Sprintf("/v1/movies/%d", created.ID),
			nil, bearerHeader(authToken))
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("deleting already-deleted movie returns 404", func(t *testing.T) {
		rr := doRequest(t, http.MethodDelete, fmt.Sprintf("/v1/movies/%d", created.ID),
			nil, bearerHeader(authToken))
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestAuthenticateMiddleware(t *testing.T) {
	cleanDB(t)

	t.Run("malformed Authorization header returns 401", func(t *testing.T) {
		rr := doRequest(t, http.MethodGet, "/v1/movies", nil, map[string]string{
			"Authorization": "NotBearer token",
		})
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("invalid token format returns 401", func(t *testing.T) {
		rr := doRequest(t, http.MethodGet, "/v1/movies", nil, map[string]string{
			"Authorization": "Bearer tooshort",
		})
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("unknown valid-format token returns 401", func(t *testing.T) {
		rr := doRequest(t, http.MethodGet, "/v1/movies", nil,
			bearerHeader("AAAAAAAAAAAAAAAAAAAAAAAAAA")) // valid format, not in DB
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestRequirePermission(t *testing.T) {
	cleanDB(t)

	t.Run("user without write permission returns 403 on POST /v1/movies", func(t *testing.T) {
		// Register and activate but do NOT grant write permission.
		actToken := registerUser(t, "ReadOnly", "readonly@example.com", "pa55word!")
		activateUser(t, actToken)
		authToken := loginUser(t, "readonly@example.com", "pa55word!")

		rr := doRequest(t, http.MethodPost, "/v1/movies", jsonBody(t, map[string]any{
			"title":   "Test",
			"year":    2020,
			"runtime": "90 mins",
			"genres":  []string{"drama"},
		}), bearerHeader(authToken))

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}
