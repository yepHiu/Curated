//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

const (
	mbIconError = 0x00000010
	mbOk        = 0x00000000
)

func ShowErrorDialog(title string, message string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	messagePtr, _ := syscall.UTF16PtrFromString(message)
	_, _, _ = messageBox.Call(
		0,
		uintptr(unsafe.Pointer(messagePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(mbOk|mbIconError),
	)
}
