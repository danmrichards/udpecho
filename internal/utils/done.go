package utils

// IsDone returns true if channel c has been closed.
func IsDone(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}
