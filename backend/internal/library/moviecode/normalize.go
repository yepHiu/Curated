package moviecode

import "strings"

// NormalizeForStorageID matches movies.id generation from scan persistence (lower, space/underscore → hyphen).
func NormalizeForStorageID(number string) string {
	id := strings.TrimSpace(number)
	id = strings.ToLower(id)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "_", "-")
	return id
}
