package appupdate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInstallerProcessHelper(t *testing.T) {
	args := os.Args
	if len(args) < 3 || args[len(args)-2] != "review-installer-helper" {
		return
	}
	base := args[len(args)-1]
	if err := os.WriteFile(base+".ready", []byte("ready"), 0600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if err := os.WriteFile(base+".done", []byte("done"), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestInstallerSurvivesRequestCancellation(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(t.TempDir(), "installer")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := launchInstallerProcess(ctx, executable, []string{"-test.run=^TestInstallerProcessHelper$", "review-installer-helper", base}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(base + ".ready"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("helper did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	time.Sleep(750 * time.Millisecond)
	if _, err := os.Stat(base + ".done"); err != nil {
		t.Fatal("installer started successfully but was killed when the request context ended")
	}
}
