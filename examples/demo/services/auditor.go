package services

import (
	"log/slog"
	"time"
)

// Auditor logs user actions for audit trails
type Auditor struct {
	logger      *slog.Logger
	userService *UserSerivce
	createdAt   time.Time
}

// NewAuditor creates a new auditor instance
// This is transient - fresh instance per injection
// It depends on singleton UserSerivce
// @Provider transient
func NewAuditor(userService *UserSerivce) *Auditor {
	return &Auditor{
		logger:      slog.Default(),
		userService: userService,
		createdAt:   time.Now(),
	}
}

// LogAction logs an action for audit
func (a *Auditor) LogAction(action string) {
	a.logger.Info("audit log",
		"action", action,
		"timestamp", a.createdAt.Format(time.RFC3339),
	)
}
