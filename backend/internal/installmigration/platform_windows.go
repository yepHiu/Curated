//go:build windows

package installmigration

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const LegacyKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1`
const componentKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\Curated.%s_is1`
const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

type Windows struct{}

func (Windows) FindLegacy() (*Legacy, error) {
	var found *Legacy
	for _, hive := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, view := range []uint32{registry.WOW64_64KEY, registry.WOW64_32KEY} {
			key, err := registry.OpenKey(hive, LegacyKey, registry.QUERY_VALUE|view)
			if errors.Is(err, registry.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, err
			}
			dir, _, e1 := key.GetStringValue("InstallLocation")
			version, _, e2 := key.GetStringValue("DisplayVersion")
			command, _, e3 := key.GetStringValue("UninstallString")
			key.Close()
			if err = errors.Join(e1, e2, e3); err != nil {
				return nil, fmt.Errorf("incomplete old Curated registration: %w", err)
			}
			dir = filepath.Clean(strings.TrimRight(dir, `\/`))
			args, err := windows.DecomposeCommandLine(command)
			if err != nil || len(args) != 1 || !filepath.IsAbs(dir) || !strings.EqualFold(args[0], filepath.Join(dir, "unins000.exe")) {
				return nil, fmt.Errorf("unrecognized legacy uninstall command; no program was removed")
			}
			parts := strings.Split(version, ".")
			if len(parts) < 3 || len(parts) > 4 {
				return nil, fmt.Errorf("unrecognized legacy version %q", version)
			}
			for _, part := range parts {
				if n, err := strconv.Atoi(part); err != nil || n < 0 {
					return nil, fmt.Errorf("unrecognized legacy version")
				}
			}
			major, _ := strconv.Atoi(parts[0])
			minor, _ := strconv.Atoi(parts[1])
			if major != 1 || minor > 5 {
				return nil, fmt.Errorf("this all-in-one version is outside the supported migration range: %s", version)
			}
			for _, file := range []string{args[0], filepath.Join(dir, "curated.ico")} {
				info, err := os.Stat(file)
				if err != nil {
					return nil, err
				}
				if !info.Mode().IsRegular() {
					return nil, fmt.Errorf("invalid legacy installation file")
				}
			}
			old := &Legacy{Directory: dir, Version: version, Uninstaller: args[0], Machine: hive == registry.LOCAL_MACHINE}
			if found != nil && *found != *old {
				return nil, fmt.Errorf("multiple old Curated installations found; select one installation before migrating")
			}
			found = old
		}
	}
	return found, nil
}

func componentDirectory(name string) (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, fmt.Sprintf(componentKey, name), registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return "", err
	}
	defer key.Close()
	dir, _, err := key.GetStringValue("InstallLocation")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("invalid %s installation location", name)
	}
	return dir, nil
}

func (Windows) CheckNoSplitServer() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, fmt.Sprintf(componentKey, "Server"), registry.QUERY_VALUE|registry.WOW64_64KEY)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	key.Close()
	return fmt.Errorf("both legacy Curated and a split Server are installed; resolve the two data sources before migrating")
}

func (platform Windows) CheckStopped(old *Legacy) error {
	startup, err := platform.StartupCommand()
	if err != nil {
		return err
	}
	if startup != "" {
		args, err := windows.DecomposeCommandLine(startup)
		if err != nil || len(args) == 0 || !within(old.Directory, args[0]) {
			return fmt.Errorf("Curated startup entry belongs to another location; no program was removed")
		}
	}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	err = windows.Process32First(snapshot, &entry)
	for err == nil {
		name := strings.ToLower(windows.UTF16ToString(entry.ExeFile[:]))
		if name == "curated.exe" || name == "curated desktop.exe" {
			process, openErr := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
			if openErr != nil {
				return fmt.Errorf("cannot verify Curated process %d; fully quit Curated in all sessions", entry.ProcessID)
			}
			buffer := make([]uint16, 32768)
			size := uint32(len(buffer))
			pathErr := windows.QueryFullProcessImageName(process, 0, &buffer[0], &size)
			windows.CloseHandle(process)
			if pathErr != nil {
				return pathErr
			}
			path := windows.UTF16ToString(buffer[:size])
			// Other users/instances are never terminated. Refuse rather than guessing
			// which custom database a same-name backend might have opened.
			return fmt.Errorf("fully quit Curated from its tray and retry Full (running: %s)", path)
		}
		err = windows.Process32Next(snapshot, &entry)
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return err
	}
	return nil
}

