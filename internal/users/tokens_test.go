package users

import (
	"testing"
	"time"

	"github.com/nikola-golijanin/greenlight/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateToken(t *testing.T) {
	ttl := 24 * time.Hour
	token := generateToken(42, ttl, ScopeAuthentication)

	assert.NotEmpty(t, token.Plaintext)
	assert.Len(t, token.Plaintext, 26, "rand.Text produces 26-char tokens")
	assert.NotEmpty(t, token.Hash)
	assert.NotEqual(t, []byte(token.Plaintext), token.Hash, "hash must differ from plaintext")
	assert.Equal(t, 42, token.UserID)
	assert.Equal(t, ScopeAuthentication, token.Scope)

	want := time.Now().Add(ttl)
	assert.WithinDuration(t, want, token.Expiry, 2*time.Second)
}

func TestTokensAreUnique(t *testing.T) {
	t1 := generateToken(1, time.Hour, ScopeAuthentication)
	t2 := generateToken(1, time.Hour, ScopeAuthentication)
	assert.NotEqual(t, t1.Plaintext, t2.Plaintext)
	assert.NotEqual(t, t1.Hash, t2.Hash)
}

func TestValidateTokenPlaintext(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantErr  bool
	}{
		{"empty", "", true},
		{"too short", "AAAAAAAAAAAAAAAAAAAAAAAAA", true},  // 25 chars
		{"valid (26 chars)", "AAAAAAAAAAAAAAAAAAAAAAAAAA", false}, // 26 chars
		{"too long", "AAAAAAAAAAAAAAAAAAAAAAAAAAA", true}, // 27 chars
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.NewValidator()
			ValidateTokenPlaintext(v, tt.token)
			if tt.wantErr {
				require.False(t, v.Valid())
			} else {
				require.True(t, v.Valid())
			}
		})
	}
}
