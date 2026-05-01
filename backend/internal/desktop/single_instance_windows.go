//go:build windows

package desktop

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// InstanceLock holds a Windows named mutex for single-instance enforcement.
type InstanceLock struct {
	handle windows.Handle
}

// AcquireSingleInstance creates or opens a named Windows mutex, returning false if another instance already holds it.
func AcquireSingleInstance(name string) (*InstanceLock, bool, error) {
	if name == "" {
		return nil, false, fmt.Errorf("empty mutex name")
	}
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, err
	}
	handle, err := windows.CreateMutex(nil, false, namePtr)
	if err != nil {
		return nil, false, err
	}
	lastErr := windows.GetLastError()
	alreadyRunning := lastErr == windows.ERROR_ALREADY_EXISTS
	lock := &InstanceLock{handle: handle}
	if alreadyRunning {
		return lock, false, nil
	}
	_ = unsafe.Pointer(nil)
	return lock, true, nil
}

// Release closes the Windows mutex handle.
func (l *InstanceLock) Release() error {
	if l == nil || l.handle == 0 {
		return nil
	}
	return windows.CloseHandle(l.handle)
}
