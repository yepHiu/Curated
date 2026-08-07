// Package webui serves the production frontend dist and delegates /api requests to the backend handler.
package webui

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	entryCacheControl  = "no-store, no-cache, must-revalidate"
	assetCacheControl  = "public, max-age=31536000, immutable"
	staticCacheControl = "no-cache"
)

// WrapHandler serves the frontend dist when it can be located locally.
// All /api requests are delegated to apiHandler unchanged.
// Non-API GET/HEAD requests use SPA fallback to index.html.
func WrapHandler(apiHandler http.Handler) http.Handler {
	if apiHandler == nil {
		return nil
	}
	distDir := FindDistDir()
	if distDir == "" {
		return apiHandler
	}
	indexPath := filepath.Join(distDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return apiHandler
	}

	return wrapHandlerWithDist(apiHandler, distDir)
}

func wrapHandlerWithDist(apiHandler http.Handler, distDir string) http.Handler {
	indexPath := filepath.Join(distDir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" {
			apiHandler.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			apiHandler.ServeHTTP(w, r)
			return
		}

		if filePath, ok := resolveRequestFile(distDir, r.URL.Path); ok {
			setFrontendCacheHeaders(w, r.URL.Path, filepath.Clean(filePath) == filepath.Clean(indexPath))
			http.ServeFile(w, r, filePath)
			return
		}

		setFrontendCacheHeaders(w, r.URL.Path, true)
		http.ServeFile(w, r, indexPath)
	})
}

func setFrontendCacheHeaders(w http.ResponseWriter, requestPath string, isEntry bool) {
	if isEntry {
		w.Header().Set("Cache-Control", entryCacheControl)
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		return
	}
	if strings.HasPrefix(requestPath, "/assets/") {
		w.Header().Set("Cache-Control", assetCacheControl)
		return
	}
	w.Header().Set("Cache-Control", staticCacheControl)
}

// FindDistDir resolves the best local frontend dist directory.
func FindDistDir() string {
	candidates := make([]string, 0, 4)
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "frontend-dist"),
			filepath.Join(exeDir, "dist"),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "frontend-dist"),
			filepath.Join(cwd, "dist"),
		)
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		indexPath := filepath.Join(candidate, "index.html")
		if info, err := os.Stat(indexPath); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func resolveRequestFile(distDir string, requestPath string) (string, bool) {
	cleanPath := filepath.Clean("/" + strings.TrimSpace(requestPath))
	relativePath := strings.TrimPrefix(cleanPath, "/")
	if relativePath == "" || relativePath == "." {
		relativePath = "index.html"
	}

	fullPath := filepath.Join(distDir, filepath.FromSlash(relativePath))
	rel, err := filepath.Rel(distDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		return "", false
	}
	return fullPath, true
}
