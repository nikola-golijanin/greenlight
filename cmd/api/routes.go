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

	r.NotFound(app.notFoundResponse)
	r.MethodNotAllowed(app.methodNotAllowedResponse)

	r.Get("/v1/healthcheck", app.healthcheckHandler)
	r.Get("/v1/movies", app.getMoviesHandler)
	r.Post("/v1/movies", app.createMovieHandler)
	r.Get("/v1/movies/{id}", app.getMovieByIdHandler)
	r.Patch("/v1/movies/{id}", app.updateMovieHandler)
	r.Delete("/v1/movies/{id}", app.deleteMovieHandler)

	return r
}
