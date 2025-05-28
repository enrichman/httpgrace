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
	signals         []os.Signal
	serverOptions   []ServerOption
}

// ServerOption configures the underlying http.Server
type ServerOption func(*http.Server)

func defaultConfig() serverConfig {
	return serverConfig{
		shutdownTimeout: 10 * time.Second,
		logger:          slog.Default(),
		signals:         []os.Signal{syscall.SIGINT, syscall.SIGTERM},
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

// WithSignals sets which OS signals trigger graceful shutdown.
func WithSignals(signals ...os.Signal) Option {
	return func(cfg *serverConfig) {
		if len(signals) > 0 {
			cfg.signals = signals
		}
	}
}

// WithServerOptions allows configuring the underlying http.Server.
func WithServerOptions(opts ...ServerOption) Option {
	return func(cfg *serverConfig) {
		cfg.serverOptions = append(cfg.serverOptions, opts...)
	}
}

// Server option helpers
func WithReadTimeout(d time.Duration) ServerOption {
	return func(srv *http.Server) { srv.ReadTimeout = d }
}

func WithWriteTimeout(d time.Duration) ServerOption {
	return func(srv *http.Server) { srv.WriteTimeout = d }
}

func WithIdleTimeout(d time.Duration) ServerOption {
	return func(srv *http.Server) { srv.IdleTimeout = d }
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

// Server wraps http.Server with built-in graceful shutdown capabilities.
type Server struct {
	*http.Server
	config serverConfig
}

// NewServer creates a new Server with graceful shutdown capabilities.
func NewServer(handler http.Handler, opts ...Option) *Server {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	srv := &http.Server{
		Handler: handler,
	}

	// Apply server options
	for _, opt := range cfg.serverOptions {
		opt(srv)
	}

	return &Server{
		Server: srv,
		config: cfg,
	}
}

// ListenAndServe starts the server with graceful shutdown on the given address.
func (s *Server) ListenAndServe(addr string) error {
	s.Server.Addr = addr
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.serve(ln, "", "")
}

// ListenAndServeTLS starts the TLS server with graceful shutdown.
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	s.Server.Addr = addr
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.serve(ln, certFile, keyFile)
}

// Serve starts the server on the given listener.
func (s *Server) Serve(ln net.Listener) error {
	return s.serve(ln, "", "")
}

// ServeTLS starts the TLS server on the given listener.
func (s *Server) ServeTLS(ln net.Listener, certFile, keyFile string) error {
	return s.serve(ln, certFile, keyFile)
}

func (s *Server) serve(ln net.Listener, certFile, keyFile string) error {
	quit := make(chan error)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, s.config.signals...)
	defer signal.Stop(sigChan)

	// Start shutdown handler
	go s.handleShutdown(sigChan, quit)

	// Log server start
	mode := "HTTP"
	if certFile != "" && keyFile != "" {
		mode = "HTTPS"
	}
	s.config.logger.Info("starting server",
		"mode", mode,
		"addr", ln.Addr().String(),
		"shutdown_timeout", s.config.shutdownTimeout)

	// Start server
	var err error
	if certFile != "" && keyFile != "" {
		err = s.Server.ServeTLS(ln, certFile, keyFile)
	} else {
		err = s.Server.Serve(ln)
	}

	// Handle server errors
	if err != nil && err != http.ErrServerClosed {
		s.config.logger.Error("server error", "error", err)
		return err
	}

	// Wait for graceful shutdown to complete and return any shutdown error
	shutdownErr := <-quit
	return shutdownErr
}

func (s *Server) handleShutdown(sigChan <-chan os.Signal, quit chan<- error) { // Changed from chan<- struct{} to chan<- error
	defer close(quit)

	sig := <-sigChan
	s.config.logger.Info("shutdown signal received", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), s.config.shutdownTimeout)
	defer cancel()

	shutdownStart := time.Now()
	err := s.Server.Shutdown(ctx)
	if err != nil {
		s.config.logger.Error(
			"server shutdown failed",
			"error", err,
			"timeout", s.config.shutdownTimeout,
			"duration", time.Since(shutdownStart),
		)
	} else {
		s.config.logger.Info(
			"server shutdown completed gracefully",
			"duration", time.Since(shutdownStart),
		)
	}
	quit <- err
}

// Internal implementation for backwards compatibility
func listenAndServeInternal(addr, certFile, keyFile string, handler http.Handler, opts ...Option) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return serveInternal(ln, certFile, keyFile, handler, opts...)
}

func serveInternal(ln net.Listener, certFile, keyFile string, handler http.Handler, opts ...Option) error {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	srv := &http.Server{
		Handler: handler,
	}

	// Apply server options
	for _, opt := range cfg.serverOptions {
		opt(srv)
	}

	server := &Server{
		Server: srv,
		config: cfg,
	}

	return server.serve(ln, certFile, keyFile)
}
