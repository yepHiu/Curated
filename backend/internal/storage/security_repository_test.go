package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newSecurityTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "security.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	return store
}

func parseSecurityTestTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("time.Parse(%q) error = %v", value, err)
	}
	return parsed
}

func TestSecuritySettingsDefaultDisabled(t *testing.T) {
	t.Parallel()

	store := newSecurityTestStore(t)
	settings, err := store.GetAppSecuritySettings(context.Background())
	if err != nil {
		t.Fatalf("GetAppSecuritySettings() error = %v", err)
	}

	if settings.PINEnabled {
		t.Fatal("PINEnabled = true, want false by default")
	}
	if settings.SessionTTLMinutes != 60 {
		t.Fatalf("SessionTTLMinutes = %d, want 60", settings.SessionTTLMinutes)
	}
	if !settings.LANRequiresPIN {
		t.Fatal("LANRequiresPIN = false, want true by default")
	}
	if !settings.LockOnRestart {
		t.Fatal("LockOnRestart = false, want true by default")
	}
}

func TestSecuritySetPINStoresHashWithoutPlaintext(t *testing.T) {
	t.Parallel()

	store := newSecurityTestStore(t)
	if err := store.SetAppPIN(context.Background(), "1234"); err != nil {
		t.Fatalf("SetAppPIN() error = %v", err)
	}

	settings, err := store.GetAppSecuritySettings(context.Background())
	if err != nil {
		t.Fatalf("GetAppSecuritySettings() error = %v", err)
	}
	if !settings.PINEnabled {
		t.Fatal("PINEnabled = false, want true after SetAppPIN")
	}
	if settings.PINHash == "" || settings.PINSalt == "" || settings.PINKDF == "" {
		t.Fatalf("expected hash, salt, and kdf; got hash=%q salt=%q kdf=%q", settings.PINHash, settings.PINSalt, settings.PINKDF)
	}
	if settings.PINHash == "1234" || settings.PINSalt == "1234" {
		t.Fatalf("security settings stored plaintext PIN: %+v", settings)
	}
	if settings.PINLength != 4 {
		t.Fatalf("PINLength = %d, want 4", settings.PINLength)
	}

	ok, err := store.VerifyAppPIN(context.Background(), "1234")
	if err != nil {
		t.Fatalf("VerifyAppPIN(correct) error = %v", err)
	}
	if !ok {
		t.Fatal("VerifyAppPIN(correct) = false, want true")
	}
	ok, err = store.VerifyAppPIN(context.Background(), "654321")
	if err != nil {
		t.Fatalf("VerifyAppPIN(wrong) error = %v", err)
	}
	if ok {
		t.Fatal("VerifyAppPIN(wrong) = true, want false")
	}
}

func TestSecurityVerifyPINBackfillsMissingPINLength(t *testing.T) {
	t.Parallel()

	store := newSecurityTestStore(t)
	if err := store.SetAppPIN(context.Background(), "1234"); err != nil {
		t.Fatalf("SetAppPIN() error = %v", err)
	}
	if _, err := store.db.Exec(`UPDATE app_security_settings SET pin_length = 0 WHERE id = 1`); err != nil {
		t.Fatalf("clear pin_length: %v", err)
	}

	ok, err := store.VerifyAppPIN(context.Background(), "1234")
	if err != nil {
		t.Fatalf("VerifyAppPIN() error = %v", err)
	}
	if !ok {
		t.Fatal("VerifyAppPIN() = false, want true")
	}
	settings, err := store.GetAppSecuritySettings(context.Background())
	if err != nil {
		t.Fatalf("GetAppSecuritySettings() error = %v", err)
	}
	if settings.PINLength != 4 {
		t.Fatalf("PINLength = %d, want 4 after successful legacy unlock", settings.PINLength)
	}
}

