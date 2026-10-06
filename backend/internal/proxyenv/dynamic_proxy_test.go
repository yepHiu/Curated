package proxyenv

import (
	"net/http"
	"testing"
)

// TestDynamicProxyEnvironment 确保修改代理后下一次查询立即读取新值。
func TestDynamicProxyEnvironment(t *testing.T) {
	t.Setenv("NO_PROXY", "")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:7890")
	request, _ := http.NewRequest(http.MethodGet, "https://javten.com/search", nil)
	first, err := ProxyFromEnvironment(request)
	if err != nil || first == nil || first.Port() != "7890" {
		t.Fatalf("first proxy %v %v", first, err)
	}
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:7891")
	second, err := ProxyFromEnvironment(request)
	if err != nil || second == nil || second.Port() != "7891" {
		t.Fatalf("stale proxy %v %v", second, err)
	}
}
