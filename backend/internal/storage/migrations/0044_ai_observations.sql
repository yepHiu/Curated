CREATE TABLE ai_runs (
 id TEXT PRIMARY KEY,
 started_at TEXT NOT NULL,
 channel TEXT NOT NULL,
 action TEXT NOT NULL DEFAULT '',
 session_id TEXT NOT NULL DEFAULT '',
 provider TEXT NOT NULL,
 model TEXT NOT NULL,
 prompt_version TEXT NOT NULL,
 status TEXT NOT NULL,
 error_code TEXT NOT NULL DEFAULT '',
 duration_ms INTEGER NOT NULL,
 first_text_ms INTEGER,
 model_calls INTEGER NOT NULL,
 usage_calls INTEGER NOT NULL,
 tool_calls INTEGER NOT NULL,
 prompt_tokens INTEGER NOT NULL,
 completion_tokens INTEGER NOT NULL,
 total_tokens INTEGER NOT NULL
);
CREATE INDEX idx_ai_runs_started ON ai_runs(started_at DESC, id DESC);
