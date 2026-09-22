package app

import (
	"curated-backend/internal/config"
	"fmt"
)

// BrowserPluginEnabled reports whether browser plugin requests are accepted.
func (a *App) BrowserPluginEnabled() bool {
	a.browserPluginMu.RLock()
	defer a.browserPluginMu.RUnlock()
	return a.browserPluginEnabled
}

// SetBrowserPluginEnabled persists the preference and applies it without restarting.
func (a *App) SetBrowserPluginEnabled(v bool) error {
	a.browserPluginMu.Lock()
	defer a.browserPluginMu.Unlock()
	if a.librarySettingsPath == "" {
		return fmt.Errorf("library settings path not configured")
	}
	if err := config.WriteLibrarySettingsMerge(a.librarySettingsPath, func(m map[string]any) error {
		m["browserPluginEnabled"] = v
		return nil
	}); err != nil {
		return err
	}
	a.browserPluginEnabled = v
	return nil
}
