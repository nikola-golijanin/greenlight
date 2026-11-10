package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(app.RateLimit)
	r.Use(app.Authenticate)

	r.NotFound(app.notFoundResponse)
	r.MethodNotAllowed(app.methodNotAllowedResponse)

	r.Get("/v1/healthcheck", app.healthcheckHandler)

	r.Group(func(r chi.Router) {
		r.Use(app.requireActivatedUser)
		// movies
		r.Route("/v1/movies", func(r chi.Router) {
			r.Get("/", app.requirePermission("movies:read", app.getMoviesHandler))
			r.Post("/", app.requirePermission("movies:write", app.createMovieHandler))
			r.Get("/{id}", app.requirePermission("movies:read", app.getMovieByIdHandler))
			r.Patch("/{id}", app.requirePermission("movies:write", app.updateMovieHandler))
			r.Delete("/{id}", app.requirePermission("movies:write", app.deleteMovieHandler))
		})
	})

	// users
	r.Route("/v1/users", func(r chi.Router) {
		r.Post("/", app.registerUserHandler)
		r.Put("/activated", app.activateUserHandler)
	})

	r.Post("/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	return r
}
