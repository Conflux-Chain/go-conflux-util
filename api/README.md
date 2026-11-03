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

`MustServe` and `MustServeFromViper` block while serving requests and are intended
to run for the lifetime of the process. They expose no graceful shutdown entry
point: canceling an application context does not stop the server, and these
helpers do not arrange to drain in-flight requests before process exit.
`ListenAndServe` errors other than `http.ErrServerClosed` are logged with `Fatal`
and exit the process. Configuration loading failures in `MustServeFromViper`
also exit the process.

`MustServeFromViper` applies default HTTP server timeouts: 5 seconds for request
headers, 15 seconds for the entire request, 30 seconds for response writing, and
60 seconds for idle keep-alive connections. Explicit zero configuration values
also use these defaults. `MustServe` uses the supplied `Config` as-is; callers
constructing it directly must set the desired timeouts themselves.

These fields follow `net/http.Server` timeout semantics. The write deadline also
includes request handling time; it does not cancel business logic. Adjust the
read/write timeouts for large uploads, downloads, or streaming APIs.

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

Callers must ensure that data passed to the JSON response helpers can be encoded
as JSON, including `BusinessError.Data`. For example, `NaN`, infinity, and cyclic
references cannot be encoded. Encoding failures are not converted into a uniform
error response; with the default Gin recovery middleware, they result in HTTP 500
with an empty body. HTTP clients must also handle responses outside this format.

## Pre-defined Error Codes

There are some pre-defined errors as below:

- 1: Invalid parameter.
- 2: Internal server error.
- 3: Too many requests.
- 4: Database error.
- 5: JWT error.

## HTTP Response Status Code

The response helpers use the following status codes for successfully encoded JSON responses:

- 200: success, or known business error, e.g. entity not found.
- 600: an unexpected system error passed to `ResponseError`, such as an I/O error.

## Middlewares

There are some common used middlewares available:

- JWT
- Metrics
- Wrapper to return (value, error) pair
- Rate limit

### JWT token purpose

`Jwt.Generate` signs access and refresh tokens with a `tokenUse` claim of
`access` and `refresh`, respectively. `Jwt.Middleware` accepts only access tokens.
Use `ValidateRefreshToken` in the refresh handler; do not protect that handler
with the access-token middleware. For manual business-request authentication,
use `ValidateAccessToken`. The former general `Validate` method has been removed;
existing callers must choose the appropriate purpose-specific method when upgrading.

Both validators require an unexpired `exp` claim and the configured signing algorithm.
`Issuer` and `Subject` are checked only when their corresponding configuration values
are nonempty. `Generate` already sets `exp`; externally issued tokens without it are
rejected. Parser validation errors can be inspected with `errors.Is` and the
`jwt/v5` error values instead of matching error text.

Tokens issued before this claim was introduced are rejected by the middleware
and both purpose-specific validators. Users must sign in again or the service
must arrange a migration; a missing claim must not default to `access`, since
legacy refresh tokens have no purpose marker either.

## Validators

There are some helpful validator tags available:

- `hex`: starts with `0x` prefix and contains an even number of hexadecimal characters, e.g. `0x`, `0x12`, `0x12abCD`. The value could be decoded via `hexutil.MustDecode(hexString)`, and it will never be panic.

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
