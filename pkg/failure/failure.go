package failure

import (
	"errors"
	"net/http"
)

type Failure struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error returns the error code and message in a formatted string.
func (e *Failure) Error() string {
	return e.Message
}

// BadRequest returns a new Failure with code for bad requests.
func BadRequest(err error) error {
	if err != nil {
		return &Failure{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		}
	}

	return nil
}

// BadRequestFromString returns a new Failure with code for bad requests with message set from string.
func BadRequestFromString(msg string) error {
	return &Failure{
		Code:    http.StatusBadRequest,
		Message: msg,
	}
}

// Unauthorized returns a new Failure with code for unauthorized requests.
func Unauthorized(msg string) error {
	return &Failure{
		Code:    http.StatusUnauthorized,
		Message: msg,
	}
}

// InternalError returns a new Failure with code for internal error and message derived from an error interface.
func InternalError(err error) error {
	if err != nil {
		return &Failure{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}

	return nil
}

// Unimplemented returns a new Failure with code for unimplemented method.
func Unimplemented(methodName string) error {
	return &Failure{
		Code:    http.StatusNotImplemented,
		Message: methodName,
	}
}

// NotFound returns a new Failure with code for entity not found.
func NotFound(entityName string) error {
	return &Failure{
		Code:    http.StatusNotFound,
		Message: entityName,
	}
}

// Conflict returns a new Failure with code for conflict situations.
func Conflict(message string) error {
	return &Failure{
		Code:    http.StatusConflict,
		Message: message,
	}
}

func Forbidden(msg string) error {
	return &Failure{
		Code:    http.StatusForbidden,
		Message: msg,
	}
}

// GetCode returns the error code of an error interface.
func GetCode(err error) int {
	var f *Failure
	if errors.As(err, &f) {
		return f.Code
	}

	return http.StatusInternalServerError
}
