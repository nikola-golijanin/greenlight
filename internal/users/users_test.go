package users

import (
	"testing"

	"github.com/nikola-golijanin/greenlight/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordSet(t *testing.T) {
	var p password
	err := p.Set("pa55word")
	require.NoError(t, err)
	assert.NotNil(t, p.hash)
	assert.NotEqual(t, []byte("pa55word"), p.hash, "hash must not be plaintext")
}

func TestPasswordMatches(t *testing.T) {
	var p password
	require.NoError(t, p.Set("pa55word"))

	match, err := p.Matches("pa55word")
	require.NoError(t, err)
	assert.True(t, match)

	match, err = p.Matches("wrongpassword")
	require.NoError(t, err)
	assert.False(t, match)
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email   string
		wantErr bool
	}{
		{"user@example.com", false},
		{"", true},
		{"not-an-email", true},
		{"missing@tld.", true},
	}

	for _, tt := range tests {
		v := validator.NewValidator()
		ValidateEmail(v, tt.email)
		if tt.wantErr {
			assert.False(t, v.Valid(), "email %q should be invalid", tt.email)
		} else {
			assert.True(t, v.Valid(), "email %q should be valid", tt.email)
		}
	}
}

func TestValidatePasswordPlaintext(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"empty", "", true},
		{"too short (7 bytes)", "1234567", true},
		{"minimum (8 bytes)", "12345678", false},
		{"valid", "pa55word!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.NewValidator()
			ValidatePasswordPlaintext(v, tt.password)
			if tt.wantErr {
				assert.False(t, v.Valid())
			} else {
				assert.True(t, v.Valid())
			}
		})
	}
}

func TestValidateUser(t *testing.T) {
	t.Run("valid user", func(t *testing.T) {
		u := &User{Name: "Alice", Email: "alice@example.com"}
		require.NoError(t, u.Password.Set("pa55word!"))
		v := validator.NewValidator()
		ValidateUser(v, u)
		assert.True(t, v.Valid())
	})

	t.Run("missing name", func(t *testing.T) {
		u := &User{Name: "", Email: "alice@example.com"}
		require.NoError(t, u.Password.Set("pa55word!"))
		v := validator.NewValidator()
		ValidateUser(v, u)
		assert.False(t, v.Valid())
		assert.Contains(t, v.Errors, "name")
	})

	t.Run("invalid email", func(t *testing.T) {
		u := &User{Name: "Alice", Email: "not-an-email"}
		require.NoError(t, u.Password.Set("pa55word!"))
		v := validator.NewValidator()
		ValidateUser(v, u)
		assert.False(t, v.Valid())
		assert.Contains(t, v.Errors, "email")
	})

	t.Run("panics on nil hash", func(t *testing.T) {
		u := &User{Name: "Alice", Email: "alice@example.com"}
		// Password.hash is nil — ValidateUser must panic
		v := validator.NewValidator()
		assert.Panics(t, func() { ValidateUser(v, u) })
	})
}
