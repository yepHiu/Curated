package serveridentity

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
)

// WriteConnectionHint writes only a local endpoint, never credentials. A shared
// per-user location lets Full installs suggest custom ports without giving
// Desktop any ownership of Server processes or private library paths.
func WriteConnectionHint(listenAddr string) error {
	root := os.Getenv("LOCALAPPDATA")
	if root == "" {
		return nil
	}
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return err
	}
	dir := filepath.Join(root, "Curated")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	data, err := json.Marshal(map[string]string{"url": "http://" + net.JoinHostPort(host, port)})
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "server-connection.json.tmp")
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "server-connection.json"))
}
