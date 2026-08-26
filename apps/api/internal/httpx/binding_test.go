package httpx

import (
	"encoding/json"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sample struct {
	Email string `json:"email" validate:"required,email"`
}

func TestBindingError_ValidationFailureIs422(t *testing.T) {
	v := validator.New()
	err := v.Struct(sample{Email: "not-an-email"})
	require.Error(t, err)

	apiErr := BindingError(err)
	assert.Equal(t, ErrValidation, apiErr.Code)
	assert.Equal(t, 422, statusByCode[apiErr.Code])
	require.Len(t, apiErr.Details, 1)
	assert.Equal(t, "email", apiErr.Details[0].Field)
}

func TestBindingError_MalformedJSONIs400(t *testing.T) {
	var target sample
	err := json.Unmarshal([]byte(`{"email": `), &target) // truncated JSON
	require.Error(t, err)

	apiErr := BindingError(err)
	assert.Equal(t, ErrBadRequest, apiErr.Code)
	assert.Equal(t, 400, statusByCode[apiErr.Code])
}

func TestBindingError_WrongTypeIs400(t *testing.T) {
	var target struct {
		Age int `json:"age"`
	}
	err := json.Unmarshal([]byte(`{"age": "not a number"}`), &target)
	require.Error(t, err)

	apiErr := BindingError(err)
	assert.Equal(t, ErrBadRequest, apiErr.Code)
}

func TestBadPathParam_Is400(t *testing.T) {
	apiErr := BadPathParam("id", "must be a positive integer")
	assert.Equal(t, ErrBadRequest, apiErr.Code)
	assert.Equal(t, 400, statusByCode[apiErr.Code])
	assert.Equal(t, "id", apiErr.Details[0].Field)
}
