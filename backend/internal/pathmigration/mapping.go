// Package pathmigration implements cross-platform, segment-aware filesystem
// path prefix mapping without depending on the host operating system.
package pathmigration

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

type Style string

const (
	StyleWindows Style = "windows"
	StyleUnix    Style = "unix"
)

type Mapping struct {
	FromRoot    string `json:"fromRoot"`
	ToRoot      string `json:"toRoot"`
	SourceStyle Style  `json:"sourceStyle"`
	TargetStyle Style  `json:"targetStyle"`

	from parsedPath
	to   parsedPath
}

type parsedPath struct {
	style          Style
	rootDisplay    string
	rootComparison string
	segments       []string
	segmentCompare []string
}

func NewMapping(fromRoot, toRoot string) (Mapping, error) {
	from, err := parseDetectedAbsolute(strings.TrimSpace(fromRoot))
	if err != nil {
		return Mapping{}, fmt.Errorf("invalid source path prefix: %w", err)
	}
	to, err := parseDetectedAbsolute(strings.TrimSpace(toRoot))
	if err != nil {
		return Mapping{}, fmt.Errorf("invalid target path prefix: %w", err)
	}
	if from.style == to.style && from.comparisonKey() == to.comparisonKey() {
		return Mapping{}, errors.New("source and target path prefixes are equivalent")
	}
	if from.style == to.style && hasPathPrefix(to, from) {
		return Mapping{}, errors.New("target path prefix must not be nested inside source path prefix")
	}
	return Mapping{
		FromRoot:    from.render(nil),
		ToRoot:      to.render(nil),
		SourceStyle: from.style,
		TargetStyle: to.style,
		from:        from,
		to:          to,
	}, nil
}

func (m Mapping) Map(value string) (mapped string, matched bool, err error) {
	candidate, err := parseAbsolute(value, m.SourceStyle)
	if err != nil {
		return "", false, err
	}
	if candidate.rootComparison != m.from.rootComparison || len(candidate.segmentCompare) < len(m.from.segmentCompare) {
		return "", false, nil
	}
	for index, segment := range m.from.segmentCompare {
		if candidate.segmentCompare[index] != segment {
			return "", false, nil
		}
	}
	suffix := candidate.segments[len(m.from.segments):]
	return m.to.render(suffix), true, nil
}

func Normalize(value string, style Style) (string, error) {
	parsed, err := parseAbsolute(value, style)
	if err != nil {
		return "", err
	}
	return parsed.comparisonKey(), nil
}

func HostSupports(style Style) bool {
	if runtime.GOOS == "windows" {
		return style == StyleWindows
	}
	return style == StyleUnix
}

func DetectStyle(value string) (Style, error) {
	switch {
	case strings.HasPrefix(value, `\\`):
		return StyleWindows, nil
	case len(value) >= 3 && isASCIIAlpha(value[0]) && value[1] == ':' && isSeparator(value[2]):
		return StyleWindows, nil
	case strings.HasPrefix(value, "/"):
		return StyleUnix, nil
	default:
		return "", fmt.Errorf("path %q is not an absolute Windows, UNC, or Unix path", value)
	}
}

func parseDetectedAbsolute(value string) (parsedPath, error) {
	style, err := DetectStyle(value)
	if err != nil {
		return parsedPath{}, err
	}
	return parseAbsolute(value, style)
}

func parseAbsolute(value string, style Style) (parsedPath, error) {
	if value == "" {
		return parsedPath{}, errors.New("path is empty")
	}
	switch style {
	case StyleWindows:
		return parseWindowsAbsolute(value)
	case StyleUnix:
		return parseUnixAbsolute(value)
	default:
		return parsedPath{}, fmt.Errorf("unsupported path style %q", style)
	}
}

func hasPathPrefix(candidate, prefix parsedPath) bool {
	if candidate.rootComparison != prefix.rootComparison || len(candidate.segmentCompare) <= len(prefix.segmentCompare) {
		return false
	}
	for index, segment := range prefix.segmentCompare {
		if candidate.segmentCompare[index] != segment {
			return false
		}
	}
	return true
}

