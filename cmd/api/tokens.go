package main

import (
	"errors"
	"net/http"

	"github.com/nikola-golijanin/greenlight/internal/users"
	"github.com/nikola-golijanin/greenlight/internal/validator"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.NewValidator()

	users.ValidateEmail(v, input.Email)
	users.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := app.tokens.Login(input.Email, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		case errors.Is(err, users.ErrInvalidPassword):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err = app.writeJSON(w, http.StatusCreated, token, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
