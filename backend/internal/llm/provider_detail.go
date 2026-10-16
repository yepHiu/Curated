package llm

import (
	"encoding/json"
	"regexp"
	"strings"
)

var providerBearer = regexp.MustCompile(`(?i)bearer\s+[^\s"'<>]+`)
var providerURL = regexp.MustCompile(`(?i)https?://[^\s"'<>]+`)
var providerCredential = regexp.MustCompile(`(?i)(["']?(?:api[_-]?key|access[_-]?token|authorization|password|secret)["']?\s*[:=]\s*)(?:"[^"\r\n]*"|'[^'\r\n]*'|[^\s,;}]+)`)

// Keep the provider's diagnostic text, not request headers or echoed credentials.
func publicProviderDetail(detail, key string) string {
	if key = strings.TrimSpace(key); key != "" {
		detail = strings.ReplaceAll(detail, key, "[redacted]")
		encoded, _ := json.Marshal(key)
		detail = strings.ReplaceAll(detail, string(encoded[1:len(encoded)-1]), "[redacted]")
	}
	detail = providerCredential.ReplaceAllString(detail, "${1}[redacted]")
	detail = providerBearer.ReplaceAllString(detail, "Bearer [redacted]")
	detail = providerURL.ReplaceAllString(detail, "[redacted URL]")
	runes := []rune(strings.ToValidUTF8(detail, "�"))
	if len(runes) > 4096 {
		return string(runes[:4096]) + "… [truncated]"
	}
	return string(runes)
}