func (Windows) StartupCommand() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer key.Close()
	command, _, err := key.GetStringValue("Curated")
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}
	return command, err
}

func (Windows) DisableOldStartup(old *Legacy, expected string) error {
	if expected == "" {
		return nil
	}
	args, err := windows.DecomposeCommandLine(expected)
	if err != nil || len(args) == 0 || !within(old.Directory, args[0]) {
		return fmt.Errorf("Curated startup entry belongs to another location; it was not changed")
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	current, _, err := key.GetStringValue("Curated")
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if current != expected {
		// Complete may have switched startup before a later step failed or
		// power was lost. Retrying must keep that already-installed entry.
		if server, lookupErr := componentDirectory("Server"); lookupErr == nil {
			installedCommand := `"` + filepath.Join(server, "curated.exe") + `" -mode tray -autostart`
			if current == installedCommand {
				return nil
			}
		}
		return fmt.Errorf("Curated startup entry changed during migration")
	}
	return key.DeleteValue("Curated")
}

func (Windows) Uninstall(ctx context.Context, old *Legacy) error {
	if !old.Machine {
		return exec.CommandContext(ctx, old.Uninstaller, "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART").Run()
	}
	// Only the registered old uninstaller elevates; Full and all new installers
	// keep the signed-in user's identity, even with over-the-shoulder UAC.
	// Path is passed as data through the environment, never PowerShell source.
	source := `$ErrorActionPreference='Stop'; try { $p = Start-Process -FilePath $env:CURATED_MIGRATION_UNINSTALLER -ArgumentList '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART' -Verb RunAs -Wait -PassThru; exit $p.ExitCode } catch { exit 1 }`
	encoded := utf16.Encode([]rune(source))
	raw := make([]byte, len(encoded)*2)
	for i, ch := range encoded {
		binary.LittleEndian.PutUint16(raw[i*2:], ch)
	}
	powershell := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	command := exec.CommandContext(ctx, powershell, "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(raw))
	command.Env = append(os.Environ(), "CURATED_MIGRATION_UNINSTALLER="+old.Uninstaller)
	return command.Run()
}

func (Windows) Complete(originalStartup string, enabled bool) error {
	server, err := componentDirectory("Server")
	if err != nil {
		return err
	}
	desktop, err := componentDirectory("Desktop")
	if err != nil {
		return err
	}
	for _, file := range []string{filepath.Join(server, "curated.exe"), filepath.Join(desktop, "Curated Desktop.exe")} {
		if info, err := os.Stat(file); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("installed component is missing: %s", file)
		}
	}
	if enabled || originalStartup != "" {
		key, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE|registry.SET_VALUE)
		if err != nil {
			return err
		}
		defer key.Close()
		command := `"` + filepath.Join(server, "curated.exe") + `" -mode tray -autostart`
		current, _, err := key.GetStringValue("Curated")
		if err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		if err == nil && current != originalStartup && current != command {
			return fmt.Errorf("Curated startup was changed by another program; not overwriting it")
		}
		if err = key.SetStringValue("Curated", command); err != nil {
			return err
		}
	}
	// The old package used the curated-desktop profile name. Preserve saved
	// connections only when the new profile has none. Authentication may require
	// unlocking again; do not merge Chromium cookie databases from two profiles.
	source := filepath.Join(os.Getenv("APPDATA"), "curated-desktop", "servers.json")
	target := filepath.Join(os.Getenv("APPDATA"), "Curated Desktop", "servers.json")
	raw, err := os.ReadFile(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !json.Valid(raw) {
		return fmt.Errorf("old Desktop connections are corrupt; original profile retained")
	}
	if _, err = os.Stat(target); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	// Publish the connection file only after a complete write. A crash must not
	// leave a truncated target that a retry would mistake for an existing profile.
	temp, err := os.CreateTemp(filepath.Dir(target), ".legacy-connections-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err = temp.Write(raw); err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	// Windows Rename replaces an existing target; hard-link publication gives
	// us an atomic no-overwrite operation on the profile's NTFS volume.
	if err = os.Link(temp.Name(), target); os.IsExist(err) {
		return nil
	}
	return err
}
