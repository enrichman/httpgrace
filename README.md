# httpgrace

`httpgrace` is a minimal Go library providing drop-in replacements for the `net/http` standard library functions to run HTTP/HTTPS servers with graceful shutdown support out of the box.

## Installation and Usage

Just `go get github.com/yourusername/httpgrace` and replace `http` with `httpgrace` in your existing code:

```go
// Before
http.ListenAndServe(":8080", handler)

// After  
httpgrace.ListenAndServe(":8080", handler)
```

That's it! Your server now gracefully shuts down on SIGINT/SIGTERM signals.

## Features

- API compatible with `net/http`'s `ListenAndServe`, `ListenAndServeTLS`, `Serve`, and `ServeTLS`  
- Graceful shutdown on `SIGINT`/`SIGTERM` signals  
- Configurable shutdown timeout (default 10s)  
- Built-in structured logging via Go's `slog` package  
- Minimal and dead-simple to integrate — just swap your import path!

## API Reference

### Simple Functions (Drop-in Replacements)

```go
// HTTP server
httpgrace.ListenAndServe(addr, handler, opts...)

// HTTPS server  
httpgrace.ListenAndServeTLS(addr, certFile, keyFile, handler, opts...)

// Custom listener
httpgrace.Serve(listener, handler, opts...)
httpgrace.ServeTLS(listener, certFile, keyFile, handler, opts...)
```

### Server Struct

If you need more granular control, you can use the Server struct directly:

```go
srv := httpgrace.NewServer(handler)
if err := srv.ListenAndServe(":8080"); err != nil {
    log.Fatal(err)
}
```

## Configuration Options

### Shutdown Options

```go
// Set graceful shutdown timeout (default: 10 seconds)
httpgrace.WithTimeout(5*time.Second)

// Customize shutdown signals (default: SIGINT, SIGTERM)
httpgrace.WithSignals(syscall.SIGTERM, syscall.SIGUSR1)

// Provide custom logger (default: slog.Default())
httpgrace.WithLogger(customLogger)
```

### Server Options

You can configure the underlying http.Server with the provided functions or custom ones:

```go
srv := httpgrace.NewServer(handler,
    httpgrace.WithServerOptions(
        httpgrace.WithReadTimeout(10*time.Second),
        httpgrace.WithWriteTimeout(10*time.Second),
        httpgrace.WithIdleTimeout(120*time.Second),
        // or with your custom ServerOption
        func(srv *http.Server) {
            srv.ErrorLog = log.New(os.Stdout, "", 0)
        },
    ),
)

// Start server
if err := srv.ListenAndServe(":8080"); err != nil {
    log.Fatal(err)
}
```

## Graceful Shutdown Behavior

`httpgrace` listens for `SIGINT` and `SIGTERM` signals. Upon receiving one, it stops accepting new connections and waits up to the configured shutdown timeout for active connections to finish before exiting.

This ensures your server shuts down cleanly without dropping in-flight requests abruptly.

## Logging

`httpgrace` logs key events such as server startup and shutdown progress using Go's `slog` package. By default, logs are output using `slog.Default()`. You can provide a custom logger with `WithLogger`.

Example log messages:

``` 
time=2025-05-28T22:14:21.301+02:00 level=INFO msg="starting server" mode=HTTP addr=[::]:8080 shutdown_timeout=10s
time=2025-05-28T22:14:28.258+02:00 level=INFO msg="shutdown signal received" signal=interrupt
time=2025-05-28T22:14:28.258+02:00 level=INFO msg="server shutdown completed gracefully" duration=204.273µs
```

## Requirements

- Go 1.21+ (for slog package support)
- No external dependencies!

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[MIT License](./LICENSE) © Enrico Candino
