package parsers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseInt(t *testing.T) {
	tests := []struct {
		value     string
		fieldName string
		want      int
		wantErr   bool
	}{
		{"123", "age", 123, false},
		{"0", "count", 0, false},
		{"-42", "balance", -42, false},
		{"abc", "age", 0, true},
		{"", "age", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseInt(tt.value, tt.fieldName)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseInt(%q, %q) error = %v, wantErr %v", tt.value, tt.fieldName, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseInt(%q, %q) = %v, want %v", tt.value, tt.fieldName, got, tt.want)
		}
	}
}

func TestParseUUID(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	_, err := ParseUUID(validUUID, "id")
	if err != nil {
		t.Errorf("ParseUUID(%q, %q) unexpected error: %v", validUUID, "id", err)
	}

	invalidUUID := "not-a-uuid"
	_, err = ParseUUID(invalidUUID, "id")
	if err == nil {
		t.Errorf("ParseUUID(%q, %q) expected error, got nil", invalidUUID, "id")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		value     string
		fieldName string
		wantErr   bool
	}{
		{"1s", "timeout", false},
		{"100ms", "delay", false},
		{"2h30m", "duration", false},
		{"invalid", "timeout", true},
	}

	for _, tt := range tests {
		_, err := ParseDuration(tt.value, tt.fieldName)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseDuration(%q, %q) error = %v, wantErr %v", tt.value, tt.fieldName, err, tt.wantErr)
		}
	}
}

func TestGetQuerySlice(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com?tags=a&tags=b&tags=c", nil)

	values := GetQuerySlice(req, "tags")
	if len(values) != 3 {
		t.Errorf("GetQuerySlice(tags) len = %d, want 3", len(values))
	}

	// Test non-existing parameter
	values = GetQuerySlice(req, "missing")
	if len(values) != 0 {
		t.Errorf("GetQuerySlice(missing) len = %d, want 0", len(values))
	}
}

func TestParseJSONBody(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	// Valid JSON
	body := strings.NewReader(`{"name":"alice","age":25}`)
	req := httptest.NewRequest("POST", "http://example.com", body)
	req.Header.Set("Content-Type", "application/json")

	result, err := ParseJSONBody[TestStruct](req)
	if err != nil {
		t.Errorf("ParseJSONBody() unexpected error: %v", err)
	}
	if result.Name != "alice" || result.Age != 25 {
		t.Errorf("ParseJSONBody() = %+v, want {Name:alice Age:25}", result)
	}

	// Invalid JSON
	body = strings.NewReader(`{invalid json}`)
	req = httptest.NewRequest("POST", "http://example.com", body)
	_, err = ParseJSONBody[TestStruct](req)
	if err == nil {
		t.Error("ParseJSONBody() expected error for invalid JSON, got nil")
	}
}
