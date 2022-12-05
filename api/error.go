package api

const (
	ErrCodeSuccess    = 0
	ErrCodeValidation = 1
	ErrCodeInternal   = 2
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

func (err *BusinessError) Error() string {
	return err.Message
}
