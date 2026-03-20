package scanner

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Site domain prefixes commonly prepended to filenames by download tools.
var sitePrefixPattern = regexp.MustCompile(`(?i)^[a-z0-9]+\.[a-z]{2,4}@`)

// Suffix tags appended to filenames (quality, censorship, etc.)
var suffixCleanPattern = regexp.MustCompile(`(?i)[-_]+(C|UC|U|HD|FHD|4K|uncensored|censored|leak|leaked)$`)

// Ordered by specificity: special formats first, then standard last.
var numberPatterns = []struct {
	re     *regexp.Regexp
	format func(matches []string) string
}{
	// FC2-PPV / FC2PPV / FC2 (numeric ID, 5-7 digits)
	{
		re: regexp.MustCompile(`(?i)\b(?:FC2[-_ ]?(?:PPV[-_ ]?)?|fc2)(\d{5,7})\b`),
		format: func(m []string) string {
			return "FC2-" + m[1]
		},
	},
	// HEYZO
	{
		re:     regexp.MustCompile(`(?i)\b(HEYZO)[-_ ]?(\d{3,6})\b`),
		format: func(m []string) string { return "HEYZO-" + m[2] },
	},
	// Tokyo-Hot
	{
		re:     regexp.MustCompile(`(?i)\b(TOKYO[-_ ]?HOT)[-_ ]?([a-z0-9-]+)\b`),
		format: func(m []string) string { return "TOKYO-HOT-" + strings.ToUpper(m[2]) },
	},
	// 1Pondo (numeric date-based ID)
	{
		re:     regexp.MustCompile(`(?i)\b(1PONDO|1PON)[-_ ]?(\d{6,10}[-_ ]?\d{0,4})\b`),
		format: func(m []string) string { return "1PONDO-" + strings.ReplaceAll(m[2], " ", "") },
	},
	// Caribbeancom
	{
		re:     regexp.MustCompile(`(?i)\b(CARIBBEANCOM|CARIB)[-_ ]?(\d{6,10}[-_ ]?\d{0,4})\b`),
		format: func(m []string) string { return "CARIBBEANCOM-" + strings.ReplaceAll(m[2], " ", "") },
	},
	// Standard番号: 2-6 alpha prefix + 2-5 digit suffix (e.g. IPZZ-788, ABP-123, START-483)
	{
		re:     regexp.MustCompile(`(?i)\b([a-z]{2,6})[-_ ]?(\d{2,5})\b`),
		format: func(m []string) string { return strings.ToUpper(m[1]) + "-" + m[2] },
	},
}

// CleanFilename strips site-domain prefixes and quality/censorship suffixes
// from a raw filename (without extension), returning the cleaned base name
// ready for number extraction.
func CleanFilename(rawName string) string {
	name := strings.TrimSuffix(rawName, filepath.Ext(rawName))
	name = strings.TrimSpace(name)

	// Strip site domain prefix: "489155.com@IPZZ-788" → "IPZZ-788"
	name = sitePrefixPattern.ReplaceAllString(name, "")

	// Strip trailing quality/censorship markers: "EBWH-287-C" → "EBWH-287"
	name = suffixCleanPattern.ReplaceAllString(name, "")

	return strings.TrimSpace(name)
}

// ExtractNumber parses a番号 from a filename.
// It first cleans the filename, then tries each pattern in priority order.
func ExtractNumber(filename string) string {
	cleaned := CleanFilename(filename)
	if cleaned == "" {
		return ""
	}

	for _, p := range numberPatterns {
		matches := p.re.FindStringSubmatch(cleaned)
		if len(matches) > 1 {
			return p.format(matches)
		}
	}

	return ""
}
