package storage

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// ErrPINSecretMissing 表示尚未保存 PIN 哈希，不能只把 pinEnabled 设为 true。
var ErrPINSecretMissing = errors.New("pin secret is not configured")

const (
	defaultSessionTTLMinutes = 60
	pinSaltBytes             = 16
	pinHashBytes             = 32
	pinArgonMemoryKiB        = 64 * 1024
	pinArgonTime             = 1
	pinArgonThreads          = 1
	pinKDFArgon2ID           = "argon2id$v=19$m=65536,t=1,p=1"
)

// AppSecuritySettings is the singleton app-lock configuration persisted in SQLite.
type AppSecuritySettings struct {
	PINEnabled        bool
	PINHash           string
	PINSalt           string
	PINKDF            string
	PINLength         int
	SessionTTLMinutes int
	LANRequiresPIN    bool
	LockOnRestart     bool
	CreatedAt         string
	UpdatedAt         string
}

// AppSecuritySettingsPatch partially updates non-secret app-lock settings.
type AppSecuritySettingsPatch struct {
	PINEnabled        *bool
	SessionTTLMinutes *int
	LockOnRestart     *bool
}

// AuthSession is one browser/device unlock session.
type AuthSession struct {
	ID             string
	PublicID       string
	ClientKey      string
	UserAgent      string
	IP             string
	CreatedAt      string
	LastSeenAt     string
	ExpiresAt      string
	TrustedForever bool
	RevokedAt      string
}

// CreateAuthSessionInput describes a new auth session to persist.
type CreateAuthSessionInput struct {
	ID             string
	ClientKey      string
	UserAgent      string
	IP             string
	TTLMinutes     int
	TrustedForever bool
	Now            time.Time
}

// GetAppSecuritySettings returns the singleton security settings row, creating defaults if needed.
func (s *SQLiteStore) GetAppSecuritySettings(ctx context.Context) (AppSecuritySettings, error) {
	if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO app_security_settings (id) VALUES (1)`); err != nil {
		return AppSecuritySettings{}, err
	}
	var settings AppSecuritySettings
	var pinEnabled int
	var lanRequiresPIN int
	var lockOnRestart int
	err := s.db.QueryRowContext(ctx, `
		SELECT
			pin_enabled,
			pin_hash,
			pin_salt,
			pin_kdf,
			pin_length,
			session_ttl_minutes,
			lan_requires_pin,
			lock_on_restart,
			created_at,
			updated_at
		FROM app_security_settings
		WHERE id = 1
	`).Scan(
		&pinEnabled,
		&settings.PINHash,
		&settings.PINSalt,
		&settings.PINKDF,
		&settings.PINLength,
		&settings.SessionTTLMinutes,
		&lanRequiresPIN,
		&lockOnRestart,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err != nil {
		return AppSecuritySettings{}, err
	}
	settings.PINEnabled = pinEnabled != 0
	settings.LANRequiresPIN = lanRequiresPIN != 0
	settings.LockOnRestart = lockOnRestart != 0
	if settings.SessionTTLMinutes <= 0 {
		settings.SessionTTLMinutes = defaultSessionTTLMinutes
	}
	return settings, nil
}

// SetAppPIN hashes and enables the app PIN. The plaintext PIN is never stored.
func (s *SQLiteStore) SetAppPIN(ctx context.Context, pin string) error {
	pin = strings.TrimSpace(pin)
	if pin == "" {
		return errors.New("pin is required")
	}
	salt, err := randomURLBase64(pinSaltBytes)
	if err != nil {
		return err
	}
	hash, err := hashPIN(pin, salt)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO app_security_settings (
			id,
			pin_enabled,
			pin_hash,
			pin_salt,
			pin_kdf,
			pin_length,
			updated_at
		) VALUES (1, 1, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			pin_enabled = 1,
			pin_hash = excluded.pin_hash,
			pin_salt = excluded.pin_salt,
			pin_kdf = excluded.pin_kdf,
			pin_length = excluded.pin_length,
			updated_at = excluded.updated_at
	`, hash, salt, pinKDFArgon2ID, len(pin), nowRFC3339(time.Now()))
	return err
}

// VerifyAppPIN compares a candidate PIN against the stored hash.
func (s *SQLiteStore) VerifyAppPIN(ctx context.Context, pin string) (bool, error) {
	settings, err := s.GetAppSecuritySettings(ctx)
	if err != nil {
		return false, err
	}
	if !settings.PINEnabled || settings.PINHash == "" || settings.PINSalt == "" {
		return false, nil
	}
	trimmed := strings.TrimSpace(pin)
	candidate, err := hashPIN(trimmed, settings.PINSalt)
	if err != nil {
		return false, err
	}
	ok := subtle.ConstantTimeCompare([]byte(candidate), []byte(settings.PINHash)) == 1
	if ok && settings.PINLength != len(trimmed) {
		_, err = s.db.ExecContext(ctx, `
			UPDATE app_security_settings
			SET pin_length = ?, updated_at = ?
			WHERE id = 1
		`, len(trimmed), nowRFC3339(time.Now()))
		if err != nil {
			return false, err
		}
	}
	return ok, nil
}

