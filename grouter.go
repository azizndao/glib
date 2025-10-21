package grouter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gerrors "github.com/azizndao/grouter/errors"
	"github.com/azizndao/grouter/middleware"
	"github.com/azizndao/grouter/ratelimit"
	"github.com/azizndao/grouter/router"
	gslog "github.com/azizndao/grouter/slog"
	"github.com/azizndao/grouter/util"
	"github.com/azizndao/grouter/validation"
	"github.com/joho/godotenv"
)

// Server represents the main grouter HTTP server with integrated middleware and lifecycle management
type Server struct {
	router          router.Router
	httpServer      *http.Server
	logger          *gslog.Logger
	shutdownTimeout time.Duration
	stores          []ratelimit.Store // Track stores for cleanup
}

// New creates a new Server with configuration loaded from environment variables
// All configuration is loaded via env vars - see .env.example for available options
//
// Parameters:
//   - locales: Optional validation locale configurations for i18n support
//     Pass validation.LocaleConfig for multi-language validation error messages
//     Example: New(validation.Locale(fr.New(), fr_translations.RegisterDefaultTranslations))
func New(locales ...validation.LocaleConfig) *Server {
	// Load server settings from env
	godotenv.Load()
	host := util.GetEnv("HOST", "localhost")
	port := util.GetEnvInt("PORT", 8080)
	readTimeout := util.GetEnvDuration("READ_TIMEOUT", 10*time.Second)
	writeTimeout := util.GetEnvDuration("WRITE_TIMEOUT", 10*time.Second)
	idleTimeout := util.GetEnvDuration("IDLE_TIMEOUT", 120*time.Second)
	shutdownTimeout := util.GetEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second)

	// Create logger from environment configuration
	logger := gslog.Create()

	// Create router with default options
	r := router.Default()

	// Build and apply middleware stack from environment variables
	r.Use(middleware.Stack(locales...)...)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", host, port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      r.Handler(),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	return &Server{
		router:          r,
		httpServer:      httpServer,
		logger:          logger,
		shutdownTimeout: shutdownTimeout,
		stores:          make([]ratelimit.Store, 0),
	}
}

// Router returns the underlying router for advanced configuration
func (s *Server) Router() router.Router {
	return s.router
}

// Logger returns the configured logger
func (s *Server) Logger() *gslog.Logger {
	return s.logger
}

// Address returns the server address (host:port)
func (s *Server) Address() string {
	return s.httpServer.Addr
}

// Listen starts the HTTP server
// Returns an error if the server fails to start
func (s *Server) Listen() error {
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return gerrors.Errorf("server failed to start: %w", err)
	}

	return nil
}

// ListenTLS starts the HTTPS server with TLS
func (s *Server) ListenTLS(certFile, keyFile string) error {
	s.logger.InfoWithSource(context.Background(), 0, "Starting TLS server",
		"address", s.httpServer.Addr,
		"cert", certFile,
	)

	if err := s.httpServer.ListenAndServeTLS(certFile, keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("TLS server failed to start: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server without interrupting active connections
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.InfoWithSource(ctx, 0, "Shutting down server")

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.ErrorWithSource(ctx, 0, fmt.Errorf("server shutdown failed: %w", err))
		return err
	}

	// Cleanup stores (rate limiters, etc.)
	for _, store := range s.stores {
		if err := store.Close(); err != nil {
			s.logger.ErrorWithSource(ctx, 0, fmt.Errorf("failed to close store: %w", err))
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
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		s.logger.InfoWithSource(context.Background(), 0, "Received shutdown signal",
			"signal", sig.String(),
		)

		// Create context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		// Attempt graceful shutdown
		if err := s.Shutdown(ctx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
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
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		s.logger.InfoWithSource(context.Background(), 0, "Received shutdown signal",
			"signal", sig.String(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
	}

	return nil
}

// RegisterStore registers a rate limit store for cleanup on shutdown
// This is useful if you're using custom rate limit stores
func (s *Server) RegisterStore(store ratelimit.Store) {
	s.stores = append(s.stores, store)
}
