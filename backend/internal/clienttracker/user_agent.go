package clienttracker

import (
	"regexp"
	"strings"
)

type DeviceType string

const (
	DeviceTypeDesktop DeviceType = "desktop"
	DeviceTypeLaptop  DeviceType = "laptop"
	DeviceTypeMobile  DeviceType = "mobile"
	DeviceTypeTablet  DeviceType = "tablet"
	DeviceTypeTool    DeviceType = "tool"
	DeviceTypeUnknown DeviceType = "unknown"
)

type UserAgentInfo struct {
	Browser        string
	BrowserVersion string
	OS             string
	OSVersion      string
	DeviceType     DeviceType
}

var versionTokenPattern = regexp.MustCompile(`([A-Za-z]+)/([0-9][0-9A-Za-z._-]*)`)

func ParseUserAgent(userAgent string) UserAgentInfo {
	ua := strings.TrimSpace(userAgent)
	if ua == "" {
		return UserAgentInfo{
			Browser:    "Unknown Client",
			OS:         "Unknown OS",
			DeviceType: DeviceTypeUnknown,
		}
	}

	info := UserAgentInfo{
		Browser:    "Unknown Client",
		OS:         "Unknown OS",
		DeviceType: DeviceTypeUnknown,
	}
	lower := strings.ToLower(ua)

	if strings.HasPrefix(lower, "curl/") {
		info.Browser = "curl"
		info.BrowserVersion = versionAfter(ua, "curl/")
		info.DeviceType = DeviceTypeTool
		return info
	}
	if strings.HasPrefix(lower, "python-requests/") {
		info.Browser = "python-requests"
		info.BrowserVersion = versionAfter(ua, "python-requests/")
		info.DeviceType = DeviceTypeTool
		return info
	}
	if strings.HasPrefix(lower, "wget/") || strings.HasPrefix(lower, "httpie/") {
		match := versionTokenPattern.FindStringSubmatch(ua)
		if len(match) == 3 {
			info.Browser = match[1]
			info.BrowserVersion = match[2]
		}
		info.DeviceType = DeviceTypeTool
		return info
	}

	switch {
	case strings.Contains(ua, "Edg/"):
		info.Browser = "Edge"
		info.BrowserVersion = versionAfter(ua, "Edg/")
	case strings.Contains(ua, "Firefox/"):
		info.Browser = "Firefox"
		info.BrowserVersion = versionAfter(ua, "Firefox/")
	case strings.Contains(ua, "Chrome/") || strings.Contains(ua, "CriOS/"):
		info.Browser = "Chrome"
		if strings.Contains(ua, "CriOS/") {
			info.BrowserVersion = versionAfter(ua, "CriOS/")
		} else {
			info.BrowserVersion = versionAfter(ua, "Chrome/")
		}
	case strings.Contains(ua, "Version/") && strings.Contains(ua, "Safari/"):
		info.Browser = "Safari"
		info.BrowserVersion = versionAfter(ua, "Version/")
	}

	switch {
	case strings.Contains(ua, "iPad"):
		info.OS = "iPadOS"
		info.OSVersion = osVersionAfter(ua, "CPU OS ")
		info.DeviceType = DeviceTypeTablet
	case strings.Contains(ua, "iPhone"):
		info.OS = "iOS"
		info.OSVersion = osVersionAfter(ua, "CPU iPhone OS ")
		info.DeviceType = DeviceTypeMobile
	case strings.Contains(ua, "Android"):
		info.OS = "Android"
		info.OSVersion = osVersionAfter(ua, "Android ")
		if strings.Contains(ua, "Mobile") {
			info.DeviceType = DeviceTypeMobile
		} else {
			info.DeviceType = DeviceTypeTablet
		}
	case strings.Contains(ua, "Mac OS X"):
		info.OS = "macOS"
		info.OSVersion = osVersionAfter(ua, "Mac OS X ")
		info.DeviceType = DeviceTypeLaptop
	case strings.Contains(ua, "Windows NT"):
		info.OS = "Windows"
		info.OSVersion = osVersionAfter(ua, "Windows NT ")
		info.DeviceType = DeviceTypeDesktop
	case strings.Contains(ua, "Linux"):
		info.OS = "Linux"
		info.DeviceType = DeviceTypeDesktop
	}

	if info.DeviceType == DeviceTypeUnknown && info.Browser != "Unknown Client" {
		info.DeviceType = DeviceTypeDesktop
	}

	return info
}

func versionAfter(s string, token string) string {
	idx := strings.Index(s, token)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(token):]
	for i, r := range rest {
		if r == ' ' || r == ';' || r == ')' {
			return rest[:i]
		}
	}
	return rest
}

func osVersionAfter(s string, token string) string {
	v := versionAfter(s, token)
	v = strings.TrimRight(v, ";)")
	return strings.ReplaceAll(v, "_", ".")
}
