package main

import (
	"net/http"

	"github.com/go-chi/httprate"
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
