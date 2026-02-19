package errs

import (
	"net/http"
)

func NewUnauthorizedError(message string, override bool) *ErrorResponse {
	return &ErrorResponse{
		Message:  message,
		Status:   http.StatusUnauthorized,
		Override: override,
		Success:  false,
	}
}

func NewForbiddenError(message string, override bool) *ErrorResponse {
	return &ErrorResponse{
		Message:  message,
		Status:   http.StatusForbidden,
		Success:  false,
		Override: override,
	}
}

func NewBadRequestError(message string, override bool, errors []FieldError, action *Action) *ErrorResponse {
	return &ErrorResponse{
		Message:  message,
		Status:   http.StatusBadRequest,
		Override: override,
		Errors:   errors,
		Action:   action,
		Success:  false,
	}
}

func NewNotFoundError(message string, override bool) *ErrorResponse {
	return &ErrorResponse{
		Message:  message,
		Status:   http.StatusNotFound,
		Override: override,
		Success:  false,
	}
}

func NewInternalServerError() *ErrorResponse {
	return &ErrorResponse{
		Message:  http.StatusText(http.StatusInternalServerError),
		Status:   http.StatusInternalServerError,
		Override: false,
		Success:  false,
	}
}

func ValidationError(err error) *ErrorResponse {
	return NewBadRequestError("Validation failed: "+err.Error(), false, nil, nil)
}
