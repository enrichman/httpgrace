# httpgrace

`httpgrace` is a minimal Go library providing drop-in replacements for the `net/http` standard library functions to run HTTP/HTTPS servers with graceful shutdown support out of the box.

---

## Features

- API compatible with `net/http`'s `ListenAndServe` and `ListenAndServeTLS`  
- Graceful shutdown on `SIGINT` / `SIGTERM` signals  
- Configurable shutdown timeout (default 10 seconds)  
- Built-in structured logging via Go's `slog` package  
- Minimal and dead-simple to integrate — just swap your import path!

---

## Installation

`go get github.com/yourusername/httpgrace`

---

## Usage

Simply replace `net/http`'s `ListenAndServe` and `ListenAndServeTLS` with `httpgrace.ListenAndServe` and `httpgrace.ListenAndServeTLS` in your existing code:

```go
package main

import (
    "github.com/yourusername/httpgrace"
    "net/http"
)

func main() {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, graceful world!\n"))
    })

    // Start HTTP server with graceful shutdown support  
    if err := httpgrace.ListenAndServe(":8080", handler); err != nil {
        panic(err)
    }
}
```

For HTTPS:

```go
err := httpgrace.ListenAndServeTLS(":8443", "cert.pem", "key.pem", handler)
```

---

## Options

Customize server behavior with functional options:

```go
httpgrace.ListenAndServe(":8080", handler,
    httpgrace.WithShutdownTimeout(5*time.Second), // default is 10s
    httpgrace.WithLogger(customLogger),            // provide your own slog.Logger
)
```


## Graceful Shutdown Behavior

`httpgrace` listens for `SIGINT` and `SIGTERM` signals. Upon receiving one, it stops accepting new connections and waits up to the configured shutdown timeout for active connections to finish before exiting.

This ensures your server shuts down cleanly without dropping in-flight requests abruptly.

---

## Logging

`httpgrace` logs key events such as server startup and shutdown progress using Go's `slog` package. By default, logs are output using `slog.Default()`. You can provide a custom logger with `WithLogger`.

Example log messages:

``` 
INFO Starting server mode=HTTP addr=:8080
INFO Signal received, shutting down server signal=interrupt
INFO Server shutdown completed
```

---


## License

[MIT License](./LICENSE) © Enrico Candino
