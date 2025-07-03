package api

const (
	ErrCodeSuccess         = 0
	ErrCodeValidation      = 1
	ErrCodeInternal        = 2
	ErrCodeTooManyRequests = 3
)

var (
	errNil = &BusinessError{ErrCodeSuccess, "Success", nil}
)

type BusinessError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func NewBusinessError(code int, message string, data ...interface{}) *BusinessError {
	if len(data) > 0 {
		return &BusinessError{code, message, data[0]}
	}

	return &BusinessError{code, message, nil}
}

func Success(data interface{}) *BusinessError {
	if data == nil {
		return errNil
	}

	return NewBusinessError(errNil.Code, errNil.Message, data)
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

func (err *BusinessError) Error() string {
	return err.Message
}
