package clienttracker

import "testing"

func TestParseUserAgentRecognizesBrowsersAndOperatingSystems(t *testing.T) {
	tests := []struct {
		name       string
		ua         string
		browser    string
		os         string
		deviceType DeviceType
	}{
		{
			name:       "edge on windows",
			ua:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/132.0.0.0",
			browser:    "Edge",
			os:         "Windows",
			deviceType: DeviceTypeDesktop,
		},
		{
			name:       "safari on iphone",
			ua:         "Mozilla/5.0 (iPhone; CPU iPhone OS 18_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Mobile/15E148 Safari/604.1",
			browser:    "Safari",
			os:         "iOS",
			deviceType: DeviceTypeMobile,
		},
		{
			name:       "firefox on macos",
			ua:         "Mozilla/5.0 (Macintosh; Intel Mac OS X 15.2; rv:135.0) Gecko/20100101 Firefox/135.0",
			browser:    "Firefox",
			os:         "macOS",
			deviceType: DeviceTypeLaptop,
		},
		{
			name:       "curl",
			ua:         "curl/8.7.1",
			browser:    "curl",
			os:         "Unknown OS",
			deviceType: DeviceTypeTool,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseUserAgent(tt.ua)
			if got.Browser != tt.browser {
				t.Fatalf("Browser = %q, want %q", got.Browser, tt.browser)
			}
			if got.OS != tt.os {
				t.Fatalf("OS = %q, want %q", got.OS, tt.os)
			}
			if got.DeviceType != tt.deviceType {
				t.Fatalf("DeviceType = %q, want %q", got.DeviceType, tt.deviceType)
			}
		})
	}
}
