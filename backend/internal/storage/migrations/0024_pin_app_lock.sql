CREATE TABLE IF NOT EXISTS app_security_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  pin_enabled INTEGER NOT NULL DEFAULT 0,
  pin_hash TEXT NOT NULL DEFAULT '',
  pin_salt TEXT NOT NULL DEFAULT '',
  pin_kdf TEXT NOT NULL DEFAULT '',
  session_ttl_minutes INTEGER NOT NULL DEFAULT 60,
  lan_requires_pin INTEGER NOT NULL DEFAULT 1,
  lock_on_restart INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

INSERT OR IGNORE INTO app_security_settings (id) VALUES (1);

CREATE TABLE IF NOT EXISTS auth_sessions (
  id TEXT PRIMARY KEY,
  client_key TEXT NOT NULL,
  user_agent TEXT NOT NULL DEFAULT '',
  ip TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  expires_at TEXT NOT NULL DEFAULT '',
  trusted_forever INTEGER NOT NULL DEFAULT 0,
  revoked_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_client_key ON auth_sessions(client_key);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_last_seen_at ON auth_sessions(last_seen_at);
