package core

import (
	"encoding/json"
	"strconv"
	"strings"
)

var filesystemKeys = map[string]struct{}{
	"location":           {},
	"path":               {},
	"localPath":          {},
	"fileName":           {},
	"sourceFileName":     {},
	"avatarLocalUrl":     {},
	"downloadedFilePath": {},
	"rootPath":           {},
}

func Project(data any, level string) any {
	if data == nil {
		return nil
	}
	if level == "" || level == SanitizeFull {
		return data
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return data
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return data
	}
	return stripValue(value, level)
}

func stripValue(value any, level string) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			if _, sensitive := filesystemKeys[key]; sensitive {
				continue
			}
			if level == SanitizeMinimal && isTitleLike(key) {
				continue
			}
			out[key] = stripValue(child, level)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = stripValue(child, level)
		}
		return out
	default:
		return value
	}
}

func isTitleLike(key string) bool {
	switch key {
	case "title", "summary", "userSummary", "comment", "body":
		return true
	default:
		return false
	}
}

func ArgsSummary(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "{}"
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		if len(trimmed) > 2048 {
			return trimmed[:2048] + "…"
		}
		return trimmed
	}
	encoded, err := json.Marshal(Project(parsed, SanitizeSanitized))
	if err != nil {
		encoded = []byte(trimmed)
	}
	const max = 2048
	if len(encoded) > max {
		return string(encoded[:max]) + "…"
	}
	return string(encoded)
}

func PageCursor(offset, limit, total int) (next string, truncated bool) {
	if limit <= 0 {
		limit = DefaultListLimit
	}
	if offset < 0 {
		offset = 0
	}
	if offset+limit < total {
		return strconv.Itoa(offset + limit), true
	}
	return "", false
}
