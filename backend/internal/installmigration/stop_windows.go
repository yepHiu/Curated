//go:build windows

package installmigration

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type restartProcess struct {
	PID     uint32
	Created windows.Filetime
}

// StopInstallation uses Restart Manager only for exact executables belonging to
// the selected installation. Other accounts and sessions are never shut down.
// It does not force termination: a busy/unresponsive application blocks upgrade.
func StopInstallation(ctx context.Context, directory string) error {
	if !filepath.IsAbs(directory) {
		return fmt.Errorf("installation directory must be absolute")
	}
	directory = filepath.Clean(directory)
	targets := []string{filepath.Join(directory, "Curated.exe"), filepath.Join(directory, "Curated Desktop.exe"), filepath.Join(directory, "resources", "app", "curated.exe")}
	currentUser, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	var session uint32
	if err = windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &session); err != nil {
		return err
	}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)
	var processes []restartProcess
	var handles []windows.Handle
	defer func() {
		for _, h := range handles {
			windows.CloseHandle(h)
		}
	}()
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if !strings.EqualFold(name, "curated.exe") && !strings.EqualFold(name, "Curated Desktop.exe") {
			continue
		}
		h, openErr := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, entry.ProcessID)
		if errors.Is(openErr, windows.ERROR_INVALID_PARAMETER) {
			continue
		} // exited
		if openErr != nil {
			return fmt.Errorf("cannot inspect Curated process %d: %w", entry.ProcessID, openErr)
		}
		handles = append(handles, h)
		buffer := make([]uint16, 32768)
		size := uint32(len(buffer))
		if err := windows.QueryFullProcessImageName(h, 0, &buffer[0], &size); err != nil {
			return err
		}
		path := windows.UTF16ToString(buffer[:size])
		matches := false
		for _, target := range targets {
			matches = matches || strings.EqualFold(path, target)
		}
		if !matches {
			handles = handles[:len(handles)-1]
			windows.CloseHandle(h)
			continue
		}
		var token windows.Token
		if err := windows.OpenProcessToken(h, windows.TOKEN_QUERY, &token); err != nil {
			return err
		}
		user, tokenErr := token.GetTokenUser()
		token.Close()
		if tokenErr != nil {
			return tokenErr
		}
		var processSession uint32
		if err := windows.ProcessIdToSessionId(entry.ProcessID, &processSession); err != nil {
			return err
		}
		if processSession != session || !user.User.Sid.Equals(currentUser.User.Sid) {
			return fmt.Errorf("Curated is running under another user or session; close it there before upgrading")
		}
		var created, exited, kernel, userTime windows.Filetime
		if err := windows.GetProcessTimes(h, &created, &exited, &kernel, &userTime); err != nil {
			return err
		}
		processes = append(processes, restartProcess{entry.ProcessID, created})
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return err
	}
	if len(processes) == 0 {
		return nil
	}
	rm := windows.NewLazySystemDLL("rstrtmgr.dll")
	var rmSession uint32
	var key [33]uint16
	code, _, _ := rm.NewProc("RmStartSession").Call(uintptr(unsafe.Pointer(&rmSession)), 0, uintptr(unsafe.Pointer(&key[0])))
	if code != 0 {
		return fmt.Errorf("Restart Manager start: %d", code)
	}
	defer rm.NewProc("RmEndSession").Call(uintptr(rmSession))
	code, _, _ = rm.NewProc("RmRegisterResources").Call(uintptr(rmSession), 0, 0, uintptr(len(processes)), uintptr(unsafe.Pointer(&processes[0])), 0, 0)
	if code != 0 {
		return fmt.Errorf("Restart Manager register: %d", code)
	}
	code, _, _ = rm.NewProc("RmShutdown").Call(uintptr(rmSession), 0, 0)
	if code != 0 {
		return fmt.Errorf("could not close Curated automatically (Windows error %d); exit from its tray and retry", code)
	}
	wait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for _, h := range handles {
		for {
			status, err := windows.WaitForSingleObject(h, 100)
			if err != nil {
				return err
			}
			if status == windows.WAIT_OBJECT_0 {
				break
			}
			if wait.Err() != nil {
				return fmt.Errorf("Curated did not exit; no program was removed: %w", wait.Err())
			}
		}
	}
	return nil
}
