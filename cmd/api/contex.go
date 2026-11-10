package main

import (
	"context"
	"net/http"

	"github.com/nikola-golijanin/greenlight/internal/users"
)

type contextKey string

const authenticatedUserContextKey = contextKey("authenticatedUser")

func (app *application) contextSetAuthUser(r *http.Request, user *users.User) *http.Request {
	ctx := context.WithValue(r.Context(), authenticatedUserContextKey, user)
	return r.WithContext(ctx)
}

func (app *application) contextGetAuthUser(r *http.Request) (*users.User, bool) {
	user, ok := r.Context().Value(authenticatedUserContextKey).(*users.User)
	return user, ok
}
