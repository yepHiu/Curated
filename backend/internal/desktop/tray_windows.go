//go:build windows

package desktop

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/sys/windows"

	"curated-backend/internal/assets"
	"curated-backend/internal/config"
	"curated-backend/internal/shellopen"
	"curated-backend/internal/version"
)

const (
	trayCommandOpenHome uint32 = 1001
	trayCommandSettings uint32 = 1002
	trayCommandLogs     uint32 = 1003
	trayCommandQuit     uint32 = 1099
)

const (
	wmApp              = 0x8000
	wmNull             = 0x0000
	wmClose            = 0x0010
	wmDestroy          = 0x0002
	wmEndSession       = 0x0016
	wmLButtonUp        = 0x0202
	wmRButtonUp        = 0x0205
	wmContextMenu      = 0x007B
	ninSelect          = wmApp
	ninKeySelect       = wmApp + 1
	notifyIconVersion4 = 4
	nifMessage         = 0x00000001
	nifIcon            = 0x00000002
	nifTip             = 0x00000004
	nimAdd             = 0x00000000
	nimModify          = 0x00000001
	nimDelete          = 0x00000002
	nimSetFocus        = 0x00000003
	nimSetVersion      = 0x00000004
	mfString           = 0x00000000
	mfSeparator        = 0x00000800
	tpmLeftAlign       = 0x0000
	tpmBottomAlign     = 0x0020
	tpmRightButton     = 0x0002
	tpmReturnCmd       = 0x0100
	imageIcon          = 1
	lrLoadFromFile     = 0x00000010
	lrDefaultSize      = 0x00000040
	swHide             = 0
)

var (
	shell32                    = windows.NewLazySystemDLL("shell32.dll")
	procShellNotifyIconW       = shell32.NewProc("Shell_NotifyIconW")
	user32                     = windows.NewLazySystemDLL("user32.dll")
	procAppendMenuW            = user32.NewProc("AppendMenuW")
	procCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	procCreateWindowExW        = user32.NewProc("CreateWindowExW")
	procDefWindowProcW         = user32.NewProc("DefWindowProcW")
	procDestroyIcon            = user32.NewProc("DestroyIcon")
	procDestroyMenu            = user32.NewProc("DestroyMenu")
	procDestroyWindow          = user32.NewProc("DestroyWindow")
	procDispatchMessageW       = user32.NewProc("DispatchMessageW")
	procGetCursorPos           = user32.NewProc("GetCursorPos")
	procGetMessageW            = user32.NewProc("GetMessageW")
	procLoadImageW             = user32.NewProc("LoadImageW")
	procPostMessageW           = user32.NewProc("PostMessageW")
	procPostQuitMessage        = user32.NewProc("PostQuitMessage")
	procRegisterClassExW       = user32.NewProc("RegisterClassExW")
	procRegisterWindowMessageW = user32.NewProc("RegisterWindowMessageW")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procShowWindow             = user32.NewProc("ShowWindow")
	procTrackPopupMenu         = user32.NewProc("TrackPopupMenu")
	procTranslateMessage       = user32.NewProc("TranslateMessage")
	procUnregisterClassW       = user32.NewProc("UnregisterClassW")
	kernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procGetModuleHandleW       = kernel32.NewProc("GetModuleHandleW")
	activeNativeTray           *nativeTrayRuntime
)

type TrayOptions struct {
	Logger      *zap.Logger
	Config      config.Config
	OpenURL     string
	SettingsURL string
	LogDir      string
	Cancel      context.CancelFunc
}

type nativeTrayRuntime struct {
	opts             TrayOptions
	className        *uint16
	windowClassName  string
	instance         windows.Handle
	window           windows.Handle
	menu             windows.Handle
	icon             windows.Handle
	iconPath         string
	trayMessageID    uint32
	taskbarCreatedID uint32
	notifyData       notifyIconData
	ready            chan error
	closeOnce        sync.Once
	cleanupOnce      sync.Once
}

type point struct {
	X int32
	Y int32
}

type msg struct {
	WindowHandle windows.Handle
	Message      uint32
	WParam       uintptr
	LParam       uintptr
	Time         uint32
	Pt           point
	LPrivate     uint32
}

type wndClassEx struct {
	Size, Style                        uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background windows.Handle
	MenuName, ClassName                *uint16
	IconSm                             windows.Handle
}

