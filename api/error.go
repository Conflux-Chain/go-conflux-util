package api

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	ErrCodeSuccess         = 0
	ErrCodeValidation      = 1
	ErrCodeInternal        = 2
	ErrCodeTooManyRequests = 3
	ErrCodeDatabase        = 4
	ErrCodeJwt             = 5
)

// ErrNil is used for success response.
var ErrNil = &BusinessError{ErrCodeSuccess, "Success", nil}

// BusinessError is uniform data structure of REST API.
type BusinessError struct {
	// Code error code, 0 indicates success, otherwise business error (e.g. user not found) or internal server error.
	Code int `json:"code"`

	// Message error message associated with `Code`.
	Message string `json:"message"`

	// Data is the return value if success. Otherwise, it is concret failure reason in string type.
	Data any `json:"data"`
}

func NewBusinessError(code int, message string, data ...any) *BusinessError {
	if len(data) > 0 {
		return &BusinessError{code, message, data[0]}
	}

	return &BusinessError{code, message, nil}
}

func ErrValidation(err error) *BusinessError {
	return NewBusinessError(ErrCodeValidation, "Invalid parameter", err.Error())
}

func ErrValidationStr(err string) *BusinessError {
	return NewBusinessError(ErrCodeValidation, "Invalid parameter", err)
}

func ErrValidationStrf(err string, args ...any) *BusinessError {
	return NewBusinessError(ErrCodeValidation, "Invalid parameter", fmt.Sprintf(err, args...))
}

func ErrInternal(err error) *BusinessError {
	return NewBusinessError(ErrCodeInternal, "Internal server error", err.Error())
}

func ErrTooManyRequests(err error) *BusinessError {
	return NewBusinessError(ErrCodeTooManyRequests, "Too many requests", err.Error())
}

func ErrDatabase(err error) *BusinessError {
	return NewBusinessError(ErrCodeDatabase, "Database error", err.Error())
}

func ErrDatabaseCause(err error, cause string) *BusinessError {
	return NewBusinessError(ErrCodeDatabase, "Database error", errors.WithMessage(err, cause).Error())
}

func ErrDatabaseCausef(err error, cause string, args ...any) *BusinessError {
	return NewBusinessError(ErrCodeDatabase, "Database error", errors.WithMessagef(err, cause, args...).Error())
}

func ErrJwt(cause string) *BusinessError {
	return NewBusinessError(ErrCodeJwt, "JWT error", cause)
}

func (err *BusinessError) WithData(data any) *BusinessError {
	return &BusinessError{err.Code, err.Message, data}
}

func (err *BusinessError) Error() string {
	if err.Data == nil {
		return fmt.Sprintf("%v: %v", err.Code, err.Message)
	}

	return fmt.Sprintf("%v: %v (%+v)", err.Code, err.Message, err.Data)
}
