//go:build windows

// curated-migrate is embedded in installers, never installed as a user-facing command.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"curated-backend/internal/installmigration"
	"curated-backend/internal/launchprofile"
)

// main dispatches private installer helper actions and writes actionable failure details.
func main() {
	action := flag.String("action", "prepare", "prepare, complete, validate-directory, probe or stop")
	data := flag.String("data-root", "", "original absolute data directory")
	config := flag.String("config", "", "original absolute runtime config")
	errorFile := flag.String("error-file", "", "UTF-8 installer error output")
	programDir := flag.String("program-dir", "", "exact installation directory to inspect or stop")
	installedDir := flag.String("installed-dir", "", "registered directory for an existing installation")
	flag.Parse()
	local := os.Getenv("LOCALAPPDATA")
	if !filepath.IsAbs(local) {
		fmt.Fprintln(os.Stderr, "LOCALAPPDATA is unavailable")
		os.Exit(1)
	}
	// Empty on retries means reuse the journal's selection. First-time default
	// selection is supplied by Full's upgrade page, including CURATED_DATA_DIR.
	o := installmigration.Options{StateDir: filepath.Join(local, "Curated", "installer-migrations"), ProfilePath: launchprofile.Path(local), DataRoot: *data, ConfigPath: *config}
	var err error
	switch *action {
	case "validate-directory":
		err = installmigration.ValidateProgramDirectory(*programDir, *installedDir)
	case "probe":
		var running bool
		running, err = installmigration.InstallationRunning(*programDir)
		if err == nil && running {
			os.Exit(2) // 0 = stopped, 2 = running, 1 = cannot determine safely
		}
	case "stop":
		err = installmigration.StopInstallation(context.Background(), *programDir)
	case "prepare":
		err = installmigration.Prepare(context.Background(), o, installmigration.Windows{})
	case "complete":
		err = installmigration.Complete(context.Background(), o, installmigration.Windows{})
	default:
		err = fmt.Errorf("unknown migration action")
	}
	if err != nil {
		message := err.Error()
		if *action == "prepare" || *action == "complete" {
			message += "\r\nMigration records and backups: " + o.StateDir
		}
		if *errorFile != "" {
			_ = os.WriteFile(*errorFile, []byte(message), 0600)
		}
		fmt.Fprintln(os.Stderr, message)
		os.Exit(1)
	}
}
