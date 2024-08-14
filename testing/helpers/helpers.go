package helpers

// Must takes return values from a function and returns the non-error one. If
// the error value is non-nil then it panics.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
