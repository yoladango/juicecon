package ptr

// Int returns a pointer to the given int value.
func Int(i int) *int {
	return &i
}
