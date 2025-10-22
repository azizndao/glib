package glib

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gerrors "github.com/azizndao/glib/errors"
	"github.com/azizndao/glib/middleware"
	"github.com/azizndao/glib/ratelimit"
	"github.com/azizndao/glib/router"
	glog "github.com/azizndao/glib/slog"
	"github.com/azizndao/glib/util"
	"github.com/azizndao/glib/validation"
	"github.com/joho/godotenv"
)

type LocaleConfig = validation.LocaleConfig

var Locale = validation.Locale

type Config struct {
	Locales []LocaleConfig
	Store   ratelimit.Store // Optional: Custom store for rate limiting (default: in-memory)
}

// Server represents the main glib HTTP server with integrated middleware and lifecycle management
type Server struct {
	router          router.Router
	httpServer      *http.Server
	logger          *glog.Logger
	shutdownTimeout time.Duration
	Stores          []ratelimit.Store // Track stores for cleanup
	Validator       *validation.Validator
}

// New creates a new Server with configuration loaded from environment variables
// All configuration is loaded via env vars - see .env.example for available options
//
// Parameters:
//   - locales: Optional validation locale configurations for i18n support
//     Pass validation.LocaleConfig for multi-language validation error messages
//     Example: New(validation.Locale(fr.New(), fr_translations.RegisterDefaultTranslations))
func New(config Config) *Server {
	// Load server settings from env
	godotenv.Load()
	host := util.GetEnv("HOST", "localhost")
	port := util.GetEnvInt("PORT", 8080)
	readTimeout := util.GetEnvDuration("READ_TIMEOUT", 10*time.Second)
	writeTimeout := util.GetEnvDuration("WRITE_TIMEOUT", 10*time.Second)
	idleTimeout := util.GetEnvDuration("IDLE_TIMEOUT", 120*time.Second)
	shutdownTimeout := util.GetEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second)

	// Create logger from environment configuration
	logger := glog.Create()

	slog.SetDefault(logger.Logger)

	// Create router with default options
	r := router.Default(logger)

	// Build and apply middleware stack from environment variables
	middlewareStack := middleware.Stack(middleware.StackConfig{
		Locales: config.Locales,
		Store:   config.Store,
	})
	r.Use(middlewareStack...)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", host, port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	validatorConfig := validation.ValidatorConfig{
		Logger:            logger,
		Locales:           config.Locales,
		UseJSONFieldNames: true,
		DefaultLocale:     "en",
	}

	server := &Server{
		router:          r,
		httpServer:      httpServer,
		logger:          logger,
		shutdownTimeout: shutdownTimeout,
		Stores:          make([]ratelimit.Store, 0),
		Validator:       validation.NewValidator(validatorConfig),
	}

	// Register custom store for cleanup if provided
	if config.Store != nil {
		server.RegisterStore(config.Store)
	}

	return server
}

// Router returns the underlying router for advanced configuration
func (s *Server) Router() router.Router {
	return s.router
}

// Logger returns the configured logger
func (s *Server) Logger() *glog.Logger {
	return s.logger
}

// Address returns the server address (host:port)
func (s *Server) Address() string {
	return s.httpServer.Addr
}

// Listen starts the HTTP server
// Returns an error if the server fails to start
func (s *Server) Listen() error {
	s.logger.InfoWithSource(context.Background(), 0, fmt.Sprintf("Starting server on %s", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return gerrors.Errorf("server failed to start: %w", err)
	}

	return nil
}

// ListenTLS starts the HTTPS server with TLS
func (s *Server) ListenTLS(certFile, keyFile string) error {
	s.logger.InfoWithSource(context.Background(), 0, fmt.Sprintf("Starting TLS server on %s", s.httpServer.Addr))

	if err := s.httpServer.ListenAndServeTLS(certFile, keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return gerrors.Errorf("TLS server failed to start: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server without interrupting active connections
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.InfoWithSource(ctx, 0, "Shutting down server")

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.ErrorWithSource(ctx, 0, gerrors.Errorf("server shutdown failed: %w", err))
		return err
	}

	// Cleanup stores (rate limiters, etc.)
	for _, store := range s.Stores {
		if err := store.Close(); err != nil {
			s.logger.ErrorWithSource(ctx, 0, gerrors.Errorf("failed to close store: %w", err))
		}
	}

	s.logger.InfoWithSource(ctx, 0, "Server stopped")
	return nil
}

// ListenWithGracefulShutdown starts the server and handles graceful shutdown on SIGINT/SIGTERM
// This is the recommended way to run the server in production
func (s *Server) ListenWithGracefulShutdown() error {
	// Create channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- s.Listen()
	}()

	// Wait for interrupt signal or server error
	select {
	case err := <-serverErrors:
		return gerrors.Errorf("server error: %w", err)
	case sig := <-quit:
		s.logger.InfoWithSource(context.Background(), 0, "Received shutdown signal",
			"signal", sig.String(),
		)

		// Create context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		// Attempt graceful shutdown
		if err := s.Shutdown(ctx); err != nil {
			return gerrors.Errorf("graceful shutdown failed: %w", err)
		}
	}

	return nil
}

// ListenTLSWithGracefulShutdown starts the TLS server and handles graceful shutdown
func (s *Server) ListenTLSWithGracefulShutdown(certFile, keyFile string) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- s.ListenTLS(certFile, keyFile)
	}()

	select {
	case err := <-serverErrors:
		return gerrors.Errorf("server error: %w", err)
	case sig := <-quit:
		s.logger.InfoWithSource(context.Background(), 0, "Received shutdown signal",
			"signal", sig.String(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			return gerrors.Errorf("graceful shutdown failed: %w", err)
		}
	}

	return nil
}

// RegisterStore registers a rate limit store for cleanup on shutdown
// This is useful if you're using custom rate limit stores
func (s *Server) RegisterStore(store ratelimit.Store) {
	s.Stores = append(s.Stores, store)
}
