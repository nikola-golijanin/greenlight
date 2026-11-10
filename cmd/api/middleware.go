package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/httprate"
	"github.com/nikola-golijanin/greenlight/internal/users"
	"github.com/nikola-golijanin/greenlight/internal/validator"
	"github.com/tomasen/realip"
)

func (app *application) RateLimit(next http.Handler) http.Handler {
	if !app.config.limiter.enabled {
		return next
	}

	return httprate.LimitBy(
		app.config.limiter.requests,
		app.config.limiter.window,
		clientIPKey,
		httprate.WithLimitHandler(app.rateLimitExceededResponse),
	)(next)
}

func clientIPKey(r *http.Request) (string, error) {
	return httprate.CanonicalizeIP(realip.FromRequest(r)), nil
}

func (app *application) Authenticate(next http.Handler) http.Handler {
	if !app.config.auth.enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		token := headerParts[1]
		v := validator.NewValidator()
		if users.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		user, err := app.users.GetUserForToken(users.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, users.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		r = app.contextSetAuthUser(r, user)
		next.ServeHTTP(w, r)
	})
}

func (app *application) requireActivatedUser(next http.Handler) http.Handler {
	if !app.config.auth.enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticatedUser, found := app.contextGetAuthUser(r)
		if !found {
			app.authenticationRequiredResponse(w, r)
			return
		}

		if !authenticatedUser.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	if !app.config.auth.enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticatedUser, found := app.contextGetAuthUser(r)

		if !found {
			app.authenticationRequiredResponse(w, r)
			return
		}

		if !authenticatedUser.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		permissions, err := app.permissions.GetUsersPermissions(authenticatedUser.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		if !permissions.Contains(code) {
			app.missingPermissionResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
