package util

func FirstOrDefault[T any](values []T, defaultValue func() T) T {
	if len(values) > 0 {
		return values[0]
	}
	return defaultValue()
}
