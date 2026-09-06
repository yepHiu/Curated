CREATE TABLE ai_apply_receipts (
  token_hash TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  tool_name TEXT NOT NULL,
  args_hash TEXT NOT NULL,
  result_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_ai_apply_receipts_session ON ai_apply_receipts(session_id);
