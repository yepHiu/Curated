ALTER TABLE auth_sessions
ADD COLUMN public_id TEXT NOT NULL DEFAULT '';

UPDATE auth_sessions
SET public_id = lower(hex(randomblob(16)))
WHERE public_id = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_sessions_public_id
  ON auth_sessions(public_id);

-- Curated now uses one global PIN-lock policy for local and LAN clients.
-- Keep the legacy column pinned to true for backward-compatible status responses.
UPDATE app_security_settings
SET lan_requires_pin = 1
WHERE id = 1;
