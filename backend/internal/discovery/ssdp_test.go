package discovery

import (
	"strings"
	"testing"
)

func TestOnlyBoundedCuratedSearchIsAnswered(t *testing.T) {
	search := "M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: \"ssdp:discover\"\r\nMX: 2\r\nST: " + ServiceType + "\r\n\r\n"
	if mx, ok := SearchMX(search); !ok || mx != 2 {
		t.Fatal("valid discovery rejected")
	}
	for _, bad := range []string{strings.Replace(search, "MX: 2", "MX: 999", 1), strings.Replace(search, ServiceType, "ssdp:all", 1), strings.Replace(search, "M-SEARCH", "POST", 1), strings.Repeat("x", 2049)} {
		if _, ok := SearchMX(bad); ok {
			t.Fatal("invalid discovery accepted")
		}
	}
	response := Response("abc", "http://192.168.1.2:9000/discovery/description.xml")
	if !strings.Contains(response, "max-age=120") || !strings.Contains(response, "USN: uuid:abc::"+ServiceType) {
		t.Fatal(response)
	}
	if !strings.Contains(Notification("abc", "http://192.168.1.2:9000/discovery/description.xml", "ssdp:byebye"), "NTS: ssdp:byebye") {
		t.Fatal("missing shutdown announcement")
	}
}
