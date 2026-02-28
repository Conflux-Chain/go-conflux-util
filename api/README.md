# REST API Utilities

This package aims to develop REST API in a standard way, and provides some utilities for unit test.

## Initialize

It is highly recommended to initialize REST API service from a configuration file. Developers only are required to provide a factory to set up routers.

```go
package main

import (
	"github.com/Conflux-Chain/go-conflux-util/api"
	"github.com/Conflux-Chain/go-conflux-util/api/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	// Setup routers
	factory := func(router *gin.Engine) {
		router.GET("/greeting", middleware.Wrap(someHandler))
	}

	// Start REST API server in a separate goroutine
	go api.MustServeFromViper(factory)

	select {}
}

func someHandler(c *gin.Context) (any, error) {
	return "hello world", nil
}

```

## Uniform HTTP Response

This module provides common HTTP responses along with standard errors in JSON format.

```go
type BusinessError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    any         `json:"data"`
}
```

`Code` **0** indicates success, and `Data` is an object indicates the return value of REST API. `Code` with non-zero value indicates any error, and developers could refer to the `Message` and `Data` fields for more details.

## Pre-defined Error Codes

There are some pre-defined errors as below:

- 1: Invalid parameter.
- 2: Internal server error.
- 3: Too many requests.
- 4: Database error.
- 5: JWT error.

## HTTP Response Status Code
To distinguish backend service error and gateway error, we only use `200` and `600` as HTTP response status code:

- 200: success, or known business error, e.g. entity not found.
- 600: unexpected system error, e.g. io error, encoding error.

## Middlewares

There are some common used middlewares available:

- JWT
- Metrics
- Wrapper to return (value, error) pair

## Unit Test

Developers could easily write unit tests for all exported REST APIs based on the available test utilities. See examples below:

```go
package foo

import (
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/api/testutil"
	"github.com/gin-gonic/gin"
)

func newTestEngine() testutil.Engine {
	engine := gin.Default()

	// setup routers
	// setupRoutes(engine)

	return *testutil.NewEngine(engine)
}

type User struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

func TestFoo(t *testing.T) {
	engine := newTestEngine()

	engine.Post("/users").						// build post URL
		MustWithJSONBody(User{Name: "Alice"}).	// build JSON body
		AssertSuccess(t, uint64(1))				// assert new user ID is 1
}
```
