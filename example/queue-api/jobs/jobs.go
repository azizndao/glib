package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/azizndao/glib/queue"
)

// SendEmailJob represents a job to send an email.
type SendEmailJob struct {
	queue.BaseJob
	To      string
	Subject string
	Body    string
}

// Handle executes the job logic.
func (j *SendEmailJob) Handle(ctx context.Context) error {
	slog.Info("Sending email",
		"to", j.To,
		"subject", j.Subject,
	)

	// Simulate email sending
	time.Sleep(2 * time.Second)

	slog.Info("Email sent successfully", "to", j.To)
	return nil
}

// Queue specifies which queue this job should be pushed to.
func (j *SendEmailJob) Queue() string {
	return "emails"
}

// Tries returns the maximum number of retry attempts.
func (j *SendEmailJob) Tries() int {
	return 3
}

// Failed is called when the job fails after all retries.
func (j *SendEmailJob) Failed(ctx context.Context, err error) {
	slog.Error("Failed to send email after retries",
		"to", j.To,
		"error", err,
	)
}

// ProcessVideoJob represents a job to process a video.
type ProcessVideoJob struct {
	queue.BaseJob
	VideoID int
	UserID  int
}

// Handle executes the video processing logic.
func (j *ProcessVideoJob) Handle(ctx context.Context) error {
	slog.Info("Processing video",
		"video_id", j.VideoID,
		"user_id", j.UserID,
	)

	// Simulate video processing
	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("video processing cancelled")
		case <-time.After(1 * time.Second):
			slog.Info("Processing progress", "progress", (i+1)*20)
		}
	}

	slog.Info("Video processed successfully", "video_id", j.VideoID)
	return nil
}

// Queue specifies the queue name.
func (j *ProcessVideoJob) Queue() string {
	return "videos"
}

// Tries returns the number of retry attempts.
func (j *ProcessVideoJob) Tries() int {
	return 2
}

// Timeout returns the maximum time the job can run.
func (j *ProcessVideoJob) Timeout() time.Duration {
	return 30 * time.Second
}

// Failed is called when the job fails.
func (j *ProcessVideoJob) Failed(ctx context.Context, err error) {
	slog.Error("Failed to process video",
		"video_id", j.VideoID,
		"error", err,
	)
}

// GenerateReportJob represents a job to generate a report.
type GenerateReportJob struct {
	queue.BaseJob
	ReportID   int
	ReportType string
}

// Handle executes the report generation logic.
func (j *GenerateReportJob) Handle(ctx context.Context) error {
	slog.Info("Generating report",
		"report_id", j.ReportID,
		"type", j.ReportType,
	)

	// Simulate report generation
	time.Sleep(3 * time.Second)

	slog.Info("Report generated successfully", "report_id", j.ReportID)
	return nil
}

// Queue specifies the queue name.
func (j *GenerateReportJob) Queue() string {
	return "default"
}
