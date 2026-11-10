package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   app.config.cors.trustedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Use(middleware.Recoverer)
	r.Use(app.RateLimit)
	r.Use(app.Authenticate)

	r.NotFound(app.notFoundResponse)
	r.MethodNotAllowed(app.methodNotAllowedResponse)

	r.Get("/v1/healthcheck", app.healthcheckHandler)

	r.Group(func(r chi.Router) {
		r.Use(app.RequireActivatedUser)
		// movies
		r.Route("/v1/movies", func(r chi.Router) {
			r.Get("/", app.RequirePermission("movies:read", app.getMoviesHandler))
			r.Post("/", app.RequirePermission("movies:write", app.createMovieHandler))
			r.Get("/{id}", app.RequirePermission("movies:read", app.getMovieByIdHandler))
			r.Patch("/{id}", app.RequirePermission("movies:write", app.updateMovieHandler))
			r.Delete("/{id}", app.RequirePermission("movies:write", app.deleteMovieHandler))
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
