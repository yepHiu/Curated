package moviecode

import (
	"errors"
	"regexp"
	"strings"
)

var wishlistCodePattern = regexp.MustCompile(`^[A-Z0-9]+(?:-[A-Z0-9]+)*$`)
var wishlistDigit = regexp.MustCompile(`[0-9]`)

// WishlistIdentity 校验番号并返回显示番号和稳定去重键，不改变现有影片 ID。
func WishlistIdentity(raw string) (string, string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	code = strings.ReplaceAll(code, "_", "-")
	if len(code) < 3 || len(code) > 80 || !wishlistCodePattern.MatchString(code) || !wishlistDigit.MatchString(code) {
		return "", "", errors.New("WISHLIST_INVALID_CODE")
	}
	key := strings.ReplaceAll(code, "-", "")
	// FC2 官方和常见 PPV 写法表示同一编号；保留其余版本/分卷后缀。
	if strings.HasPrefix(key, "FC2PPV") {
		key = "FC2" + strings.TrimPrefix(key, "FC2PPV")
	}
	return code, key, nil
}
