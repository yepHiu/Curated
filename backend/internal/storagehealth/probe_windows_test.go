//go:build windows

package storagehealth

import "testing"

func TestWindowsRootFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "drive root", path: `E:\`, want: `E:\`},
		{name: "drive child", path: `E:\Movies\Vault`, want: `E:\`},
		{name: "forward slashes", path: `E:/Movies/Vault`, want: `E:\`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := windowsRootFromPath(tt.path); got != tt.want {
				t.Fatalf("windowsRootFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
