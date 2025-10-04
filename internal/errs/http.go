package errs

import "strings"

// Action and ActionType define a structured way of communicating "what the client should do next"
// when an error or special condition occurs. Instead of returning only a plain error message,
// the backend can send an actionable response that the frontend can interpret and execute.
//
// Example use case: If a user's session has expired, the backend returns an Action with
// Type = "redirect", Message = "Your session has expired. Please log in again.",
// and Value = "/login". The frontend, upon seeing ActionTypeRedirect, knows it should
// navigate the user to the login page.
type ActionType string

const (
	ActionTypeRedirect ActionType = "redirect"
)

type Action struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Value   string `json:"value"`
}

// for input field specific errors
type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

// HttpError represents a structured error for HTTP responses.
type HttpError struct {
	Code     string       `json:"code"`
	Status   int          `json:"status"`
	Message  string       `json:"message"`
	Override bool         `json:"override"`
	Action   *Action      `json:"action,omitempty"`
	Errors   []FieldError `json:"fields,omitempty"`
}

func (e *HttpError) Error() string {
	return e.Message
}

func (e *HttpError) Is(target error) bool {
	_, ok := target.(*HttpError)

	return ok
}

func (e *HttpError) WithMessage(message string) *HttpError {
	return &HttpError{
		Code:     e.Code,
		Status:   e.Status,
		Message:  message,
		Override: e.Override,
		Action:   e.Action,
		Errors:   e.Errors,
	}
}

func MakeUpperCaseWithUnderscores(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, " ", "_"))
}