type notifyIconData struct {
	Size                       uint32
	Wnd                        windows.Handle
	ID, Flags, CallbackMessage uint32
	Icon                       windows.Handle
	Tip                        [128]uint16
	State, StateMask           uint32
	Info                       [256]uint16
	Timeout, Version           uint32
	InfoTitle                  [64]uint16
	InfoFlags                  uint32
	GuidItem                   windows.GUID
	BalloonIcon                windows.Handle
}

func RunTray(ctx context.Context, opts TrayOptions) error {
	if opts.Cancel == nil {
		return fmt.Errorf("nil tray cancel func")
	}

	runtimeCtx, runtimeCancel := context.WithCancel(ctx)
	tray := &nativeTrayRuntime{
		opts:            opts,
		windowClassName: fmt.Sprintf("CuratedTrayWindow.%d", os.Getpid()),
		trayMessageID:   wmApp + 1,
		ready:           make(chan error, 1),
	}

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer runtimeCancel()
		activeNativeTray = tray
		defer func() {
			if activeNativeTray == tray {
				activeNativeTray = nil
			}
		}()
		if err := tray.run(runtimeCtx); err != nil {
			select {
			case tray.ready <- err:
			default:
			}
		}
	}()

	select {
	case err := <-tray.ready:
		return err
	case <-time.After(10 * time.Second):
		runtimeCancel()
		return fmt.Errorf("tray startup timeout")
	case <-ctx.Done():
		runtimeCancel()
		return ctx.Err()
	}
}

func (t *nativeTrayRuntime) run(ctx context.Context) error {
	if err := t.initialize(); err != nil {
		return err
	}
	defer t.cleanup()
	t.signalReady(nil)

	go func() {
		<-ctx.Done()
		t.requestClose()
	}()
	if ctx.Err() != nil {
		return ctx.Err()
	}

	for {
		var message msg
		ret, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		switch int32(ret) {
		case -1:
			return err
		case 0:
			return nil
		default:
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
		}
	}
}

func (t *nativeTrayRuntime) signalReady(err error) {
	select {
	case t.ready <- err:
	default:
	}
}

func (t *nativeTrayRuntime) initialize() error {
	classNamePtr, err := windows.UTF16PtrFromString(t.windowClassName)
	if err != nil {
		return err
	}
	t.className = classNamePtr

	instance, _, err := procGetModuleHandleW.Call(0)
	if instance == 0 {
		return err
	}
	t.instance = windows.Handle(instance)

	taskbarCreatedName, err := windows.UTF16PtrFromString("TaskbarCreated")
	if err != nil {
		return err
	}
	res, _, err := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(taskbarCreatedName)))
	if res == 0 {
		return err
	}
	t.taskbarCreatedID = uint32(res)

	wc := &wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   windows.NewCallback(nativeTrayWndProc),
		Instance:  t.instance,
		ClassName: t.className,
	}
	res, _, err = procRegisterClassExW.Call(uintptr(unsafe.Pointer(wc)))
	if res == 0 {
		return err
	}

	windowHandle, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(t.className)),
		uintptr(unsafe.Pointer(t.className)),
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		uintptr(t.instance),
		0,
	)
	if windowHandle == 0 {
		_, _, _ = procUnregisterClassW.Call(uintptr(unsafe.Pointer(t.className)), uintptr(t.instance))
		return err
	}
	t.window = windows.Handle(windowHandle)
	procShowWindow.Call(uintptr(t.window), swHide)

	menuHandle, _, err := procCreatePopupMenu.Call()
	if menuHandle == 0 {
		return err
	}
	t.menu = windows.Handle(menuHandle)

	if err := t.appendMenuString(trayCommandOpenHome, "Open Curated"); err != nil {
		return err
	}
	if err := t.appendMenuString(trayCommandSettings, "Open Settings"); err != nil {
		return err
	}
	if err := t.appendMenuString(trayCommandLogs, "Open Logs"); err != nil {
		return err
	}
	if err := t.appendMenuSeparator(); err != nil {
		return err
	}
	if err := t.appendMenuString(trayCommandQuit, "Quit"); err != nil {
		return err
	}

	iconPath, err := writeTrayIconTempFile(assets.TrayIconICO)
	if err != nil {
		return err
	}
	t.iconPath = iconPath

	iconHandle, err := loadTrayIcon(iconPath)
	if err != nil {
		return err
	}
	t.icon = iconHandle

	t.notifyData = notifyIconData{
		Wnd:             t.window,
		ID:              1,
		Flags:           nifMessage | nifIcon | nifTip,
		CallbackMessage: t.trayMessageID,
		Icon:            t.icon,
	}
	t.notifyData.Size = uint32(unsafe.Sizeof(t.notifyData))
	copyTooltip(&t.notifyData.Tip, "Curated "+version.Display())

	if err := t.addNotifyIcon(); err != nil {
		return err
	}
	if err := t.setNotifyVersion(notifyIconVersion4); err != nil && t.opts.Logger != nil {
		t.opts.Logger.Warn("tray: failed to set notification icon version", zap.Error(err))
	}

	return nil
}

