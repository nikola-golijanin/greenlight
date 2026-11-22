package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterUser(t *testing.T) {
	cleanDB(t)

	t.Run("valid registration returns 202 with user body", func(t *testing.T) {
		rr := doRequest(t, http.MethodPost, "/v1/users", jsonBody(t, map[string]any{
			"name":     "Alice",
			"email":    "alice@example.com",
			"password": "pa55word!",
		}), nil)

		assert.Equal(t, http.StatusAccepted, rr.Code)

		var body struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
		assert.Equal(t, "Alice", body.Name)
		assert.Equal(t, "alice@example.com", body.Email)

		// Background goroutine sends welcome email with activation token
		testApp.wg.Wait()
		mail := testMailer.pop(t)
		assert.Equal(t, "alice@example.com", mail.recipient)
		assert.NotEmpty(t, mail.data["activationToken"])
	})

	t.Run("duplicate email returns 422", func(t *testing.T) {
		payload := jsonBody(t, map[string]any{
			"name":     "Bob",
			"email":    "bob@example.com",
			"password": "pa55word!",
		})
		doRequest(t, http.MethodPost, "/v1/users", payload, nil)
		testApp.wg.Wait()
		testMailer.drain()

		payload = jsonBody(t, map[string]any{
			"name":     "Bob2",
			"email":    "bob@example.com",
			"password": "pa55word!",
		})
		rr := doRequest(t, http.MethodPost, "/v1/users", payload, nil)
		assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)

		var body struct {
			Error map[string]string `json:"error"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
		assert.Contains(t, body.Error["email"], "already exists")
	})

	t.Run("missing fields returns 422 with field errors", func(t *testing.T) {
		rr := doRequest(t, http.MethodPost, "/v1/users", jsonBody(t, map[string]any{
			"email": "nodename@example.com",
		}), nil)

		assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)

		var body struct {
			Error map[string]string `json:"error"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
		assert.Contains(t, body.Error, "name")
		assert.Contains(t, body.Error, "password")
	})
}

func TestActivateUser(t *testing.T) {
	cleanDB(t)

	t.Run("valid token activates user and returns 200", func(t *testing.T) {
		activationToken := registerUser(t, "Carol", "carol@example.com", "pa55word!")

		rr := doRequest(t, http.MethodPut, "/v1/users/activated", jsonBody(t, map[string]any{
			"token": activationToken,
		}), nil)

		assert.Equal(t, http.StatusOK, rr.Code)

		var body struct {
			Activated bool `json:"activated"`
		}
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
		assert.True(t, body.Activated)
	})

	t.Run("invalid token returns 422", func(t *testing.T) {
		rr := doRequest(t, http.MethodPut, "/v1/users/activated", jsonBody(t, map[string]any{
			"token": "AAAAAAAAAAAAAAAAAAAAAAAAAA", // valid format, wrong token
		}), nil)

		assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	})

	t.Run("malformed token returns 422", func(t *testing.T) {
		rr := doRequest(t, http.MethodPut, "/v1/users/activated", jsonBody(t, map[string]any{
			"token": "tooshort",
		}), nil)

		assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	})
}
