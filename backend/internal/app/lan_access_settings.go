package app

import (
	"fmt"

	"curated-backend/internal/config"
)

// LANEnabled reports the persisted preference to expose the HTTP server on the LAN.
func (a *App) LANEnabled() bool {
	a.lanEnabledMu.RLock()
	defer a.lanEnabledMu.RUnlock()
	return a.lanEnabled
}

// LANListening reports whether this process is currently bound to a non-loopback address.
func (a *App) LANListening() bool {
	a.lanEnabledMu.RLock()
	defer a.lanEnabledMu.RUnlock()
	return !config.HTTPAddrIsLoopback(a.cfg.HttpAddr)
}

// LANAccessURLs lists private IPv4 URLs that LAN clients can try after the listener is rebound.
func (a *App) LANAccessURLs() []string {
	a.lanEnabledMu.RLock()
	defer a.lanEnabledMu.RUnlock()
	urls := config.LANAccessURLs(a.cfg.HttpAddr)
	if urls == nil {
		return []string{}
	}
	return urls
}

// SetLANEnabled persists lanEnabled to library-config.cfg. PIN lock is not required.
// The current HTTP bind address is not changed until Curated fully restarts.
func (a *App) SetLANEnabled(v bool) error {
	path := a.librarySettingsPath
	if path == "" {
		return fmt.Errorf("library settings path not configured")
	}
	if err := config.WriteLibrarySettingsMerge(path, func(m map[string]any) error {
		m["lanEnabled"] = v
		return nil
	}); err != nil {
		return err
	}
	a.lanEnabledMu.Lock()
	a.lanEnabled = v
	a.cfg.LANEnabled = v
	a.lanEnabledMu.Unlock()
	return nil
}
