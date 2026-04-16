//go:build windows

package desktop

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestNativeTrayRequestAppCancelOnlyRunsOnce(t *testing.T) {
	cancelCount := 0
	tray := &nativeTrayRuntime{
		opts: TrayOptions{
			Cancel: func() {
				cancelCount += 1
			},
		},
	}

	tray.requestAppCancel()
	tray.requestAppCancel()

	if cancelCount != 1 {
		t.Fatalf("cancel count = %d, want 1", cancelCount)
	}
}

func TestNativeTrayWindowCloseCancelsProcess(t *testing.T) {
	prevDestroyWindowFn := destroyWindowFn
	prevShellNotifyIconFn := shellNotifyIconFn
	defer func() {
		destroyWindowFn = prevDestroyWindowFn
		shellNotifyIconFn = prevShellNotifyIconFn
	}()

	destroyWindowFn = func(windows.Handle) {}
	shellNotifyIconFn = func(uintptr, *notifyIconData) {}

	cancelCount := 0
	tray := &nativeTrayRuntime{
		opts: TrayOptions{
			Cancel: func() {
				cancelCount += 1
			},
		},
		window: windows.Handle(1),
	}

	tray.handleWindowMessage(0, wmClose, 0, 0)

	if cancelCount != 1 {
		t.Fatalf("cancel count = %d, want 1", cancelCount)
	}
}
