package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/gabivlj/httpx"
)

var ErrCodeNotFound = httpx.NewCode("NOT_FOUND", http.StatusNotFound)
var ErrCreatingFile = httpx.NewCode("CREATING_FILE", http.StatusBadRequest)
var ErrCopyFile = httpx.NewCode("COPY_FILE", http.StatusBadRequest)
var ErrOpenFile = httpx.NewCode("OPEN_FILE", http.StatusBadRequest)

func main() {
	http.HandleFunc("/hellox", httpx.H(func(w http.ResponseWriter, r *http.Request) error {
		if len(r.URL.Query()) > 0 {
			return ErrCodeNotFound.JSON()
		}

		// httpx shouldn't be used for returning successful responses,
		// however you can use it like that.
		return httpx.Code(http.StatusOK).JSON(map[string]string{"hello": "world"})
	}))

	// handle errors that aren't httpx.Response
	httpx.DefaultErrorHandler = func(err error) *httpx.Response {
		return httpx.Code(http.StatusBadRequest).Text(err.Error())
	}

	// function that will be fired after request is processed, default is no-op.
	//
	httpx.DefaultAfterMiddleware = func(w http.ResponseWriter, r *http.Request, e error) {
		switch err := e.(type) {
		case *httpx.Response:
			log.Println("returned:", err.Code)
		default:
			log.Println("returned non httpx.Response")
		}
	}

	func() error {
		return httpx.Code(http.StatusUnauthorized).Headers(map[string]string{
			"Content-Type": "hello/world",
		}).Headers(map[string]string{"X-Chainable": "headers"}).Text("Unauthorized")
	}()

	http.Handle("/stream", http.MaxBytesHandler(http.HandlerFunc(httpx.H(func(w http.ResponseWriter, r *http.Request) error {
		tmp := path.Join(os.TempDir(), "my-file")
		fd, err := os.Create(tmp)
		if err != nil {
			return ErrCreatingFile.JSON(err.Error())
		}

		defer fd.Close()
		_, err = io.Copy(fd, r.Body)
		if err != nil {
			return ErrCopyFile.JSON(err.Error())
		}

		fd, err = os.Open(tmp)
		if err != nil {
			return ErrOpenFile.JSON(err.Error())
		}

		return httpx.Code(http.StatusOK).ReadCloser(fd)
	})), 1024*1024*10))

	http.HandleFunc("/hello", httpx.H(func(w http.ResponseWriter, r *http.Request) error {
		return fmt.Errorf("plain text and returns a 400")
	}))

	http.ListenAndServe(":8080", nil)
}