func parseWindowsAbsolute(value string) (parsedPath, error) {
	normalized := strings.ReplaceAll(value, "/", `\`)
	var rootDisplay, rootComparison, remainder string
	switch {
	case strings.HasPrefix(normalized, `\\`):
		parts, err := cleanSegments(strings.Split(strings.TrimLeft(normalized, `\`), `\`), StyleWindows)
		if err != nil {
			return parsedPath{}, err
		}
		if len(parts) < 2 {
			return parsedPath{}, errors.New("UNC path requires server and share segments")
		}
		rootDisplay = `\\` + parts[0] + `\` + parts[1]
		rootComparison = strings.ToLower(rootDisplay)
		return newParsedPath(StyleWindows, rootDisplay, rootComparison, parts[2:]), nil
	case len(normalized) >= 3 && isASCIIAlpha(normalized[0]) && normalized[1] == ':' && normalized[2] == '\\':
		rootDisplay = strings.ToUpper(normalized[:1]) + `:\`
		rootComparison = strings.ToLower(rootDisplay)
		remainder = strings.TrimLeft(normalized[3:], `\`)
	default:
		return parsedPath{}, fmt.Errorf("path %q is not an absolute Windows or UNC path", value)
	}
	segments, err := cleanSegments(strings.Split(remainder, `\`), StyleWindows)
	if err != nil {
		return parsedPath{}, err
	}
	return newParsedPath(StyleWindows, rootDisplay, rootComparison, segments), nil
}

func parseUnixAbsolute(value string) (parsedPath, error) {
	if !strings.HasPrefix(value, "/") {
		return parsedPath{}, fmt.Errorf("path %q is not an absolute Unix path", value)
	}
	segments, err := cleanSegments(strings.Split(strings.TrimLeft(value, "/"), "/"), StyleUnix)
	if err != nil {
		return parsedPath{}, err
	}
	return newParsedPath(StyleUnix, "/", "/", segments), nil
}

func newParsedPath(style Style, rootDisplay, rootComparison string, segments []string) parsedPath {
	comparisons := make([]string, len(segments))
	for index, segment := range segments {
		comparisons[index] = segment
		if style == StyleWindows {
			comparisons[index] = strings.ToLower(segment)
		}
	}
	return parsedPath{
		style:          style,
		rootDisplay:    rootDisplay,
		rootComparison: rootComparison,
		segments:       append([]string(nil), segments...),
		segmentCompare: comparisons,
	}
}

func cleanSegments(raw []string, style Style) ([]string, error) {
	segments := make([]string, 0, len(raw))
	for _, segment := range raw {
		switch segment {
		case "", ".":
			continue
		case "..":
			return nil, errors.New("parent path segments are not allowed")
		}
		if style == StyleWindows && strings.Contains(segment, ":") {
			return nil, fmt.Errorf("invalid colon in Windows path segment %q", segment)
		}
		segments = append(segments, segment)
	}
	return segments, nil
}

func (p parsedPath) comparisonKey() string {
	separator := "/"
	if p.style == StyleWindows {
		separator = `\`
	}
	if len(p.segmentCompare) == 0 {
		return p.rootComparison
	}
	return p.rootComparison + separator + strings.Join(p.segmentCompare, separator)
}

func (p parsedPath) render(extraSegments []string) string {
	segments := make([]string, 0, len(p.segments)+len(extraSegments))
	segments = append(segments, p.segments...)
	segments = append(segments, extraSegments...)
	if p.style == StyleUnix {
		if len(segments) == 0 {
			return "/"
		}
		return "/" + strings.Join(segments, "/")
	}
	if len(segments) == 0 {
		return p.rootDisplay
	}
	if strings.HasSuffix(p.rootDisplay, `\`) {
		return p.rootDisplay + strings.Join(segments, `\`)
	}
	return p.rootDisplay + `\` + strings.Join(segments, `\`)
}

func isASCIIAlpha(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func isSeparator(value byte) bool {
	return value == '\\' || value == '/'
}
