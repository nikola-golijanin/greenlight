package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
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
		want := "{\n\t\"error\": \"the requested resource could not be found\"\n}\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestCreateMovie(t *testing.T) {
	t.Run("Create movie", func(t *testing.T) {
		input := struct {
			Title   string       `json:"title"`
			Year    int          `json:"year"`
			Runtime data.Runtime `json:"runtime"`
			Genres  []string     `json:"genres"`
		}{
			Title:   "Moana",
			Year:    2016,
			Runtime: 107,
			Genres:  []string{"animation", "adventure"},
		}

		js, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}

		request := httptest.NewRequest(http.MethodPost, "/v1/movies", bytes.NewReader(js))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		app := &application{logger: slog.Default()}
		app.createMovieHandler(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
		}

		got := response.Body.String()
		want := fmt.Sprintf("%v\n", input)
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestCreateMovieValidation(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(body map[string]any)
		wantErrs map[string]string
	}{
		{
			name:     "missing title",
			modify:   func(b map[string]any) { delete(b, "title") },
			wantErrs: map[string]string{"title": "must be provided"},
		},
		{
			name:     "title longer than 500 bytes",
			modify:   func(b map[string]any) { b["title"] = strings.Repeat("a", 501) },
			wantErrs: map[string]string{"title": "must not be more than 500 bytes long"},
		},
		{
			name:     "missing year",
			modify:   func(b map[string]any) { delete(b, "year") },
			wantErrs: map[string]string{"year": "must be provided"},
		},
		{
			name:     "year before 1888",
			modify:   func(b map[string]any) { b["year"] = 1887 },
			wantErrs: map[string]string{"year": "must be greater than 1888"},
		},
		{
			name:     "year in the future",
			modify:   func(b map[string]any) { b["year"] = time.Now().Year() + 1 },
			wantErrs: map[string]string{"year": "must not be in the future"},
		},
		{
			name:     "missing runtime",
			modify:   func(b map[string]any) { delete(b, "runtime") },
			wantErrs: map[string]string{"runtime": "must be provided"},
		},
		{
			name:     "negative runtime",
			modify:   func(b map[string]any) { b["runtime"] = data.Runtime(-10) },
			wantErrs: map[string]string{"runtime": "must be a positive integer"},
		},
		{
			name:     "missing genres",
			modify:   func(b map[string]any) { delete(b, "genres") },
			wantErrs: map[string]string{"genres": "must be provided"},
		},
		{
			name:     "empty genres",
			modify:   func(b map[string]any) { b["genres"] = []string{} },
			wantErrs: map[string]string{"genres": "must contain at least 1 genre"},
		},
		{
			name: "more than 5 genres",
			modify: func(b map[string]any) {
				b["genres"] = []string{"a", "b", "c", "d", "e", "f"}
			},
			wantErrs: map[string]string{"genres": "must not contain more than 5 genres"},
		},
		{
			name:     "duplicate genres",
			modify:   func(b map[string]any) { b["genres"] = []string{"drama", "drama"} },
			wantErrs: map[string]string{"genres": "must not contain duplicate values"},
		},
		{
			name:   "all fields missing reports first error per field",
			modify: func(b map[string]any) { clear(b) },
			wantErrs: map[string]string{
				"title":   "must be provided",
				"year":    "must be provided",
				"runtime": "must be provided",
				"genres":  "must be provided",
			},
		},
	}

	app := &application{logger: slog.Default()}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := validMovieBody()
			tt.modify(body)

			response := postMovie(t, app, body)

			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("got status %d, want %d; body: %s",
					response.Code, http.StatusUnprocessableEntity, response.Body.String())
			}

			var got struct {
				Error map[string]string `json:"error"`
			}
			if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
				t.Fatalf("response is not valid JSON: %v; body: %s", err, response.Body.String())
			}

			if !maps.Equal(got.Error, tt.wantErrs) {
				t.Errorf("got errors %v, want %v", got.Error, tt.wantErrs)
			}
		})
	}
}

func validMovieBody() map[string]any {
	return map[string]any{
		"title":   "Moana",
		"year":    2016,
		"runtime": data.Runtime(107),
		"genres":  []string{"animation", "adventure"},
	}
}

func postMovie(t testing.TB, app *application, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()

	js, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/movies", bytes.NewReader(js))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	app.createMovieHandler(response, request)
	return response
}

func TestCreateMovieErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "XML body",
			body:    `<?xml version="1.0" encoding="UTF-8"?><note><to>Alex</to></note>`,
			wantErr: "body contains badly-formed JSON (at character 1)",
		},
		{
			name:    "malformed JSON (trailing comma)",
			body:    `{"title": "Moana", }`,
			wantErr: "body contains badly-formed JSON (at character 20)",
		},
		{
			name:    "JSON array instead of object",
			body:    `["foo", "bar"]`,
			wantErr: "body contains incorrect JSON type (at character 1)",
		},
		{
			name:    "numeric title",
			body:    `{"title": 123}`,
			wantErr: `body contains incorrect JSON type for field "title"`,
		},
		{
			name:    "empty body",
			body:    "",
			wantErr: "body must not be empty",
		},
	}

	app := &application{logger: slog.Default()}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v1/movies", strings.NewReader(tt.body))
			response := httptest.NewRecorder()

			app.createMovieHandler(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("got status %d, want %d", response.Code, http.StatusBadRequest)
			}

			var got struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
				t.Fatalf("response is not valid JSON: %v; body: %s", err, response.Body.String())
			}

			if got.Error != tt.wantErr {
				t.Errorf("got error %q, want %q", got.Error, tt.wantErr)
			}
		})
	}
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
