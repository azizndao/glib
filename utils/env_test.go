package utils

import (
	"os"
	"testing"
	"time"
)

func TestGetEnv(t *testing.T) {
	key := "TEST_VAR"
	expectedValue := "test-value"

	// Set env var
	os.Setenv(key, expectedValue)
	defer os.Unsetenv(key)

	value, ok := GetEnv(key)
	if !ok || value != expectedValue {
		t.Errorf("GetEnv(%q) = %q, %v, want %q, true", key, value, ok, expectedValue)
	}

	// Test non-existing var
	value, ok = GetEnv("NON_EXISTING_VAR")
	if ok || value != "" {
		t.Errorf("GetEnv(NON_EXISTING_VAR) = %q, %v, want %q, false", value, ok, "")
	}
}

func TestGetEnvOr(t *testing.T) {
	key := "TEST_VAR_OR"
	fallback := "fallback-value"

	// Test with existing var
	os.Setenv(key, "actual-value")
	defer os.Unsetenv(key)

	value := GetEnvOr(key, fallback)
	if value != "actual-value" {
		t.Errorf("GetEnvOr(%q, %q) = %q, want %q", key, fallback, value, "actual-value")
	}

	// Test with non-existing var
	value = GetEnvOr("NON_EXISTING", fallback)
	if value != fallback {
		t.Errorf("GetEnvOr(NON_EXISTING, %q) = %q, want %q", fallback, value, fallback)
	}
}

func TestGetEnvSlice(t *testing.T) {
	key := "TEST_SLICE"

	// Test with comma-separated values
	os.Setenv(key, "a,b,c")
	defer os.Unsetenv(key)

	values := GetEnvSlice(key, "")
	if len(values) != 3 || values[0] != "a" || values[1] != "b" || values[2] != "c" {
		t.Errorf("GetEnvSlice(%q) = %v, want [a b c]", key, values)
	}

	// Test with whitespace
	os.Setenv(key, " x , y , z ")
	values = GetEnvSlice(key, "")
	if len(values) != 3 || values[0] != "x" || values[1] != "y" || values[2] != "z" {
		t.Errorf("GetEnvSlice(%q) = %v, want [x y z]", key, values)
	}

	// Test with empty value (should use fallback)
	os.Unsetenv(key)
	values = GetEnvSlice(key, "default,values")
	if len(values) != 2 || values[0] != "default" || values[1] != "values" {
		t.Errorf("GetEnvSlice(%q) = %v, want [default values]", key, values)
	}

	// Test with no value and no fallback
	values = GetEnvSlice("MISSING_VAR", "")
	if len(values) != 0 {
		t.Errorf("GetEnvSlice(MISSING_VAR) = %v, want []", values)
	}
}

func TestGetEnvInt(t *testing.T) {
	key := "TEST_INT"

	// Valid int
	os.Setenv(key, "123")
	defer os.Unsetenv(key)

	value, err := GetEnvInt(key, 0)
	if err != nil {
		t.Errorf("GetEnvInt(%q, 0) unexpected error: %v", key, err)
	}
	if value != 123 {
		t.Errorf("GetEnvInt(%q, 0) = %d, want 123", key, value)
	}

	// Test fallback
	value, err = GetEnvInt("MISSING_INT", 42)
	if err != nil {
		t.Errorf("GetEnvInt(MISSING_INT, 42) unexpected error: %v", err)
	}
	if value != 42 {
		t.Errorf("GetEnvInt(MISSING_INT, 42) = %d, want 42", value)
	}

	// Invalid int
	os.Setenv(key, "not-a-number")
	_, err = GetEnvInt(key, 0)
	if err == nil {
		t.Errorf("GetEnvInt(%q, 0) expected error for invalid int", key)
	}
}

func TestGetEnvBool(t *testing.T) {
	key := "TEST_BOOL"

	tests := []struct {
		value   string
		want    bool
		wantErr bool
	}{
		{"true", true, false},
		{"false", false, false},
		{"1", true, false},
		{"0", false, false},
		{"invalid", false, true},
	}

	for _, tt := range tests {
		os.Setenv(key, tt.value)
		got, err := GetEnvBool(key, false)
		if (err != nil) != tt.wantErr {
			t.Errorf("GetEnvBool(%q=%q, false) error = %v, wantErr %v", key, tt.value, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("GetEnvBool(%q=%q, false) = %v, want %v", key, tt.value, got, tt.want)
		}
	}

	os.Unsetenv(key)

	// Test fallback
	value, err := GetEnvBool("MISSING_BOOL", true)
	if err != nil {
		t.Errorf("GetEnvBool(MISSING_BOOL, true) unexpected error: %v", err)
	}
	if value != true {
		t.Errorf("GetEnvBool(MISSING_BOOL, true) = %v, want true", value)
	}
}

func TestGetEnvDuration(t *testing.T) {
	key := "TEST_DURATION"

	// Valid duration
	os.Setenv(key, "5s")
	defer os.Unsetenv(key)

	value, err := GetEnvDuration(key, 0)
	if err != nil {
		t.Errorf("GetEnvDuration(%q, 0) unexpected error: %v", key, err)
	}
	if value != 5*time.Second {
		t.Errorf("GetEnvDuration(%q, 0) = %v, want 5s", key, value)
	}

	// Test fallback
	value, err = GetEnvDuration("MISSING_DURATION", 10*time.Second)
	if err != nil {
		t.Errorf("GetEnvDuration(MISSING_DURATION, 10s) unexpected error: %v", err)
	}
	if value != 10*time.Second {
		t.Errorf("GetEnvDuration(MISSING_DURATION, 10s) = %v, want 10s", value)
	}

	// Invalid duration
	os.Setenv(key, "invalid")
	_, err = GetEnvDuration(key, 0)
	if err == nil {
		t.Errorf("GetEnvDuration(%q, 0) expected error for invalid duration", key)
	}
}
