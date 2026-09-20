CREATE TABLE ai_context_checkpoints (
    session_id TEXT PRIMARY KEY REFERENCES ai_chat_sessions(id) ON DELETE CASCADE,
    through_seq INTEGER NOT NULL,
    summary TEXT NOT NULL
);
