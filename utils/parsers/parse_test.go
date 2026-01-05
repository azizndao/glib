package parsers

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test soft-parse functions (used for query/header parameters)
func TestParseIntOrZero(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{"valid int", "42", 42},
		{"negative int", "-10", -10},
		{"zero", "0", 0},
		{"invalid int returns 0", "abc", 0},
		{"empty string returns 0", "", 0},
		{"float returns 0", "3.14", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseIntOrZero(tt.value)
			if got != tt.want {
				t.Errorf("ParseIntOrZero(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseInt64OrZero(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int64
	}{
		{"valid int64", "9223372036854775807", 9223372036854775807},
		{"negative int64", "-123456789", -123456789},
		{"invalid int64 returns 0", "abc", 0},
		{"empty string returns 0", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseInt64OrZero(tt.value)
			if got != tt.want {
				t.Errorf("ParseInt64OrZero(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseUint64OrZero(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  uint64
	}{
		{"valid uint64", "12345", 12345},
		{"zero", "0", 0},
		{"invalid uint64 returns 0", "abc", 0},
		{"negative returns 0", "-10", 0},
		{"empty string returns 0", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseUint64OrZero(tt.value)
			if got != tt.want {
				t.Errorf("ParseUint64OrZero(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseFloat64OrZero(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  float64
	}{
		{"valid float", "3.14", 3.14},
		{"negative float", "-2.5", -2.5},
		{"integer as float", "42", 42.0},
		{"invalid float returns 0", "abc", 0.0},
		{"empty string returns 0", "", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFloat64OrZero(tt.value)
			if got != tt.want {
				t.Errorf("ParseFloat64OrZero(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseBoolOrFalse(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true", "true", true},
		{"True", "True", true},
		{"1", "1", true},
		{"false", "false", false},
		{"False", "False", false},
		{"0", "0", false},
		{"invalid returns false", "abc", false},
		{"empty string returns false", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseBoolOrFalse(tt.value)
			if got != tt.want {
				t.Errorf("ParseBoolOrFalse(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseUUIDOrNil(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	parsed, _ := uuid.Parse(validUUID)

	tests := []struct {
		name  string
		value string
		want  uuid.UUID
	}{
		{"valid uuid", validUUID, parsed},
		{"invalid uuid returns Nil", "not-a-uuid", uuid.Nil},
		{"empty string returns Nil", "", uuid.Nil},
		{"partial uuid returns Nil", "550e8400", uuid.Nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseUUIDOrNil(tt.value)
			if got != tt.want {
				t.Errorf("ParseUUIDOrNil(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseDurationOrZero(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{"valid duration seconds", "5s", 5 * time.Second},
		{"valid duration minutes", "2m", 2 * time.Minute},
		{"valid duration mixed", "1h30m", 90 * time.Minute},
		{"invalid duration returns 0", "abc", 0},
		{"empty string returns 0", "", 0},
		{"number without unit returns 0", "123", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseDurationOrZero(tt.value)
			if got != tt.want {
				t.Errorf("ParseDurationOrZero(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseURLOrNil(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantNil bool
	}{
		{"valid https URL", "https://example.com", false},
		{"valid http URL", "http://example.com/path", false},
		{"relative URL", "/path/to/resource", false},
		// Note: url.Parse is very permissive and rarely returns errors
		// It only fails on truly invalid URLs with control characters
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseURLOrNil(tt.value)
			if tt.wantNil && got != nil {
				t.Errorf("ParseURLOrNil(%q) = %v, want nil", tt.value, got)
			} else if !tt.wantNil && got == nil {
				t.Errorf("ParseURLOrNil(%q) = nil, want non-nil", tt.value)
			}
		})
	}
}
