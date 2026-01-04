package glib

import (
	"net/http"
)

// Next is a function that invokes the next middleware in the chain,
// or the actual API handler if this is the last middleware.
// It returns a Response that can be inspected and modified.
type Next func(Request) Response

// Response represents the API handler's response.
// Middleware can inspect and modify this response.
type Response struct {
	// Payload is the API's response data.
	// For (T, error) handlers, this is the value of T.
	// For raw HTTP handlers, this is nil.
	//
	// Middleware can modify the payload, but the type must remain the same
	// as the handler's return type. Changing the type will cause a runtime error.
	Payload any

	// Err is the error returned by the handler or previous middleware.
	// If non-nil, it will be serialized as an error response using glib.WriteError.
	// Setting this to non-nil effectively short-circuits success and returns an error.
	Err error

	// HTTPStatus is the HTTP status code for the response.
	// If zero, glib will automatically determine the status code based on:
	//   - The error code (if Err is non-nil)
	//   - The HTTP method and success (e.g., 201 for POST, 204 for DELETE with no payload)
	//   - Response struct tags (response:"httpstatus")
	//
	// Set this to a non-zero value to override the automatic status code selection.
	HTTPStatus int

	// headers stores response headers that middleware can set.
	// Use Header() to access and modify headers.
	headers http.Header
}

// Header returns the HTTP response headers.
// Middleware can use this to add or modify headers.
//
// For raw HTTP handlers, header modifications may not take effect if
// the handler has already called w.WriteHeader() or written to w.
//
// Example:
//
//	resp.Header().Set("X-Request-ID", requestID)
//	resp.Header().Add("X-Custom", "value")
func (r *Response) Header() http.Header {
	if r.headers == nil {
		r.headers = make(http.Header)
	}
	return r.headers
}

func (r *Response) WithHeader(key, value string) *Response {
	r.Header().Set(key, value)
	return r
}

// Middleware is the signature that middleware functions should have.
// This type exists solely for documentation purposes.
//
// Middleware forms a chain where each middleware can:
//  1. Inspect and modify the incoming request
//  2. Call next(req) to invoke the next middleware or handler
//  3. Inspect and modify the outgoing response
//  4. Return early with an error without calling next
//
// Example:
//
//	func Auth(jwtService *JWTService) func(glib.Request, glib.Next) glib.Response {
//	    return func(req glib.Request, next glib.Next) glib.Response {
//	        token := req.Header("Authorization")
//	        if token == "" {
//	            return glib.Response{Err: errs.NewUnauthorized().Msg("missing token").Err()}
//	        }
//
//	        // Validate and add to context
//	        claims, err := jwtService.ValidateToken(token)
//	        if err != nil {
//	            return glib.Response{Err: err}
//	        }
//
//	        req = req.WithValue("user_id", claims.UserID)
//
//	        // Continue chain and add response header
//	        resp := next(req)
//	        resp.Header().Set("X-User-ID", claims.UserID.String())
//	        return resp
//	    }
//	}
type Middleware func(Request, Next) Response
