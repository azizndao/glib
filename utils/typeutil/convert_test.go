package typeutil

import (
	"testing"
)

type TestStruct struct {
	Name    string  `json:"name"`
	Age     int     `json:"age"`
	Email   string  `json:"email"`
	Active  bool    `json:"active"`
	Balance float64 `json:"balance"`
}

func TestConvertIssues(t *testing.T) {
	t.Run("buffer reuse issue", func(t *testing.T) {
		// The current implementation creates encoder and decoder on same buffer
		// This is inefficient - encoder writes, then decoder reads from same buffer
		data := map[string]any{"name": "test", "age": 30}
		result, err := Convert[map[string]any](data)
		if err != nil {
			t.Fatalf("Convert failed: %v", err)
		}
		t.Logf("Result: %+v", result)
	})

	t.Run("unnecessary marshal/unmarshal for simple types", func(t *testing.T) {
		// Converting int to int still goes through JSON encoding
		result, err := Convert[int](42)
		if err != nil {
			t.Fatalf("Convert failed: %v", err)
		}
		if result != 42 {
			t.Errorf("Expected 42, got %d", result)
		}
	})

	t.Run("memory allocation on every call", func(t *testing.T) {
		// Buffer is created fresh every time - could use sync.Pool
		data := TestStruct{Name: "John", Age: 30}
		for range 5 {
			_, _ = Convert[map[string]any](data)
		}
	})

	t.Run("encoder writes before decoder is created", func(t *testing.T) {
		// This works but is confusing - encoder and decoder share same buffer
		// Better to use separate buffers or Marshal/Unmarshal directly
		data := map[string]string{"key": "value"}
		result, err := Convert[map[string]any](data)
		if err != nil {
			t.Fatalf("Convert failed: %v", err)
		}
		t.Logf("Result: %+v", result)
	})
}

func TestConvertCorrectness(t *testing.T) {
	t.Run("map to struct", func(t *testing.T) {
		data := map[string]any{
			"name": "John",
			"age":  30,
		}
		result, err := Convert[TestStruct](data)
		if err != nil {
			t.Fatalf("Convert failed: %v", err)
		}
		if result.Name != "John" || result.Age != 30 {
			t.Errorf("Expected {John 30}, got %+v", result)
		}
	})

	t.Run("struct to struct", func(t *testing.T) {
		type Source struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		type Target struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		source := Source{Name: "John", Age: 30}
		result, err := Convert[Target](source)
		if err != nil {
			t.Fatalf("Convert failed: %v", err)
		}
		if result.Name != "John" || result.Age != 30 {
			t.Errorf("Expected {John 30}, got %+v", result)
		}
	})
}
