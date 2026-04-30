//go:build !windows

package desktop

// ShowErrorDialog is a no-op on non-Windows platforms.
func ShowErrorDialog(_ string, _ string) {}
