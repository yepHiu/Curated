//go:build !windows

package desktop

// InstanceLock represents a no-op single-instance lock on non-Windows platforms.
type InstanceLock struct{}

// AcquireSingleInstance always succeeds on non-Windows platforms.
func AcquireSingleInstance(_ string) (*InstanceLock, bool, error) {
	return &InstanceLock{}, true, nil
}

// Release is a no-op on non-Windows platforms.
func (l *InstanceLock) Release() error {
	return nil
}
