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

func TestNativeTrayRestartManagerShutdown(t *testing.T) {
	previousNotify, previousQuit := shellNotifyIconFn, postQuitMessageFn
	defer func() { shellNotifyIconFn, postQuitMessageFn = previousNotify, previousQuit }()
	removed, quit, cancelled := 0, 0, 0
	shellNotifyIconFn = func(uintptr, *notifyIconData) { removed++ }
	postQuitMessageFn = func(int32) { quit++ }
	tray := &nativeTrayRuntime{opts: TrayOptions{Cancel: func() { cancelled++ }}, trayMessageID: wmApp + 1, taskbarCreatedID: wmApp + 2}
	if result := tray.handleWindowMessage(0, wmQueryEndSession, 0, 1); result != 1 {
		t.Fatal("shutdown not accepted")
	}
	tray.handleWindowMessage(0, wmEndSession, 0, 1)
	if removed != 0 || quit != 0 || cancelled != 0 {
		t.Fatal("cancelled shutdown stopped Server")
	}
	tray.handleWindowMessage(0, wmEndSession, 1, 1)
	if removed != 1 || quit != 1 || cancelled != 1 {
		t.Fatal("confirmed shutdown did not stop Server")
	}
}
