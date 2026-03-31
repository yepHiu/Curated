package webui

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
			http.ServeFile(w, r, filePath)
			return
		}

		http.ServeFile(w, r, indexPath)
	})
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
