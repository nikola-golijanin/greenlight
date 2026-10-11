package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/nikola-golijanin/greenlight/internal/movies"
	"github.com/nikola-golijanin/greenlight/internal/validator"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string         `json:"title"`
		Year    int            `json:"year"`
		Runtime movies.Runtime `json:"runtime"`
		Genres  []string       `json:"genres"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	movie := &movies.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}

	v := validator.NewValidator()

	if movies.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.movies.Insert(movie)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	err = app.writeJSON(w, http.StatusCreated, movie, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getMovieByIdHandler(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.movies.GetById(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			app.notFoundResponse(w, r)
			return
		}
		app.badRequestResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, movie, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.movies.GetById(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			app.notFoundResponse(w, r)
			return
		}
		app.badRequestResponse(w, r, err)
		return
	}

	var input struct {
		Title   string         `json:"title"`
		Year    int            `json:"year"`
		Runtime movies.Runtime `json:"runtime"`
		Genres  []string       `json:"genres"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	movie.Title = input.Title
	movie.Year = input.Year
	movie.Runtime = input.Runtime
	movie.Genres = input.Genres

	v := validator.NewValidator()
	if movies.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.movies.Update(movie)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Write the updated movie record in a JSON response.
	err = app.writeJSON(w, http.StatusOK, movie, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.movies.GetById(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			app.notFoundResponse(w, r)
			return
		}
		app.badRequestResponse(w, r, err)
		return
	}

	err = app.movies.Delete(movie.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
