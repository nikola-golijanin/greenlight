package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestGetMovieById(t *testing.T) {
	tests := []struct {
		name     string
		request  *http.Request
		response *httptest.ResponseRecorder
		want     string
	}{
		{
			name:     "returns movie by id",
			request:  WithUrlParam(newRequest(t, http.MethodGet, "/v1/movies/1"), "id", "1"),
			response: httptest.NewRecorder(),
			want:     "show the details of movie 1\n",
		},
		{
			name:     "fails to parse id",
			request:  WithUrlParam(newRequest(t, http.MethodGet, "/v1/movies/nikola"), "id", "nikola"),
			response: httptest.NewRecorder(),
			want:     "404 page not found\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := &application{logger: slog.Default()}
			app.getMovieById(test.response, test.request)

			got := test.response.Body.String()

			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
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
