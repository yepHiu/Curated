//go:build windows

package desktop

import "testing"

type fakeRunKeyStore struct {
	setName    string
	setValue   string
	deleteName string
}

func (s *fakeRunKeyStore) SetStringValue(name string, value string) error {
	s.setName = name
	s.setValue = value
	return nil
}

func (s *fakeRunKeyStore) DeleteValue(name string) error {
	s.deleteName = name
	return nil
}

func TestBuildLaunchAtLoginCommand_QuotesExecutableAndAppendsSilentTrayFlags(t *testing.T) {
	t.Parallel()

	got, err := buildLaunchAtLoginCommand(`C:\Program Files\Curated\curated.exe`)
	if err != nil {
		t.Fatal(err)
	}

	want := `"C:\Program Files\Curated\curated.exe" -mode tray -autostart`
	if got != want {
		t.Fatalf("buildLaunchAtLoginCommand() = %q, want %q", got, want)
	}
}

func TestSyncLaunchAtLoginStore_EnableWritesRunValue(t *testing.T) {
	t.Parallel()

	store := &fakeRunKeyStore{}
	command := `"C:\Program Files\Curated\curated.exe" -mode tray -autostart`

	if err := syncLaunchAtLoginStore(store, true, "Curated", command); err != nil {
		t.Fatal(err)
	}

	if store.setName != "Curated" {
		t.Fatalf("set name = %q, want Curated", store.setName)
	}
	if store.setValue != command {
		t.Fatalf("set value = %q, want %q", store.setValue, command)
	}
	if store.deleteName != "" {
		t.Fatalf("delete name = %q, want empty", store.deleteName)
	}
}

func TestSyncLaunchAtLoginStore_DisableDeletesRunValue(t *testing.T) {
	t.Parallel()

	store := &fakeRunKeyStore{}

	if err := syncLaunchAtLoginStore(store, false, "Curated", "ignored"); err != nil {
		t.Fatal(err)
	}

	if store.deleteName != "Curated" {
		t.Fatalf("delete name = %q, want Curated", store.deleteName)
	}
	if store.setName != "" {
		t.Fatalf("set name = %q, want empty", store.setName)
	}
}
