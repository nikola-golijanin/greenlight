package validator_test

import (
	"testing"

	"github.com/nikola-golijanin/greenlight/internal/validator"
	"github.com/stretchr/testify/assert"
)

func TestValidatorValid(t *testing.T) {
	v := validator.NewValidator()
	assert.True(t, v.Valid())
}

func TestValidatorCheck(t *testing.T) {
	v := validator.NewValidator()
	v.Check(false, "field", "must be provided")
	assert.False(t, v.Valid())
	assert.Equal(t, "must be provided", v.Errors["field"])
}

func TestValidatorCheckDoesNotOverwrite(t *testing.T) {
	v := validator.NewValidator()
	v.Check(false, "field", "first error")
	v.Check(false, "field", "second error")
	assert.Equal(t, "first error", v.Errors["field"], "first error should not be overwritten")
}

func TestValidatorAddError(t *testing.T) {
	v := validator.NewValidator()
	v.AddError("key", "value")
	assert.Equal(t, "value", v.Errors["key"])
}

func TestValidatorMatches(t *testing.T) {
	assert.True(t, validator.Matches("user@example.com", validator.EmailRX))
	assert.False(t, validator.Matches("not-an-email", validator.EmailRX))
	assert.False(t, validator.Matches("", validator.EmailRX))
}

func TestValidatorUnique(t *testing.T) {
	assert.True(t, validator.Unique([]string{"a", "b", "c"}))
	assert.False(t, validator.Unique([]string{"a", "b", "a"}))
	assert.True(t, validator.Unique([]string{}))
}

func TestValidatorPermittedValue(t *testing.T) {
	assert.True(t, validator.PermittedValue("asc", "asc", "desc"))
	assert.False(t, validator.PermittedValue("random", "asc", "desc"))
}
