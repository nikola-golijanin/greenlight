package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikola-golijanin/greenlight/internal/users"
	"github.com/stretchr/testify/assert"
)

// sentinelHandler writes a fixed 200 so tests can tell middleware passed through.
var sentinelHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func newAuthApp() *application {
	return &application{
		logger: slog.Default(),
		config: config{auth: struct{ enabled bool }{enabled: true}},
	}
}

func TestRequireActivatedUser(t *testing.T) {
	app := newAuthApp()

	t.Run("no user in context → 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		app.RequireActivatedUser(sentinelHandler).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("unactivated user → 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = app.contextSetAuthUser(req, &users.User{Activated: false})
		rr := httptest.NewRecorder()
		app.RequireActivatedUser(sentinelHandler).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("activated user → passes through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = app.contextSetAuthUser(req, &users.User{Activated: true})
		rr := httptest.NewRecorder()
		app.RequireActivatedUser(sentinelHandler).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("auth disabled → always passes through", func(t *testing.T) {
		disabledApp := &application{
			logger: slog.Default(),
			config: config{auth: struct{ enabled bool }{enabled: false}},
		}
		// Even with no user in context, should pass through when auth is disabled
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		disabledApp.RequireActivatedUser(sentinelHandler).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