// PatchAppSecuritySettings 部分更新 PIN 锁的非密钥设置。
// 关闭 PIN 时会同时清空哈希并撤销全部会话，之后必须重新 setup 才能启用。
func (s *SQLiteStore) PatchAppSecuritySettings(ctx context.Context, patch AppSecuritySettingsPatch) (AppSecuritySettings, error) {
	current, err := s.GetAppSecuritySettings(ctx)
	if err != nil {
		return AppSecuritySettings{}, err
	}
	pinEnabled := current.PINEnabled
	if patch.PINEnabled != nil {
		pinEnabled = *patch.PINEnabled
	}
	if pinEnabled && (current.PINHash == "" || current.PINSalt == "") {
		return AppSecuritySettings{}, ErrPINSecretMissing
	}
	ttl := current.SessionTTLMinutes
	if patch.SessionTTLMinutes != nil {
		ttl = normalizeSessionTTLMinutes(*patch.SessionTTLMinutes)
	}
	lockOnRestart := current.LockOnRestart
	if patch.LockOnRestart != nil {
		lockOnRestart = *patch.LockOnRestart
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AppSecuritySettings{}, err
	}
	defer func() { _ = tx.Rollback() }()

	disablingPIN := current.PINEnabled && !pinEnabled
	if disablingPIN {
		_, err = tx.ExecContext(ctx, `
			UPDATE app_security_settings
			SET
				pin_enabled = 0,
				pin_hash = '',
				pin_salt = '',
				pin_kdf = '',
				pin_length = 0,
				session_ttl_minutes = ?,
				lan_requires_pin = 1,
				lock_on_restart = ?,
				updated_at = ?
			WHERE id = 1
		`, ttl, boolToInt(lockOnRestart), nowRFC3339(time.Now()))
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE app_security_settings
			SET
				pin_enabled = ?,
				session_ttl_minutes = ?,
				lan_requires_pin = 1,
				lock_on_restart = ?,
				updated_at = ?
			WHERE id = 1
		`, boolToInt(pinEnabled), ttl, boolToInt(lockOnRestart), nowRFC3339(time.Now()))
	}
	if err != nil {
		return AppSecuritySettings{}, err
	}
	if disablingPIN {
		if _, err := tx.ExecContext(ctx, `
			UPDATE auth_sessions
			SET revoked_at = ?
			WHERE revoked_at = ''
		`, nowRFC333Nanos(time.Now().UTC())); err != nil {
			return AppSecuritySettings{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return AppSecuritySettings{}, err
	}
	return s.GetAppSecuritySettings(ctx)
}

// CreateAuthSession creates a server-side unlock session.
func (s *SQLiteStore) CreateAuthSession(ctx context.Context, input CreateAuthSessionInput) (AuthSession, error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		var err error
		id, err = randomURLBase64(32)
		if err != nil {
			return AuthSession{}, err
		}
	}
	publicID, err := randomURLBase64(16)
	if err != nil {
		return AuthSession{}, err
	}
	clientKey := strings.TrimSpace(input.ClientKey)
	if clientKey == "" {
		return AuthSession{}, errors.New("client key is required")
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	createdAt := nowRFC3339(now)
	expiresAt := ""
	if !input.TrustedForever {
		ttl := normalizeSessionTTLMinutes(input.TTLMinutes)
		expiresAt = nowRFC3339(now.Add(time.Duration(ttl) * time.Minute))
	}
	session := AuthSession{
		ID:             id,
		PublicID:       publicID,
		ClientKey:      clientKey,
		UserAgent:      strings.TrimSpace(input.UserAgent),
		IP:             strings.TrimSpace(input.IP),
		CreatedAt:      createdAt,
		LastSeenAt:     createdAt,
		ExpiresAt:      expiresAt,
		TrustedForever: input.TrustedForever,
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO auth_sessions (
			id,
			public_id,
			client_key,
			user_agent,
			ip,
			created_at,
			last_seen_at,
			expires_at,
			trusted_forever,
			revoked_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '')
	`,
		session.ID,
		session.PublicID,
		session.ClientKey,
		session.UserAgent,
		session.IP,
		session.CreatedAt,
		session.LastSeenAt,
		session.ExpiresAt,
		boolToInt(session.TrustedForever),
	)
	if err != nil {
		return AuthSession{}, err
	}
	return session, nil
}

