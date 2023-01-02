// package httpx is a package designed to speed up and prevent bugs regarding error returns on HTTP servers.
package httpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// DefaultErrorHandler is the function that will be fired when an error that is not httpx.Response if returned
var DefaultErrorHandler = func(err error) *Response {
	return Code(http.StatusBadRequest).Text(err.Error())
}

// DefaultAfterMiddleware is the function that will be fired after writing to the client
var DefaultAfterMiddleware = func(w http.ResponseWriter, r *http.Request, e error) {}

// CopyErrorHandler is the function that will be fired after an error copying to the http.ResponseWriter
var CopyErrorHandler = func(err error) {}

// Response is the standard struct that implements error for httpx. It should be mainly used to return errors
type Response struct {
	Code    int
	headers map[string]string
	Copy    func(io.Writer) error
}

func (r *Response) Error() string {
	b := &bytes.Buffer{}
	err := r.Copy(b)
	if err != nil {
		return fmt.Errorf("reading all bytes from Reader: %w", err).Error()
	}

	return b.String()
}

// Handler is the httpx handler, prepared to be the same as a http handler where you're able to return errors
type Handler func(w http.ResponseWriter, r *http.Request) error

// HttpRouterHandler is the httpxrouter handler, prepared to be the same as a httprouter handle where you're able to return errors
type HttpRouterHandler func(w http.ResponseWriter, r *http.Request, p httprouter.Params) error

// H wraps a httpx handler with a http.HandlerFunc
func H(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fireAfterMiddleware(h(w, r), w, r)
	}
}

// HRouter wraps a httpxrouter handler with a httprouter.Handle
func HRouter(h HttpRouterHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		fireAfterMiddleware(h(w, r, p), w, r)
	}
}

// fireAfterMiddleware handles the error with the w and r
func fireAfterMiddleware(err error, w http.ResponseWriter, r *http.Request) {
	if err == nil {
		return
	}

	res, ok := err.(*Response)
	if !ok {
		res = DefaultErrorHandler(err)
	}

	for key, value := range res.headers {
		w.Header().Set(key, value)
	}

	w.WriteHeader(res.Code)
	if res.Copy == nil {
		return
	}

	err = res.Copy(w)
	if err != nil {
		CopyErrorHandler(err)
	}

	DefaultAfterMiddleware(w, r, res)
}

// Code returns an empty response with the status code set
func Code(code int) *Response {
	return &Response{Code: code, headers: map[string]string{}}
}

// JSON returns a JSON response with application/json
func (r *Response) JSON(value any) *Response {
	r.headers["Content-Type"] = "application/json"
	r.Copy = func(w io.Writer) error {
		return json.NewEncoder(w).Encode(value)
	}
	return r
}

// NoContent just returns an empty OK response
func NoContent() *Response {
	return Code(http.StatusOK)
}

// Text returns a plain text response
func (r *Response) Text(s string) *Response {
	r.Copy = func(w io.Writer) error {
		_, err := io.Copy(w, strings.NewReader(s))
		return err
	}

	return r
}

// Headers set the headers in the parameter to the response headers.
// You can chain this function multiple times.
// The headers set on http.ResponseWriter will be kept in mind.
func (r *Response) Headers(m map[string]string) *Response {
	if r.headers != nil {
		for k, v := range m {
			r.headers[k] = v
		}

		return r
	}

	r.headers = m
	return r
}

// Reader sets the reader as the body response
func (r *Response) Reader(reader io.Reader) *Response {
	r.Copy = func(w io.Writer) error { _, err := io.Copy(w, reader); return err }
	return r
}

// ReadCloser is the same as Reader but closes the ReadCloser when it finishes copying.
func (r *Response) ReadCloser(reader io.ReadCloser) *Response {
	r.Copy = func(w io.Writer) error { defer reader.Close(); _, err := io.Copy(w, reader); return err }
	return r
}

// ErrorJSONCode lets you create codes that you can use as errors in a very opinionated way for httpx.
// Example:
//	var ErrNotFound = httpx.NewCode("NOT_FOUND", http.StatusNotFound)
//
//	func handler(w http.ResponseWriter, r *http.Request) error {
//		return ErrNotFound.JSON() // 404 { "code": "NOT_FOUND" }
//	}
type ErrorJSONCode struct {
	Code   string
	Status int
	Extra  any
}

// NewCode returns a new ErrorJSONCode to generate error codes formated in JSON
func NewCode(code string, status int) *ErrorJSONCode {
	return &ErrorJSONCode{
		Code:   code,
		Status: status,
	}
}

// JSON creates a Response from a ErrorJSONCode
func (e *ErrorJSONCode) JSON(extra ...any) *Response {
	var value any
	if len(extra) > 0 {
		value = extra[0]
	}

	json := map[string]any{
		"code": e.Code,
	}
	if value != nil {
		json["extra"] = value
	}
	return Code(e.Status).JSON(json)
}