func (t *nativeTrayRuntime) cleanup() {
	t.cleanupOnce.Do(func() {
		if t.window != 0 {
			_, _, _ = procDestroyWindow.Call(uintptr(t.window))
			t.window = 0
		}
		if t.menu != 0 {
			_, _, _ = procDestroyMenu.Call(uintptr(t.menu))
			t.menu = 0
		}
		if t.icon != 0 {
			_, _, _ = procDestroyIcon.Call(uintptr(t.icon))
			t.icon = 0
		}
		if t.className != nil && t.instance != 0 {
			_, _, _ = procUnregisterClassW.Call(uintptr(unsafe.Pointer(t.className)), uintptr(t.instance))
		}
	})
}

func (t *nativeTrayRuntime) appendMenuString(commandID uint32, label string) error {
	labelPtr, err := windows.UTF16PtrFromString(label)
	if err != nil {
		return err
	}
	res, _, callErr := procAppendMenuW.Call(uintptr(t.menu), mfString, uintptr(commandID), uintptr(unsafe.Pointer(labelPtr)))
	if res == 0 {
		return callErr
	}
	return nil
}

func (t *nativeTrayRuntime) appendMenuSeparator() error {
	res, _, callErr := procAppendMenuW.Call(uintptr(t.menu), mfSeparator, 0, 0)
	if res == 0 {
		return callErr
	}
	return nil
}

func (t *nativeTrayRuntime) handleWindowMessage(hWnd windows.Handle, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case t.trayMessageID:
		switch uint32(lParam & 0xffff) {
		case wmRButtonUp, wmContextMenu, ninSelect, ninKeySelect:
			t.showContextMenu()
		case wmLButtonUp:
			t.handleCommand(trayCommandOpenHome)
		}
		return 0
	case t.taskbarCreatedID:
		if err := t.addNotifyIcon(); err != nil && t.opts.Logger != nil {
			t.opts.Logger.Warn("tray: failed to restore icon after explorer restart", zap.Error(err))
		}
		if err := t.setNotifyVersion(notifyIconVersion4); err != nil && t.opts.Logger != nil {
			t.opts.Logger.Warn("tray: failed to restore notification icon version", zap.Error(err))
		}
		return 0
	case wmClose:
		t.removeNotifyIcon()
		_, _, _ = procDestroyWindow.Call(uintptr(t.window))
		return 0
	case wmDestroy, wmEndSession:
		t.removeNotifyIcon()
		procPostQuitMessage.Call(0)
		return 0
	default:
		result, _, _ := procDefWindowProcW.Call(uintptr(hWnd), uintptr(message), wParam, lParam)
		return result
	}
}

func (t *nativeTrayRuntime) showContextMenu() {
	if t.menu == 0 || t.window == 0 {
		return
	}

	var pt point
	res, _, err := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if res == 0 {
		if t.opts.Logger != nil {
			t.opts.Logger.Warn("tray: failed to get cursor position", zap.Error(err))
		}
		return
	}

	procSetForegroundWindow.Call(uintptr(t.window))
	cmd, _, err := procTrackPopupMenu.Call(
		uintptr(t.menu),
		tpmLeftAlign|tpmBottomAlign|tpmRightButton|tpmReturnCmd,
		uintptr(pt.X),
		uintptr(pt.Y),
		0,
		uintptr(t.window),
		0,
	)
	procPostMessageW.Call(uintptr(t.window), wmNull, 0, 0)
	_ = t.setNotifyFocus()
	if cmd == 0 {
		if err != windows.ERROR_SUCCESS && t.opts.Logger != nil {
			t.opts.Logger.Warn("tray: popup menu returned no command", zap.Error(err))
		}
		return
	}
	t.handleCommand(uint32(cmd))
}

func (t *nativeTrayRuntime) handleCommand(commandID uint32) {
	switch commandID {
	case trayCommandOpenHome:
		go t.openURL(t.opts.OpenURL, "tray: open home failed")
	case trayCommandSettings:
		go t.openURL(t.opts.SettingsURL, "tray: open settings failed")
	case trayCommandLogs:
		go t.openDirectory(t.opts.LogDir, "tray: open logs failed")
	case trayCommandQuit:
		go t.opts.Cancel()
		t.requestClose()
	}
}

