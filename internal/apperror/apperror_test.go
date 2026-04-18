package apperror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/apperror"
)

func TestStatusOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code   apperror.Code
		status int
	}{
		{apperror.CodeValidation, http.StatusBadRequest},
		{apperror.CodeNotFound, http.StatusNotFound},
		{apperror.CodeConflict, http.StatusConflict},
		{apperror.CodeInternal, http.StatusInternalServerError},
		{apperror.Code("UNKNOWN"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(string(tc.code), func(t *testing.T) {
			t.Parallel()
			e := &apperror.Error{Code: tc.code}
			require.Equal(t, tc.status, apperror.StatusOf(e))
		})
	}
}

// TestError_Unwrap — Unwrap 체인이 errors.Is 와 errors.As 에 정상 작동.
func TestError_Unwrap(t *testing.T) {
	t.Parallel()

	root := errors.New("root cause")
	e := apperror.Internal("wrapped", root)

	// errors.Is 로 root 를 찾을 수 있어야 함.
	require.ErrorIs(t, e, root)

	// errors.As 로 *apperror.Error 타입 추출.
	var appErr *apperror.Error
	require.True(t, errors.As(e, &appErr))
	require.Equal(t, apperror.CodeInternal, appErr.Code)
}

func TestError_ErrorString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     *apperror.Error
		wantHas string
	}{
		{
			name:    "NotFound with id",
			err:     apperror.NotFound("profile", "p1"),
			wantHas: "NOT_FOUND: profile not found: p1",
		},
		{
			name:    "Validation no cause",
			err:     apperror.Validation("invalid input", nil),
			wantHas: "VALIDATION_FAILED: invalid input",
		},
		{
			name:    "Internal with cause includes cause",
			err:     apperror.Internal("db failed", fmt.Errorf("connection refused")),
			wantHas: "cause: connection refused",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Contains(t, tc.err.Error(), tc.wantHas)
		})
	}
}

func TestValidation_FieldsPreserved(t *testing.T) {
	t.Parallel()
	fields := []apperror.FieldError{
		{Field: "id", Tag: "required", Message: "id is required"},
		{Field: "name", Tag: "max", Message: "name is too long"},
	}
	e := apperror.Validation("test", fields)
	require.Equal(t, fields, e.Fields)
}
