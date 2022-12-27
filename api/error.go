package api

const (
	ErrCodeSuccess         = 0
	ErrCodeValidation      = 1
	ErrCodeInternal        = 2
	ErrCodeTooManyRequests = 3
)

type BusinessError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func NewBusinessError(code int, message string, data interface{}) *BusinessError {
	return &BusinessError{code, message, data}
}

func Success(data interface{}) *BusinessError {
	return NewBusinessError(ErrCodeSuccess, "Success", data)
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
