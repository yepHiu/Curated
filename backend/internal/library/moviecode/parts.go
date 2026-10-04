package moviecode

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var explicitPartPattern = regexp.MustCompile(`(?i)(?:^|[-_ .])(?:CD|PART|PT|DISC|DISK)[-_ .]*([0-9]{1,3})(?:[^0-9]|$)`)
var numericPartPattern = regexp.MustCompile(`^[-_ .]+([0-9]{1,3})(?:[-_ .].*)?$`)
var fc2CodePattern = regexp.MustCompile(`(?i)^FC2[-_ ]*(?:PPV[-_ ]*)?([0-9]+)$`)

// ExtractPartIndex 识别明确分片标记或番号后的数字，不把番号数字本身当成分片。
func ExtractPartIndex(path, code string) int {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if match := explicitPartPattern.FindStringSubmatch(base); len(match) > 1 {
		index, _ := strconv.Atoi(match[1])
		return index
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return 0
	}
	fields := strings.FieldsFunc(code, func(r rune) bool {
		// 将番号连接符归一成可匹配的连接符集合。
		return r == '-' || r == '_' || r == ' '
	})
	for i := range fields {
		fields[i] = regexp.QuoteMeta(fields[i])
	}
	pattern := strings.Join(fields, `[-_ ]*`)
	if match := fc2CodePattern.FindStringSubmatch(code); len(match) > 1 {
		pattern = `FC2[-_ ]*(?:PPV[-_ ]*)?` + match[1]
	}
	end := regexp.MustCompile(`(?i)` + pattern).FindStringIndex(base)
	if end == nil {
		return 0
	}
	if match := numericPartPattern.FindStringSubmatch(base[end[1]:]); len(match) > 1 {
		index, _ := strconv.Atoi(match[1])
		return index
	}
	return 0
}
