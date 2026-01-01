// Package ui provides UI helpers for the CLI
package ui

import (
	"io"
	"os"

	"golang.org/x/term"
)

type Renderer struct {
	isTTY  bool
	output io.Writer
}

func NewRenderer() *Renderer {
	return &Renderer{
		isTTY:  term.IsTerminal(int(os.Stdout.Fd())),
		output: os.Stdout,
	}
}

func (r *Renderer) IsTTY() bool {
	return r.isTTY
}

func (r *Renderer) Render(styled, plain string) string {
	if r.isTTY {
		return styled
	}
	return plain
}