// GetAuthSession returns a session by id regardless of expiry or revocation.
func (s *SQLiteStore) GetAuthSession(ctx context.Context, id string) (AuthSession, bool, error) {
	var session AuthSession
	var trustedForever int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, public_id, client_key, user_agent, ip, created_at, last_seen_at, expires_at, trusted_forever, revoked_at
		FROM auth_sessions
		WHERE id = ?
	`, strings.TrimSpace(id)).Scan(
		&session.ID,
		&session.PublicID,
		&session.ClientKey,
		&session.UserAgent,
		&session.IP,
		&session.CreatedAt,
		&session.LastSeenAt,
		&session.ExpiresAt,
		&trustedForever,
		&session.RevokedAt,
	)
	if err == sql.ErrNoRows {
		return AuthSession{}, false, nil
	}
	if err != nil {
		return AuthSession{}, false, err
	}
	session.TrustedForever = trustedForever != 0
	return session, true, nil
}

// GetValidAuthSession returns a session only when it is not revoked and not expired.
func (s *SQLiteStore) GetValidAuthSession(ctx context.Context, id string, now time.Time) (AuthSession, bool, error) {
	session, ok, err := s.GetAuthSession(ctx, id)
	if err != nil || !ok {
		return AuthSession{}, ok, err
	}
	if strings.TrimSpace(session.RevokedAt) != "" {
		return AuthSession{}, false, nil
	}
	if !session.TrustedForever {
		expiresAt, err := time.Parse(time.RFC3339Nano, session.ExpiresAt)
		if err != nil {
			return AuthSession{}, false, nil
		}
		if !now.UTC().Before(expiresAt) {
			return AuthSession{}, false, nil
		}
		lastSeenAt, err := time.Parse(time.RFC3339Nano, session.LastSeenAt)
		if err != nil {
			lastSeenAt, err = time.Parse(time.RFC3339Nano, session.CreatedAt)
			if err != nil {
				return AuthSession{}, false, nil
			}
		}
		idleDuration := expiresAt.Sub(lastSeenAt)
		if idleDuration <= 0 {
			return AuthSession{}, false, nil
		}
		session.ExpiresAt = nowRFC3339(now.UTC().Add(idleDuration))
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET last_seen_at = ?,
			expires_at = ?
		WHERE id = ?
	`, nowRFC3339(now.UTC()), session.ExpiresAt, session.ID)
	if err != nil {
		return AuthSession{}, false, err
	}
	session.LastSeenAt = nowRFC3339(now.UTC())
	return session, true, nil
}

// RevokeAuthSession 将指定会话标记为已撤销。
func (s *SQLiteStore) RevokeAuthSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = ?
		WHERE id = ?
	`, nowRFC333Nanos(time.Now().UTC()), strings.TrimSpace(id))
	return err
}

// ListTrustedAuthSessions returns active trusted-forever sessions without exposing their bearer-token IDs.
func (s *SQLiteStore) ListTrustedAuthSessions(ctx context.Context) ([]AuthSession, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, public_id, client_key, user_agent, ip, created_at, last_seen_at, expires_at, trusted_forever, revoked_at
		FROM auth_sessions
		WHERE trusted_forever = 1
			AND revoked_at = ''
		ORDER BY last_seen_at DESC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	sessions := make([]AuthSession, 0)
	for rows.Next() {
		var session AuthSession
		var trustedForever int
		if err := rows.Scan(
			&session.ID,
			&session.PublicID,
			&session.ClientKey,
			&session.UserAgent,
			&session.IP,
			&session.CreatedAt,
			&session.LastSeenAt,
			&session.ExpiresAt,
			&trustedForever,
			&session.RevokedAt,
		); err != nil {
			return nil, err
		}
		session.TrustedForever = trustedForever != 0
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

// RevokeTrustedAuthSessionByPublicID revokes one active trusted session by its non-secret public ID.
func (s *SQLiteStore) RevokeTrustedAuthSessionByPublicID(ctx context.Context, publicID string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = ?
		WHERE public_id = ?
			AND trusted_forever = 1
			AND revoked_at = ''
	`, nowRFC333Nanos(time.Now().UTC()), strings.TrimSpace(publicID))
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count > 0, err
}

// RevokeOtherTrustedAuthSessions revokes every active trusted session except currentSessionID.
func (s *SQLiteStore) RevokeOtherTrustedAuthSessions(ctx context.Context, currentSessionID string) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = ?
		WHERE trusted_forever = 1
			AND revoked_at = ''
			AND id != ?
	`, nowRFC333Nanos(time.Now().UTC()), strings.TrimSpace(currentSessionID))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ApplyAuthStartupPolicy revokes regular sessions when restart locking is enabled.
// Trusted-forever sessions intentionally survive restarts.
func (s *SQLiteStore) ApplyAuthStartupPolicy(ctx context.Context) error {
	settings, err := s.GetAppSecuritySettings(ctx)
	if err != nil {
		return err
	}
	if !settings.PINEnabled || !settings.LockOnRestart {
		return nil
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = ?
		WHERE revoked_at = ''
			AND trusted_forever = 0
	`, nowRFC333Nanos(time.Now().UTC()))
	return err
}

func normalizeSessionTTLMinutes(value int) int {
	if value <= 0 {
		return defaultSessionTTLMinutes
	}
	if value > 30*24*60 {
		return 30 * 24 * 60
	}
	return value
}

func hashPIN(pin string, saltEncoded string) (string, error) {
	salt, err := base64.RawURLEncoding.DecodeString(saltEncoded)
	if err != nil {
		return "", fmt.Errorf("decode pin salt: %w", err)
	}
	sum := argon2.IDKey([]byte(pin), salt, pinArgonTime, pinArgonMemoryKiB, pinArgonThreads, pinHashBytes)
	return base64.RawURLEncoding.EncodeToString(sum), nil
}

func randomURLBase64(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func nowRFC3339(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func nowRFC333Nanos(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
