package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nikola-golijanin/greenlight/internal/data"
)

func TestGetMovieById(t *testing.T) {
	t.Run("returns movie by id", func(t *testing.T) {
		request := WithUrlParam(newRequest(t, http.MethodGet, "/v1/movies/1"), "id", "1")
		response := httptest.NewRecorder()

		app := &application{logger: slog.Default()}
		app.getMovieById(response, request)

		var gotMovie data.Movie
		json.Unmarshal(response.Body.Bytes(), &gotMovie)

		wantedMovie := data.Movie{
			ID:        1,
			CreatedAt: time.Now(),
			Title:     "Movie",
			Runtime:   150,
			Genres:    []string{"drama", "romance"},
			Version:   1,
		}

		if gotMovie.Title != wantedMovie.Title {
			t.Errorf("got %q, want %q", gotMovie.Title, wantedMovie.Title)
		}
	})
	t.Run("fails to parse id", func(t *testing.T) {
		request := WithUrlParam(newRequest(t, http.MethodGet, "/v1/movies/nikola"), "id", "nikola")
		response := httptest.NewRecorder()

		app := &application{logger: slog.Default()}
		app.getMovieById(response, request)

		got := response.Body.String()
		want := "404 page not found\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestCreateMovie(t *testing.T) {
	t.Run("Create movie", func(t *testing.T) {
		request := newRequest(t, http.MethodPost, "/v1/movies")
		response := httptest.NewRecorder()
		app := &application{logger: slog.Default()}
		app.createMovieHandler(response, request)

		got := response.Body.String()
		want := "create a new movie\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func newRequest(t testing.TB, method, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Errorf("error while creating request")
	}
	return req
}

// WithUrlParam returns a pointer to a request object with the given URL params
// added to a new chi.Context object.
func WithUrlParam(r *http.Request, key, value string) *http.Request {
	chiCtx := chi.NewRouteContext()
	req := r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
	chiCtx.URLParams.Add(key, value)
	return req
}
