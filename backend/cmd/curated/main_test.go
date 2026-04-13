package main

import (
	"context"
	"testing"

	"curated-backend/internal/config"
	"curated-backend/internal/desktop"
)

func TestRunTrayMode_SilentAutostartSkipsInitialBrowserOpen(t *testing.T) {
	restore := stubTrayModeDependencies()
	defer restore()

	openCount := 0
	openURLFn = func(_ context.Context, _ string) error {
		openCount += 1
		return nil
	}

	boot := &bootstrap{cfg: config.Config{HttpAddr: ":18080"}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := runTrayMode(ctx, cancel, boot, trayModeOptions{silentInitialBrowser: true}); err != nil {
		t.Fatal(err)
	}
	if openCount != 0 {
		t.Fatalf("openURL count = %d, want 0", openCount)
	}
}

func TestRunTrayMode_ManualLaunchOpensInitialBrowserOnce(t *testing.T) {
	restore := stubTrayModeDependencies()
	defer restore()

	openCount := 0
	openURLFn = func(_ context.Context, _ string) error {
		openCount += 1
		return nil
	}

	boot := &bootstrap{cfg: config.Config{HttpAddr: ":18080"}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := runTrayMode(ctx, cancel, boot, trayModeOptions{}); err != nil {
		t.Fatal(err)
	}
	if openCount != 1 {
		t.Fatalf("openURL count = %d, want 1", openCount)
	}
}

func stubTrayModeDependencies() func() {
	prevAcquire := acquireSingleInstanceFn
	prevRunHTTP := runHTTPFn
	prevWaitForServer := waitForServerOrExitFn
	prevRunTray := runTrayFn
	prevOpenURL := openURLFn

	acquireSingleInstanceFn = func(string) (*desktop.InstanceLock, bool, error) {
		return &desktop.InstanceLock{}, true, nil
	}
	runHTTPFn = func(context.Context, *bootstrap) error {
		return nil
	}
	waitForServerOrExitFn = func(context.Context, <-chan error, string) error {
		return nil
	}
	runTrayFn = func(context.Context, desktop.TrayOptions) error {
		return nil
	}
	openURLFn = func(context.Context, string) error {
		return nil
	}

	return func() {
		acquireSingleInstanceFn = prevAcquire
		runHTTPFn = prevRunHTTP
		waitForServerOrExitFn = prevWaitForServer
		runTrayFn = prevRunTray
		openURLFn = prevOpenURL
	}
}
