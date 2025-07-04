package api

const (
	ErrCodeSuccess         = 0
	ErrCodeValidation      = 1
	ErrCodeInternal        = 2
	ErrCodeTooManyRequests = 3
)

var ErrNil = &BusinessError{ErrCodeSuccess, "Success", nil}

type BusinessError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
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

func ErrInternal(err error) *BusinessError {
	return NewBusinessError(ErrCodeInternal, "Internal server error", err.Error())
}

func ErrTooManyRequests(err error) *BusinessError {
	return NewBusinessError(ErrCodeTooManyRequests, "Too many requests", err.Error())
}

func (err *BusinessError) WithData(data any) *BusinessError {
	return &BusinessError{err.Code, err.Message, data}
}

func (err *BusinessError) Error() string {
	return err.Message
}
