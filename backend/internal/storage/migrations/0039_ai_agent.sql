CREATE TABLE IF NOT EXISTS ai_chat_sessions (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ai_chat_messages (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  tool_name TEXT NOT NULL DEFAULT '',
  tool_call_id TEXT NOT NULL DEFAULT '',
  seq INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (session_id) REFERENCES ai_chat_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_session_seq
  ON ai_chat_messages(session_id, seq);

CREATE TABLE IF NOT EXISTS ai_tool_invocations (
  id TEXT PRIMARY KEY,
  channel TEXT NOT NULL,
  session_id TEXT NOT NULL DEFAULT '',
  tool_name TEXT NOT NULL,
  permission TEXT NOT NULL,
  args_summary TEXT NOT NULL DEFAULT '',
  result TEXT NOT NULL,
  error_code TEXT NOT NULL DEFAULT '',
  duration_ms INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_tool_invocations_created
  ON ai_tool_invocations(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ai_chat_sessions_updated
  ON ai_chat_sessions(updated_at DESC);
