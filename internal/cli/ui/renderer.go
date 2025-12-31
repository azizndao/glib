// Package ui provides UI helpers for the CLI
package ui

import (
	"io"
	"os"

	"golang.org/x/term"
)

// Renderer handles output rendering with TTY detection
type Renderer struct {
	isTTY  bool
	output io.Writer
}

// NewRenderer creates a new renderer with TTY detection
func NewRenderer() *Renderer {
	return &Renderer{
		isTTY:  term.IsTerminal(int(os.Stdout.Fd())),
		output: os.Stdout,
	}
}

// IsTTY returns whether output is a terminal
func (r *Renderer) IsTTY() bool {
	return r.isTTY
}

// Render returns styled output for TTY, plain text otherwise
func (r *Renderer) Render(styled, plain string) string {
	if r.isTTY {
		return styled
	}
	return plain
}
