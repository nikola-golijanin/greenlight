package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	cleanDB(t)

	const email = "dave@example.com"
	const password = "pa55word!"

	activationToken := registerUser(t, "Dave", email, password)
	activateUser(t, activationToken)

	t.Run("valid credentials return 201 with token", func(t *testing.T) {
		rr := doRequest(t, http.MethodPost, "/v1/tokens/authentication", jsonBody(t, map[string]any{
			"email":    email,
			"password": password,
		}), nil)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var body struct {
			Token  string `json:"token"`
			Expiry string `json:"expiry"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
		assert.Len(t, body.Token, 26)
		assert.NotEmpty(t, body.Expiry)
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		rr := doRequest(t, http.MethodPost, "/v1/tokens/authentication", jsonBody(t, map[string]any{
			"email":    email,
			"password": "wrongpassword",
		}), nil)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("unknown email returns 401", func(t *testing.T) {
		rr := doRequest(t, http.MethodPost, "/v1/tokens/authentication", jsonBody(t, map[string]any{
			"email":    "nobody@example.com",
			"password": password,
		}), nil)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("missing fields returns 422", func(t *testing.T) {
		rr := doRequest(t, http.MethodPost, "/v1/tokens/authentication",
			jsonBody(t, map[string]any{}), nil)

		assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	})
}
