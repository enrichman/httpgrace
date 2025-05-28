package httpgrace

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Option configures the server behavior.
type Option func(*serverConfig)

type serverConfig struct {
	shutdownTimeout time.Duration
	logger          *slog.Logger
}

func defaultConfig() serverConfig {
	return serverConfig{
		shutdownTimeout: 10 * time.Second,
		logger:          slog.Default(),
	}
}

// WithTimeout sets graceful shutdown timeout duration.
func WithTimeout(d time.Duration) Option {
	return func(cfg *serverConfig) {
		cfg.shutdownTimeout = d
	}
}

// WithLogger sets a custom slog.Logger for logging.
func WithLogger(l *slog.Logger) Option {
	return func(cfg *serverConfig) {
		if l != nil {
			cfg.logger = l
		}
	}
}

// ListenAndServe starts a non-TLS HTTP server with graceful shutdown.
func ListenAndServe(addr string, handler http.Handler, opts ...Option) error {
	return listenAndServeInternal(addr, "", "", handler, opts...)
}

// ListenAndServeTLS starts a TLS HTTP server with graceful shutdown.
func ListenAndServeTLS(addr, certFile, keyFile string, handler http.Handler, opts ...Option) error {
	return listenAndServeInternal(addr, certFile, keyFile, handler, opts...)
}

// Serve starts a non-TLS HTTP server with graceful shutdown on a custom net.Listener.
func Serve(ln net.Listener, handler http.Handler, opts ...Option) error {
	return serveInternal(ln, "", "", handler, opts...)
}

// ServeTLS starts a TLS HTTP server with graceful shutdown on a custom net.Listener.
func ServeTLS(ln net.Listener, certFile, keyFile string, handler http.Handler, opts ...Option) error {
	return serveInternal(ln, certFile, keyFile, handler, opts...)
}

// Internal ListenAndServe implementation.
func listenAndServeInternal(addr, certFile, keyFile string, handler http.Handler, opts ...Option) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return serveInternal(ln, certFile, keyFile, handler, opts...)
}

// Internal unified serve function for both Listener-based servers.
func serveInternal(ln net.Listener, certFile, keyFile string, handler http.Handler, opts ...Option) error {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	srv := &http.Server{
		Handler: handler,
	}

	// Log server start with address and mode (TLS or not)
	mode := "HTTP"
	if certFile != "" && keyFile != "" {
		mode = "HTTPS"
	}
	cfg.logger.Info("starting server", "mode", mode, "addr", ln.Addr().String())

	quit := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)

		sig := <-sigint
		cfg.logger.Info("signal received, shutting down server", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			cfg.logger.Error("error during server shutdown", "error", err)
		}

		close(quit)
	}()

	var err error
	if certFile != "" && keyFile != "" {
		err = srv.ServeTLS(ln, certFile, keyFile)
	} else {
		err = srv.Serve(ln)
	}

	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-quit
	return nil
}
