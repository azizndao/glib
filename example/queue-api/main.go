package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/azizndao/glib/example/queue-api/jobs"
	"github.com/azizndao/glib/queue"
	"github.com/azizndao/glib/queue/drivers/redis"
)

func main() {
	// Set up structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Get mode from environment (server or worker)
	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "server"
	}

	// Create queue manager
	manager := setupQueue()

	// Register job types
	queue.Register(&jobs.SendEmailJob{})
	queue.Register(&jobs.ProcessVideoJob{})
	queue.Register(&jobs.GenerateReportJob{})

	if mode == "worker" {
		// Run as worker
		runWorker(manager)
	} else {
		// Run as API server
		runServer(manager)
	}
}

func setupQueue() *queue.Manager {
	// Create manager
	manager := queue.NewManager()

	// Register Redis driver (using Asynq)
	manager.RegisterDriver("redis", redis.New)

	// Register default connection
	manager.RegisterConfig("default", queue.Config{
		Driver: "redis",
		Connection: map[string]any{
			"addr":     getEnv("REDIS_ADDR", "localhost:6379"),
			"password": getEnv("REDIS_PASSWORD", ""),
			"db":       0,
		},
		Queue: "default",
	})

	// Set as global default
	queue.SetDefaultManager(manager)

	return manager
}

func runServer(manager *queue.Manager) {
	slog.Info("Starting API server on :8080")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Queue API Example\n")
		fmt.Fprintf(w, "\nEndpoints:\n")
		fmt.Fprintf(w, "  POST /send-email - Dispatch email job\n")
		fmt.Fprintf(w, "  POST /process-video - Dispatch video processing job\n")
		fmt.Fprintf(w, "  POST /generate-report - Dispatch report generation job\n")
		fmt.Fprintf(w, "  GET /stats - Get queue statistics\n")
	})

	// Endpoint to dispatch email job
	http.HandleFunc("/send-email", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		job := &jobs.SendEmailJob{
			To:      r.URL.Query().Get("to"),
			Subject: r.URL.Query().Get("subject"),
			Body:    r.URL.Query().Get("body"),
		}

		if job.To == "" {
			job.To = "user@example.com"
		}
		if job.Subject == "" {
			job.Subject = "Test Email"
		}
		if job.Body == "" {
			job.Body = "This is a test email from the queue system!"
		}

		// Dispatch the job
		id, err := queue.Dispatch(job).Dispatch()
		if err != nil {
			slog.Error("Failed to dispatch email job", "error", err)
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Email job dispatched successfully!\nJob ID: %s\n", id)
	})

	// Endpoint to dispatch video processing job
	http.HandleFunc("/process-video", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		job := &jobs.ProcessVideoJob{
			VideoID: 123,
			UserID:  456,
		}

		// Dispatch with delay
		id, err := queue.Dispatch(job).
			OnQueue("videos").
			Delay(5 * time.Second).
			Dispatch()
		if err != nil {
			slog.Error("Failed to dispatch video job", "error", err)
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Video processing job dispatched (will start in 5 seconds)!\nJob ID: %s\n", id)
	})

	// Endpoint to dispatch report generation job
	http.HandleFunc("/generate-report", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		job := &jobs.GenerateReportJob{
			ReportID:   789,
			ReportType: "monthly",
		}

		id, err := queue.Dispatch(job).Dispatch()
		if err != nil {
			slog.Error("Failed to dispatch report job", "error", err)
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Report generation job dispatched!\nJob ID: %s\n", id)
	})

	// Endpoint to get queue statistics
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		q, err := manager.Default()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}

		stats, err := q.Stats(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Queue Statistics:\n\n")
		for _, stat := range stats {
			fmt.Fprintf(w, "Queue: %s\n", stat.Queue)
			fmt.Fprintf(w, "  Pending: %d\n", stat.Pending)
			fmt.Fprintf(w, "  Active: %d\n", stat.Active)
			fmt.Fprintf(w, "  Scheduled: %d\n", stat.Scheduled)
			fmt.Fprintf(w, "  Retry: %d\n", stat.Retry)
			fmt.Fprintf(w, "  Failed: %d\n", stat.Failed)
			fmt.Fprintf(w, "  Processed: %d\n", stat.Processed)
			fmt.Fprintf(w, "\n")
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func runWorker(manager *queue.Manager) {
	slog.Info("Starting queue worker")

	// Create worker config
	config := queue.WorkerConfig{
		Connection:  "default",
		Concurrency: 5,
		Queues: map[string]int{
			"emails":  3, // Higher priority
			"videos":  2,
			"default": 1, // Lower priority
		},
		StrictPriority: false,
		Logger:         slog.Default(),
	}

	// Create worker
	worker := queue.NewWorker(manager, config)

	// Register jobs
	worker.RegisterJobs(
		&jobs.SendEmailJob{},
		&jobs.ProcessVideoJob{},
		&jobs.GenerateReportJob{},
	)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start worker in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := worker.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	slog.Info("Worker started successfully, press Ctrl+C to stop")

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		slog.Info("Received shutdown signal, stopping worker...")
		cancel()
		if err := worker.Stop(); err != nil {
			slog.Error("Error stopping worker", "error", err)
		}
	case err := <-errChan:
		slog.Error("Worker error", "error", err)
	}

	slog.Info("Worker stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
