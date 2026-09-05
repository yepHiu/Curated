package playback

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
)

var processControlDLL = windows.NewLazySystemDLL("ntdll.dll")
var suspendProcessProc = processControlDLL.NewProc("NtSuspendProcess")
var resumeProcessProc = processControlDLL.NewProc("NtResumeProcess")

// Control only the FFmpeg child owned by this session, never a discovered PID.
func setProcessPaused(process *os.Process, paused bool) error {
	proc := resumeProcessProc
	if paused {
		proc = suspendProcessProc
	}
	if err := proc.Find(); err != nil {
		return err
	}
	handle, err := windows.OpenProcess(windows.PROCESS_SUSPEND_RESUME, false, uint32(process.Pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	status, _, _ := proc.Call(uintptr(handle))
	if status != 0 {
		return fmt.Errorf("process pause=%t: NTSTATUS %#x", paused, status)
	}
	return nil
}
