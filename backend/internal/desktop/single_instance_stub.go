//go:build !windows

package desktop

type InstanceLock struct{}

func AcquireSingleInstance(_ string) (*InstanceLock, bool, error) {
	return &InstanceLock{}, true, nil
}

func (l *InstanceLock) Release() error {
	return nil
}
