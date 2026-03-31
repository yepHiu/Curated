package assets

import _ "embed"

// TrayIconICO contains the Windows tray icon used by release builds.
//
//go:embed curated.ico
var TrayIconICO []byte
