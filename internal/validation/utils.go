package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/Barry-dE/go-backend-boilerplate/internal/errs"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Validatable interface {
	Validate() error
}

type CustomValidationError struct {
	Field   string
	Message string
}

type CustomValidationErrors []CustomValidationError

func (c CustomValidationErrors) Error() string {
	return "Validation failed"
}

func BindAndValidate(c echo.Context, payload Validatable) error {
	if err := c.Bind(payload); err != nil {
		message := strings.Split(strings.Split(err.Error(), ",")[1], "message=")[1]
		return errs.BadRequestError(message, false, nil, nil, nil)
	}

	if msg, fieldErrors := validateStruct(payload); fieldErrors != nil {
		return errs.BadRequestError(msg, true, nil, fieldErrors, nil)
	}

	return nil
}

func validateStruct(v Validatable) (string, []errs.FieldError) {
	if err := v.Validate(); err != nil {
		return extractValidationErrors(err)
	}
	return "", nil
}

func extractValidationErrors(err error) (string, []errs.FieldError) {
	var fieldErrors []errs.FieldError

	// Check if the error is a custom validation error type
	if customValidationError, ok := err.(CustomValidationErrors); ok {
		for _, err := range customValidationError {
			fieldErrors = append(fieldErrors, errs.FieldError{
				Field: err.Field,
				Error: err.Message,
			})
		}
	}

	// Check if the error is a validation error from the validator package
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, err := range validationErrors {
			fieldErrors = append(fieldErrors, errs.FieldError{
				Field: strings.ToLower(err.Field()),
				Error: getValidationMessage(err),
			})
		}
	}

	return "Validation failed", fieldErrors
}

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func IsValidUUID(uuid string) bool {
	return uuidRegex.MatchString(uuid)
}

// getValidationMessage generates user-friendly error messages based on validation tag
func getValidationMessage(err validator.FieldError) string {
	isString := err.Type().Kind() == reflect.String

	switch err.Tag() {
	case "required":
		return "is required"
	case "min":
		if isString {
			return fmt.Sprintf("must be at least %s characters", err.Param())
		}
		return fmt.Sprintf("must be at least %s", err.Param())
	case "max":
		if isString {
			return fmt.Sprintf("must not exceed %s characters", err.Param())
		}
		return fmt.Sprintf("must not exceed %s", err.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", err.Param())
	case "email":
		return "must be a valid email address"
	case "e164":
		return "must be a valid phone number with country code"
	case "uuid":
		return "must be a valid UUID"
	case "uuidList":
		return "must be a comma-separated list of valid UUIDs"
	case "dive":
		return "some items are invalid"
	default:
		field := strings.ToLower(err.Field())
		if err.Param() != "" {
			return fmt.Sprintf("%s: %s:%s", field, err.Tag(), err.Param())
		}
		return fmt.Sprintf("%s: %s", field, err.Tag())
	}
}
