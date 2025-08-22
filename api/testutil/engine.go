package testutil

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Engine provides utilities to create HTTP request for unit tests.
type Engine struct {
	inner *gin.Engine

	baseURL string
}

// NewEngine creates a test engine with optional base URL.
func NewEngine(engine *gin.Engine, baseURL ...string) *Engine {
	var url string
	if len(baseURL) > 0 {
		url = baseURL[0]
	}

	return &Engine{
		inner:   engine,
		baseURL: url,
	}
}

// Request creates a request with given HTTP method and URL.
func (e *Engine) Request(method, url string) *Request {
	return NewRequest(e.inner, method, e.baseURL+url)
}

// Get creates a GET request with given URL.
func (e *Engine) Get(url string) *Request {
	return NewRequest(e.inner, http.MethodGet, e.baseURL+url)
}

// Post creates a POST request with given URL.
func (e *Engine) Post(url string) *Request {
	return NewRequest(e.inner, http.MethodPost, e.baseURL+url)
}

// Put creates a PUT request with given URL.
func (e *Engine) Put(url string) *Request {
	return NewRequest(e.inner, http.MethodPut, e.baseURL+url)
}

// Delete creates a DELETE request with given URL.
func (e *Engine) Delete(url string) *Request {
	return NewRequest(e.inner, http.MethodDelete, e.baseURL+url)
}
