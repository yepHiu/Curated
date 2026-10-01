-- Preserve a reversible naming audit independently of classification decisions.
CREATE TABLE topic_label_renames (
 id INTEGER PRIMARY KEY,
 topic_id TEXT NOT NULL REFERENCES library_topics(id),
 old_name TEXT NOT NULL,
 new_name TEXT NOT NULL,
 created_at TEXT NOT NULL
);
