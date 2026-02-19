package errs

import (
	"strings"
)

type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

type ActionType string

const (
	ActionTypeRedirect ActionType = "redirect"
)

type Action struct {
	Type    ActionType `json:"type"`
	Message string     `json:"message"`
	Value   string     `json:"value"`
}

type ErrorResponse struct {
	Message  string `json:"message"`
	Status   int    `json:"status"`
	Success  bool   `json:"success"`
	Override bool   `json:"override"`
	// field level errors
	Errors []FieldError `json:"errors"`
	// action to be taken
	Action *Action `json:"action"`
}

func (e *ErrorResponse) Error() string {
	return e.Message
}

func (e *ErrorResponse) Is(target error) bool {
	_, ok := target.(*ErrorResponse)

	return ok
}

func (e *ErrorResponse) WithMessage(message string) *ErrorResponse {
	return &ErrorResponse{
		Success:  false,
		Message:  message,
		Status:   e.Status,
		Override: e.Override,
		Errors:   e.Errors,
		Action:   e.Action,
	}
}

func MakeUpperCaseWithUnderscores(str string) string {
	return strings.ToUpper(strings.ReplaceAll(str, " ", "_"))
}
