# Httpx, http errors simplified

Highly opinionated way of returning errors in `http` or `httprouter` handlers.

## Why?

Sometimes this happens:

```go
func myCoolHandler(w http.ResponseWriter, r *http.Request) {
    user, err := accessSomeDB()
    if err != nil {
        w.Write(http.StatusBadRequest)
        // missing return, making successful path
        // to keep running
    }
}
```

It could be better to just do this

```go
func myCoolHandler(w http.ResponseWriter, r *http.Request) error {
    user, err := accessSomeDB()
    if err != nil {
        return httpx.Code(http.BadRequest)
    }
}
```

And to integrate in your routes you would do something like this:

```go
http.HandleFunc("/", httpx.H(myCoolHandler))
```

In `httpx` you can even do this if you're lazy and prototyping APIs:

```go
func myCoolHandler(w http.ResponseWriter, r *http.Request) error {
    user, err := accessSomeDB()
    if err != nil {
        return fmt.Errorf("some error: %w", err)
    }
}
```

By default this would return a 400 and some error text to the browser.
If you want to customize it, like instead of returning a 400 you can
return a 500.

```go
httpx.DefaultErrorHandler = func(err error) *httpx.Response {
    return httpx.Code(http.StatusInternalServerError).Text(err.Error())
}
```

httpx also has useful methods for returning JSON, text, and reader streams in the body.

- JSON

Sets Content-Type "application/json" for you

```go
return httpx.Code(http.StatusBadRequest).JSON(map[string]string{"hello": "world"})
```

- Text

```go
return httpx.Code(http.StatusBadRequest).Text("hello world")
```

- Reader

```go
reader := strings.NewReader("hello world")
// note: if using a ReadCloser like *os.File, you should use ReadCloser()
return httpx.Code(http.StatusBadRequest).Reader(reader)
```

### Setting headers

You can set headers on your error object. Note that they will override same-value headers set by you
on the http.ResponseWriter.

```go
return httpx.Code(http.StatusUnauthorized).Headers(map[string]string{
			"Content-Type": "hello/world",
	}).Headers(map[string]string{"X-Chainable": "headers"}).Text("Unauthorized")
```

### Error JSON codes

The thing more opinionated in the library is here. You can define custom error codes
that you can return as JSON in your errors.

```go
var ErrCreatingFile = httpx.NewCode("CREATING_FILE", http.StatusBadRequest)
var ErrCopyFile = httpx.NewCode("COPY_FILE", http.StatusBadRequest)
var ErrOpenFile = httpx.NewCode("OPEN_FILE", http.StatusBadRequest)
http.Handle("/stream", http.MaxBytesHandler(http.HandlerFunc(httpx.H(func(w http.ResponseWriter, r *http.Request) error {
    tmp := path.Join(os.TempDir(), "my-file")
    fd, err := os.Create(tmp)
    if err != nil {
        // returns: 400 {"code": "CREATING_FILE", "extra": <output of err.Error()> }
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
```

It's useful to define what kind of errors your server will return. You can attach an extra payload that will
be returned on the "extra" field.

### Httprouter integration

It works exactly the same like the `H()` wrapper function, but it's called `HRouter()`.

## Why this one though?

Some other libraries solve this by adding either their own context
and not integrating with the http package out of the box.
This one is for people that really like httprouter and wish there was a less boilerplatey
way of returning errors.

## Sample usage

You have a couple of methods to return errors in httpx, as you can see in the `examples` folder.
