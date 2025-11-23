package httperror

import (
	"fmt"
	"net/http"
)

type Error struct {
	Status  int         `json:"-"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(status int, code, message string, details interface{}) *Error {
	if status == 0 {
		status = http.StatusInternalServerError
	}

	if code == "" {
		code = "internal_server_error"
	}

	if message == "" {
		message = http.StatusText(status)
	}

	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
		Details: details,
	}
}

func BadRequest(code, message string, details interface{}) *Error {
	return New(http.StatusBadRequest, code, message, details)
}

func Unauthorized(code, message string, details interface{}) *Error {
	return New(http.StatusUnauthorized, code, message, details)
}

func Conflict(code, message string, details interface{}) *Error {
	return New(http.StatusConflict, code, message, details)
}

func NotFound(code, message string, details interface{}) *Error {
	return New(http.StatusNotFound, code, message, details)
}

func InternalServerError(code, message string, details interface{}) *Error {
	return New(http.StatusInternalServerError, code, message, details)
}

func UnprocessableEntity(code, message string, details interface{}) *Error {
	return New(http.StatusUnprocessableEntity, code, message, details)
}

func NoContent(code, message string, details interface {}) *Error {
	return New(http.StatusNoContent, code, message, details)
}

func Accepted(code, message string, details interface {}) *Error {
	return New(http.StatusAccepted, code, message, details)
}