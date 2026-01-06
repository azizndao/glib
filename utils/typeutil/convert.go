package typeutil

import (
	"encoding/json"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 512)
	},
}

// Convert anything into the desired type using JSON marshaling and unmarshaling.
func Convert[T any](data any) (T, error) {
	var result T

	// Fast path: if data is already the target type
	if v, ok := data.(T); ok {
		return v, nil
	}

	// Marshal to JSON
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf[:0]) // Reset and return to pool

	jsonData, err := json.Marshal(data)
	if err != nil {
		return result, err
	}

	// Unmarshal to target type
	err = json.Unmarshal(jsonData, &result)
	return result, err
}

// MustConvert anything into the desired type using JSON marshaling and unmarshaling.
// Panics if it fails.
func MustConvert[T any](data any) T {
	res, err := Convert[T](data)
	if err != nil {
		panic(err)
	}
	return res
}
