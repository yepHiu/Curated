//go:build windows

// curated-migrate is embedded in Full, never installed as a user-facing command.
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

func main() {
	action := flag.String("action", "prepare", "prepare or complete")
	data := flag.String("data-root", "", "original absolute data directory")
	config := flag.String("config", "", "original absolute runtime config")
	errorFile := flag.String("error-file", "", "UTF-8 error output for Full")
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
	case "prepare":
		err = installmigration.Prepare(context.Background(), o, installmigration.Windows{})
	case "complete":
		err = installmigration.Complete(context.Background(), o, installmigration.Windows{})
	default:
		err = fmt.Errorf("unknown migration action")
	}
	if err != nil {
		message := err.Error() + "\r\nMigration records and backups: " + o.StateDir
		if *errorFile != "" {
			_ = os.WriteFile(*errorFile, []byte(message), 0600)
		}
		fmt.Fprintln(os.Stderr, message)
		os.Exit(1)
	}
}