func TestSecurityTrustedForeverSessionHasNoExpiry(t *testing.T) {
	t.Parallel()

	store := newSecurityTestStore(t)
	session, err := store.CreateAuthSession(context.Background(), CreateAuthSessionInput{
		ClientKey:      "client-a",
		UserAgent:      "test-browser",
		IP:             "127.0.0.1",
		TrustedForever: true,
		TTLMinutes:     15,
		Now:            parseSecurityTestTime(t, "2026-05-15T10:00:00Z"),
	})
	if err != nil {
		t.Fatalf("CreateAuthSession() error = %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected generated session ID")
	}
	if !session.TrustedForever {
		t.Fatal("TrustedForever = false, want true")
	}
	if session.ExpiresAt != "" {
		t.Fatalf("ExpiresAt = %q, want empty for trusted forever", session.ExpiresAt)
	}

	valid, ok, err := store.GetValidAuthSession(context.Background(), session.ID, parseSecurityTestTime(t, "2036-05-15T10:00:00Z"))
	if err != nil {
		t.Fatalf("GetValidAuthSession() error = %v", err)
	}
	if !ok {
		t.Fatal("trusted forever session should still be valid without expiry")
	}
	if valid.ID != session.ID || !valid.TrustedForever {
		t.Fatalf("valid session = %+v, want trusted session %q", valid, session.ID)
	}
}

func TestSecurityRegularSessionExtendsExpiryOnActivity(t *testing.T) {
	t.Parallel()

	store := newSecurityTestStore(t)
	start := parseSecurityTestTime(t, "2026-05-15T10:00:00Z")
	session, err := store.CreateAuthSession(context.Background(), CreateAuthSessionInput{
		ClientKey:  "regular-client",
		TTLMinutes: 15,
		Now:        start,
	})
	if err != nil {
		t.Fatalf("CreateAuthSession() error = %v", err)
	}

	activityAt := start.Add(10 * time.Minute)
	active, ok, err := store.GetValidAuthSession(context.Background(), session.ID, activityAt)
	if err != nil {
		t.Fatalf("GetValidAuthSession(activity) error = %v", err)
	}
	if !ok {
		t.Fatal("session should still be valid before idle timeout")
	}
	if active.LastSeenAt != nowRFC3339(activityAt) {
		t.Fatalf("LastSeenAt = %q, want %q", active.LastSeenAt, nowRFC3339(activityAt))
	}
	if active.ExpiresAt != nowRFC3339(activityAt.Add(15*time.Minute)) {
		t.Fatalf("ExpiresAt = %q, want idle expiry %q", active.ExpiresAt, nowRFC3339(activityAt.Add(15*time.Minute)))
	}

	_, ok, err = store.GetValidAuthSession(context.Background(), session.ID, activityAt.Add(14*time.Minute))
	if err != nil {
		t.Fatalf("GetValidAuthSession(after extension) error = %v", err)
	}
	if !ok {
		t.Fatal("session should remain valid after activity extends idle expiry")
	}
}

func TestSecurityStartupPolicyKeepsTrustedForeverSessions(t *testing.T) {
	t.Parallel()

	store := newSecurityTestStore(t)
	if err := store.SetAppPIN(context.Background(), "123456"); err != nil {
		t.Fatalf("SetAppPIN() error = %v", err)
	}
	now := parseSecurityTestTime(t, "2026-05-15T10:00:00Z")
	regular, err := store.CreateAuthSession(context.Background(), CreateAuthSessionInput{
		ClientKey:  "regular-client",
		TTLMinutes: 60,
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreateAuthSession(regular) error = %v", err)
	}
	trusted, err := store.CreateAuthSession(context.Background(), CreateAuthSessionInput{
		ClientKey:      "trusted-client",
		TTLMinutes:     60,
		TrustedForever: true,
		Now:            now,
	})
	if err != nil {
		t.Fatalf("CreateAuthSession(trusted) error = %v", err)
	}

	if err := store.ApplyAuthStartupPolicy(context.Background()); err != nil {
		t.Fatalf("ApplyAuthStartupPolicy() error = %v", err)
	}

	if _, ok, err := store.GetValidAuthSession(context.Background(), regular.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("GetValidAuthSession(regular) error = %v", err)
	} else if ok {
		t.Fatal("regular session should be revoked on startup")
	}
	if _, ok, err := store.GetValidAuthSession(context.Background(), trusted.ID, now.Add(365*24*time.Hour)); err != nil {
		t.Fatalf("GetValidAuthSession(trusted) error = %v", err)
	} else if !ok {
		t.Fatal("trusted forever session should survive startup lock")
	}
}
