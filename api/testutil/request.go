package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/api"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Request is used to send HTTP request in unit tests.
type Request struct {
	engine  *gin.Engine
	method  string
	url     string
	body    io.Reader
	headers map[string]string
}

// NewRequest creates a request with specified method and URL.
//
// Note, query params are set by `WithQueryParams` instead of concatenated with URL.
func NewRequest(engine *gin.Engine, method, url string) *Request {
	return &Request{
		engine:  engine,
		method:  method,
		url:     url,
		headers: make(map[string]string),
	}
}

// MustWithJSONBody builds JSON body with content type application/json.
func (r *Request) MustWithJSONBody(body any) *Request {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		panic(errors.WithMessagef(err, "Failed to marshal JSON body %v", body))
	}

	r.body = bytes.NewBufferString(string(jsonBody))
	r.headers["Content-Type"] = "application/json"

	return r
}

// WithAuth builds Authorization header with "Bearer xxx".
func (r *Request) WithAuth(accessToken string) *Request {
	r.headers["Authorization"] = "Bearer " + accessToken

	return r
}

// WithQueryParams builds query params to concatenate with URL.
func (r *Request) WithQueryParams(params map[string]string) *Request {
	var builder strings.Builder

	for k, v := range params {
		if builder.Len() == 0 {
			builder.WriteString("?")
		} else {
			builder.WriteString("&")
		}

		builder.WriteString(k + "=" + v)
	}

	r.url += builder.String()

	return r
}

// MustWithMultipart builds form data in request body.
func (r *Request) MustWithMultipart(fields map[string]string, files ...map[string]string) *Request {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	for k, v := range fields {
		fieldWriter, err := writer.CreateFormField(k)
		if err != nil {
			panic(errors.WithMessagef(err, "Failed to create form field: %v", k))
		}

		if _, err = fieldWriter.Write([]byte(v)); err != nil {
			panic(errors.WithMessage(err, "Failed to write form field"))
		}
	}

	if len(files) > 0 {
		for k, v := range files[0] {
			data, err := os.ReadFile(v)
			if err != nil {
				panic(errors.WithMessagef(err, "Failed to read file at %v", v))
			}

			fileWriter, err := writer.CreateFormFile(k, filepath.Base(v))
			if err != nil {
				panic(errors.WithMessagef(err, "Failed to create form file: %v", k))
			}

			if _, err = fileWriter.Write(data); err != nil {
				panic(errors.WithMessagef(err, "Failed to write form file at %v", v))
			}
		}
	}

	if err := writer.Close(); err != nil {
		panic(errors.WithMessage(err, "Failed to close multipart writer"))
	}

	r.body = body
	r.headers["Content-Type"] = writer.FormDataContentType()

	return r
}

// MustExecRaw sends HTTP request and requires 200 OK response.
//
// It panics on any error, or response status code is not 200.
func (r *Request) MustExecRaw() *httptest.ResponseRecorder {
	req, err := http.NewRequest(r.method, r.url, r.body)
	if err != nil {
		panic(errors.WithMessage(err, "Failed to create http request"))
	}

	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		panic(fmt.Sprintf("Unexpected http status code %v, response = %v", w.Code, w.Body.String()))
	}

	return w
}

// MustExec sends HTTP request and unmarshal response body to `api.BusinessError`.
func (r *Request) MustExec() *api.BusinessError {
	w := r.MustExecRaw()

	responseContentType := w.Header().Get("Content-Type")
	if !strings.Contains(responseContentType, "application/json") {
		panic(fmt.Sprintf("Invalid content type, expected = application/json, actual = %v", responseContentType))
	}

	result := new(api.BusinessError)
	data := w.Body.Bytes()

	if err := json.Unmarshal(data, result); err != nil {
		panic(errors.WithMessagef(err, "Failed to unmarshal data into business error, data = %v", string(data)))
	}

	return result
}

// AssertSuccess sends HTTP request and asserts success with optional data.
func (r *Request) AssertSuccess(t *testing.T, data ...any) {
	result := r.MustExec()

	if len(data) == 0 {
		assert.Equal(t, api.ErrNil.Code, result.Code)
	} else {
		assert.Equal(t, api.ErrNil.WithData(data[0]), result)
	}
}

// AssertSuccessGet sends HTTP request and asserts to unmarshal data with specified `dataPtr`.
func (r *Request) AssertSuccessGet(t *testing.T, dataPtr any) {
	result := r.MustExec()

	assertSuccess(t, result, dataPtr)
}

// AssertErrValidation sends HTTP request and asserts api.ErrCodeValidation retrieved.
func (r *Request) AssertErrValidation(t *testing.T) {
	result := r.MustExec()

	assert.Equal(t, api.ErrCodeValidation, result.Code)
}

// AssertError sends HTTP request and asserts an expected business error.
func (r *Request) AssertError(t *testing.T, expectedError *api.BusinessError) {
	result := r.MustExec()

	assert.Equal(t, expectedError, result)
}

// AssertSuccess asserts to unmarshal the business error and returns specific type of data.
func AssertSuccess[T any](t *testing.T, result *api.BusinessError) (data T) {
	assertSuccess(t, result, &data)
	return
}

func assertSuccess(t *testing.T, result *api.BusinessError, dataPtr any) {
	assert.Equal(t, api.ErrCodeSuccess, result.Code)

	encoded, err := json.Marshal(result.Data)
	if err != nil {
		panic(errors.WithMessagef(err, "Failed to marshal data: %v", result.Data))
	}

	if err = json.Unmarshal(encoded, dataPtr); err != nil {
		panic(errors.WithMessage(err, "Failed to unmarshal into given dataPtr"))
	}
}
