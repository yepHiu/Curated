package moviecode

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var explicitPartPattern = regexp.MustCompile(`(?i)(?:^|[-_ .])(?:CD|PART|PT|DISC|DISK)[-_ .]*([0-9]{1,3})(?:[^0-9]|$)`)
var numericPartPattern = regexp.MustCompile(`^[-_ .]+([0-9]{1,3})(?:[-_ .].*)?$`)
var letterPartPattern = regexp.MustCompile(`(?i)^([a-z])(?:[-_ .].*)?$`)
var fc2CodePattern = regexp.MustCompile(`(?i)^FC2[-_ ]*(?:PPV[-_ ]*)?([0-9]+)$`)

// ExtractPartIndex 识别明确标记、数字后缀及紧接番号的 A-Z 分部；带连接符的 -C 等版本标记不算分部。
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
	suffix := base[end[1]:]
	if match := numericPartPattern.FindStringSubmatch(suffix); len(match) > 1 {
		index, _ := strconv.Atoi(match[1])
		return index
	}
	if match := letterPartPattern.FindStringSubmatch(suffix); len(match) > 1 {
		// 仅接受紧接番号的一位字母，STAR-684B-C 对应第 2 部，SSIS-562-C 保持未编号。
		return int(strings.ToUpper(match[1])[0]-'A') + 1
	}
	return 0
}
