package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	result := Success("test message")
	if !strings.Contains(result, "test message") {
		t.Errorf("Success() should contain message, got: %s", result)
	}
	if !strings.Contains(result, IconSuccess) {
		t.Errorf("Success() should contain success icon, got: %s", result)
	}
	if !strings.HasPrefix(result, Green) {
		t.Errorf("Success() should start with green color")
	}
	if !strings.HasSuffix(result, Reset) {
		t.Errorf("Success() should end with reset")
	}
}

func TestSuccessf(t *testing.T) {
	result := Successf("test %s %d", "message", 42)
	expected := "test message 42"
	if !strings.Contains(result, expected) {
		t.Errorf("Successf() should contain formatted message, got: %s", result)
	}
}

func TestError(t *testing.T) {
	result := Error("error message")
	if !strings.Contains(result, "error message") {
		t.Errorf("Error() should contain message, got: %s", result)
	}
	if !strings.Contains(result, IconError) {
		t.Errorf("Error() should contain error icon")
	}
	if !strings.HasPrefix(result, Red) {
		t.Errorf("Error() should start with red color")
	}
	if !strings.HasSuffix(result, Reset) {
		t.Errorf("Error() should end with reset")
	}
}

func TestErrorf(t *testing.T) {
	result := Errorf("error %s", "test")
	if !strings.Contains(result, "error test") {
		t.Errorf("Errorf() should contain formatted message, got: %s", result)
	}
}

func TestWarning(t *testing.T) {
	result := Warning("warning message")
	if !strings.Contains(result, "warning message") {
		t.Errorf("Warning() should contain message")
	}
	if !strings.Contains(result, IconWarning) {
		t.Errorf("Warning() should contain warning icon")
	}
	if !strings.HasPrefix(result, Yellow) {
		t.Errorf("Warning() should start with yellow color")
	}
}

func TestWarningf(t *testing.T) {
	result := Warningf("warning %d", 123)
	if !strings.Contains(result, "warning 123") {
		t.Errorf("Warningf() should contain formatted message")
	}
}

func TestInfo(t *testing.T) {
	result := Info("info message")
	if !strings.Contains(result, "info message") {
		t.Errorf("Info() should contain message")
	}
	if !strings.Contains(result, IconInfo) {
		t.Errorf("Info() should contain info icon")
	}
	if !strings.HasPrefix(result, Cyan) {
		t.Errorf("Info() should start with cyan color")
	}
}

func TestInfof(t *testing.T) {
	result := Infof("info %s", "test")
	if !strings.Contains(result, "info test") {
		t.Errorf("Infof() should contain formatted message")
	}
}

func TestMuted(t *testing.T) {
	result := Muted("muted text")
	if !strings.Contains(result, "muted text") {
		t.Errorf("Muted() should contain message")
	}
	if !strings.HasPrefix(result, Gray) {
		t.Errorf("Muted() should start with gray color")
	}
	if !strings.HasSuffix(result, Reset) {
		t.Errorf("Muted() should end with reset")
	}
}

func TestMutedf(t *testing.T) {
	result := Mutedf("muted %s", "text")
	if !strings.Contains(result, "muted text") {
		t.Errorf("Mutedf() should contain formatted message")
	}
}

func TestPrimary(t *testing.T) {
	result := Primary("primary text")
	if !strings.Contains(result, "primary text") {
		t.Errorf("Primary() should contain message")
	}
	if !strings.HasPrefix(result, Blue) {
		t.Errorf("Primary() should start with blue color")
	}
}

func TestPrimaryf(t *testing.T) {
	result := Primaryf("primary %s", "text")
	if !strings.Contains(result, "primary text") {
		t.Errorf("Primaryf() should contain formatted message")
	}
}

func TestBoldText(t *testing.T) {
	result := BoldText("bold text")
	if !strings.Contains(result, "bold text") {
		t.Errorf("BoldText() should contain message")
	}
	if !strings.HasPrefix(result, Bold) {
		t.Errorf("BoldText() should start with bold")
	}
	if !strings.HasSuffix(result, Reset) {
		t.Errorf("BoldText() should end with reset")
	}
}

func TestBoldTextf(t *testing.T) {
	result := BoldTextf("bold %s", "text")
	if !strings.Contains(result, "bold text") {
		t.Errorf("BoldTextf() should contain formatted message")
	}
}

func TestColorConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"Reset", Reset},
		{"Bold", Bold},
		{"Red", Red},
		{"Green", Green},
		{"Yellow", Yellow},
		{"Blue", Blue},
		{"Cyan", Cyan},
		{"Gray", Gray},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Errorf("%s constant should not be empty", tt.name)
			}
			if !strings.HasPrefix(tt.constant, "\033") {
				t.Errorf("%s should be ANSI escape sequence, got: %s", tt.name, tt.constant)
			}
		})
	}
}

func TestIconConstants(t *testing.T) {
	tests := []struct {
		name string
		icon string
	}{
		{"IconSuccess", IconSuccess},
		{"IconError", IconError},
		{"IconWarning", IconWarning},
		{"IconInfo", IconInfo},
		{"IconController", IconController},
		{"IconProvider", IconProvider},
		{"IconMiddleware", IconMiddleware},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.icon == "" {
				t.Errorf("%s constant should not be empty", tt.name)
			}
		})
	}
}

func TestNewRenderer(t *testing.T) {
	renderer := NewRenderer()
	if renderer == nil {
		t.Fatal("NewRenderer() should not return nil")
	}
	if renderer.output == nil {
		t.Error("Renderer output should not be nil")
	}
	// Note: IsTTY() depends on os.Stdout, so we just verify it returns a bool
	_ = renderer.IsTTY()
}

func TestRenderer_IsTTY(t *testing.T) {
	renderer := &Renderer{
		isTTY:  true,
		output: &bytes.Buffer{},
	}

	if !renderer.IsTTY() {
		t.Error("IsTTY() should return true when isTTY is true")
	}

	renderer.isTTY = false
	if renderer.IsTTY() {
		t.Error("IsTTY() should return false when isTTY is false")
	}
}

func TestRenderer_Render_WithTTY(t *testing.T) {
	renderer := &Renderer{
		isTTY:  true,
		output: &bytes.Buffer{},
	}

	styled := "\033[32mcolored\033[0m"
	plain := "colored"

	result := renderer.Render(styled, plain)
	if result != styled {
		t.Errorf("Render() with TTY should return styled text, got: %s", result)
	}
}

func TestRenderer_Render_WithoutTTY(t *testing.T) {
	renderer := &Renderer{
		isTTY:  false,
		output: &bytes.Buffer{},
	}

	styled := "\033[32mcolored\033[0m"
	plain := "colored"

	result := renderer.Render(styled, plain)
	if result != plain {
		t.Errorf("Render() without TTY should return plain text, got: %s", result)
	}
}

func TestRenderer_Render_Integration(t *testing.T) {
	tests := []struct {
		name   string
		isTTY  bool
		styled string
		plain  string
		want   string
	}{
		{
			name:   "TTY renders styled",
			isTTY:  true,
			styled: Green + "success" + Reset,
			plain:  "success",
			want:   Green + "success" + Reset,
		},
		{
			name:   "Non-TTY renders plain",
			isTTY:  false,
			styled: Red + "error" + Reset,
			plain:  "error",
			want:   "error",
		},
		{
			name:   "Empty strings",
			isTTY:  true,
			styled: "",
			plain:  "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := &Renderer{
				isTTY:  tt.isTTY,
				output: &bytes.Buffer{},
			}
			got := renderer.Render(tt.styled, tt.plain)
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}
