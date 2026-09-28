//go:build windows

package installmigration

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// A real hidden Win32 window simulates the tray application's session shutdown
// contract. No installed Curated instance or registry key is used by this test.
func TestStopWindowFixture(t *testing.T) {
	ready := os.Getenv("CURATED_STOP_TEST_READY")
	if ready == "" {
		return
	}
	runtime.LockOSThread()
	user32 := windows.NewLazySystemDLL("user32.dll")
	callback := windows.NewCallback(func(hwnd uintptr, message uint32, wp, lp uintptr) uintptr {
		if message == 0x11 {
			if os.Getenv("CURATED_STOP_TEST_VETO") == "1" {
				return 0
			}
			return 1
		} // WM_QUERYENDSESSION
		if message == 0x16 && wp != 0 {
			os.Exit(0)
		}
		result, _, _ := user32.NewProc("DefWindowProcW").Call(hwnd, uintptr(message), wp, lp)
		return result
	})
	type windowClass struct {
		Size, Style                        uint32
		Callback                           uintptr
		ClassExtra, WindowExtra            int32
		Instance, Icon, Cursor, Background uintptr
		Menu, Class                        *uint16
		SmallIcon                          uintptr
	}
	name, _ := windows.UTF16PtrFromString("CuratedInstallerStopFixture")
	class := windowClass{Callback: callback, Class: name}
	class.Size = uint32(unsafe.Sizeof(class))
	if result, _, err := user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&class))); result == 0 {
		t.Fatal(err)
	}
	if result, _, err := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(name)), 0, 0, 0, 0, 0, 0, 0, 0, 0); result == 0 {
		t.Fatal(err)
	}
	if err := os.WriteFile(ready, []byte("ready"), 0600); err != nil {
		t.Fatal(err)
	}
	var message [64]byte
	for {
		result, _, _ := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&message[0])), 0, 0, 0)
		if result == 0 {
			return
		}
		user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&message[0])))
		user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&message[0])))
	}
}

func startStopFixture(t *testing.T) (string, *exec.Cmd) {
	t.Helper()
	dir := t.TempDir()
	current, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(current)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	binary := filepath.Join(dir, "curated.exe")
	target, err := os.Create(binary)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil || closeErr != nil {
		t.Fatalf("copy fixture: %v %v", copyErr, closeErr)
	}
	ready := filepath.Join(dir, "ready")
	cmd := exec.Command(binary, "-test.run=^TestStopWindowFixture$")
	cmd.Env = append(os.Environ(), "CURATED_STOP_TEST_READY="+ready)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("window fixture did not start")
		}
		time.Sleep(50 * time.Millisecond)
	}
	return dir, cmd
}

// TestStopInstallationClosesOnlySelectedDirectory checks probing and shutdown isolation with two live app instances.
func TestStopInstallationClosesOnlySelectedDirectory(t *testing.T) {
	dir, selected := startStopFixture(t)
	_, other := startStopFixture(t)
	if running, err := InstallationRunning(dir); err != nil || !running {
		t.Fatalf("probe failed: running=%v err=%v", running, err)
	}
	if running, err := InstallationRunning(t.TempDir()); err != nil || running {
		t.Fatalf("unrelated directory reported running: %v %v", running, err)
	}
	if err := StopInstallation(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	if err := selected.Wait(); err != nil {
		t.Fatalf("graceful exit: %v", err)
	}
	if running, err := InstallationRunning(dir); err != nil || running {
		t.Fatalf("stopped probe: %v %v", running, err)
	}
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(other.Process.Pid))
	if err != nil {
		t.Fatal("unrelated process was closed:", err)
	}
	defer windows.CloseHandle(handle)
	if status, err := windows.WaitForSingleObject(handle, 0); err != nil || status != uint32(windows.WAIT_TIMEOUT) {
		t.Fatalf("unrelated process changed: %v %v", status, err)
	}
	if err := StopInstallation(context.Background(), dir); err != nil {
		t.Fatal("retry:", err)
	}
}

// TestStopInstallationRejectsRelativeDirectory prevents ambiguous probe and shutdown targets.
func TestStopInstallationRejectsRelativeDirectory(t *testing.T) {
	if _, err := InstallationRunning("."); err == nil {
		t.Fatal("probe accepted relative directory")
	}
	if err := StopInstallation(context.Background(), "."); err == nil {
		t.Fatal("accepted relative directory")
	}
}

func TestStopInstallationDoesNotForceBusyApplication(t *testing.T) {
	t.Setenv("CURATED_STOP_TEST_VETO", "1")
	dir, cmd := startStopFixture(t)
	if err := StopInstallation(context.Background(), dir); err == nil {
		t.Fatal("busy application did not block upgrade")
	}
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(cmd.Process.Pid))
	if err != nil {
		t.Fatal("busy application was killed:", err)
	}
	defer windows.CloseHandle(handle)
	if status, err := windows.WaitForSingleObject(handle, 0); err != nil || status != uint32(windows.WAIT_TIMEOUT) {
		t.Fatalf("busy application changed: %v %v", status, err)
	}
}