func (t *nativeTrayRuntime) openURL(target string, message string) {
	if err := shellopen.OpenURL(context.Background(), target); err != nil && t.opts.Logger != nil {
		t.opts.Logger.Warn(message, zap.Error(err))
	}
}

func (t *nativeTrayRuntime) openDirectory(target string, message string) {
	if err := shellopen.OpenDirectory(context.Background(), target); err != nil && t.opts.Logger != nil {
		t.opts.Logger.Warn(message, zap.Error(err))
	}
}

func (t *nativeTrayRuntime) requestClose() {
	t.closeOnce.Do(func() {
		if t.window != 0 {
			procPostMessageW.Call(uintptr(t.window), wmClose, 0, 0)
		}
	})
}

func (t *nativeTrayRuntime) addNotifyIcon() error {
	t.notifyData.Size = uint32(unsafe.Sizeof(t.notifyData))
	res, _, err := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&t.notifyData)))
	if res == 0 {
		return err
	}
	return nil
}

func (t *nativeTrayRuntime) removeNotifyIcon() {
	t.notifyData.Size = uint32(unsafe.Sizeof(t.notifyData))
	_, _, _ = procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&t.notifyData)))
}

func (t *nativeTrayRuntime) setNotifyVersion(versionValue uint32) error {
	t.notifyData.Version = versionValue
	t.notifyData.Size = uint32(unsafe.Sizeof(t.notifyData))
	res, _, err := procShellNotifyIconW.Call(nimSetVersion, uintptr(unsafe.Pointer(&t.notifyData)))
	if res == 0 {
		return err
	}
	return nil
}

func (t *nativeTrayRuntime) setNotifyFocus() error {
	t.notifyData.Size = uint32(unsafe.Sizeof(t.notifyData))
	res, _, err := procShellNotifyIconW.Call(nimSetFocus, uintptr(unsafe.Pointer(&t.notifyData)))
	if res == 0 {
		return err
	}
	return nil
}

func nativeTrayWndProc(hWnd windows.Handle, message uint32, wParam, lParam uintptr) uintptr {
	if activeNativeTray == nil {
		result, _, _ := procDefWindowProcW.Call(uintptr(hWnd), uintptr(message), wParam, lParam)
		return result
	}
	return activeNativeTray.handleWindowMessage(hWnd, message, wParam, lParam)
}

func loadTrayIcon(iconPath string) (windows.Handle, error) {
	iconPathPtr, err := windows.UTF16PtrFromString(iconPath)
	if err != nil {
		return 0, err
	}
	res, _, callErr := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(iconPathPtr)),
		imageIcon,
		0,
		0,
		lrLoadFromFile|lrDefaultSize,
	)
	if res == 0 {
		return 0, callErr
	}
	return windows.Handle(res), nil
}

func writeTrayIconTempFile(iconBytes []byte) (string, error) {
	if len(iconBytes) == 0 {
		return "", fmt.Errorf("empty tray icon bytes")
	}
	sum := md5.Sum(iconBytes)
	name := "curated_tray_" + hex.EncodeToString(sum[:]) + ".ico"
	path := filepath.Join(os.TempDir(), name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.WriteFile(path, iconBytes, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func copyTooltip(dst *[128]uint16, text string) {
	utf16Text, _ := windows.UTF16FromString(text)
	copy(dst[:], utf16Text)
}

func WaitForServerReady(ctx context.Context, baseURL string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	healthURL := strings.TrimRight(baseURL, "/") + "/api/health"
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err == nil {
			resp, reqErr := client.Do(req)
			if reqErr == nil {
				_ = resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 500 {
					return nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func ResolveBaseURL(addr string) string {
	host := strings.TrimSpace(addr)
	if host == "" {
		host = ":8080"
	}
	if strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}
	host = strings.ReplaceAll(host, "0.0.0.0", "127.0.0.1")
	host = strings.ReplaceAll(host, "[::]", "127.0.0.1")
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	return strings.TrimRight(host, "/")
}

func ResolveDefaultLogDir(cfg config.Config) string {
	if strings.TrimSpace(cfg.LogDir) != "" {
		if abs, err := filepath.Abs(cfg.LogDir); err == nil {
			return abs
		}
		return cfg.LogDir
	}
	settingsPath := config.DefaultLibrarySettingsPath()
	root := filepath.Dir(filepath.Dir(settingsPath))
	return filepath.Join(root, "logs")
}
