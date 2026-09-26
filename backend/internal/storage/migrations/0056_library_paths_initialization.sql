CREATE TABLE library_paths_initialization (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  initialized INTEGER NOT NULL CHECK (initialized IN (0, 1))
);

-- Existing databases own their path list, including an intentionally empty list.
-- Migrate marks only a fresh database as eligible for initial configuration seeding.
INSERT INTO library_paths_initialization (id, initialized) VALUES (1, 1);
