package main

import (
	"errors"
	"net/http"

	"github.com/nikola-golijanin/greenlight/internal/users"
	"github.com/nikola-golijanin/greenlight/internal/validator"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &users.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.NewValidator()

	if users.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrDuplicateEmail):
			v.AddError("email", "a user with this email already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusCreated, user, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
