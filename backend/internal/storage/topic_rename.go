package storage

import (
	"context"
	"curated-backend/internal/contracts"
	"encoding/json"
	"fmt"
	"strings"
)

// RenameLibraryTopic changes the user label while retaining topic/tag IDs and membership.
// It rejects live jobs and conflicting names; historical snapshots keep equivalent undo semantics.
func (s *SQLiteStore) RenameLibraryTopic(ctx context.Context, id, expectedName, name string) error {
	normalized, err := NormalizeUserTagsForPatch([]string{name})
	if err != nil || len(normalized) != 1 || normalized[0] != name {
		return ErrInvalidUserTags
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var active int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_tag_organization_jobs WHERE status IN ('queued','running')`).Scan(&active); err != nil {
		return err
	}
	if active > 0 {
		return fmt.Errorf("stop organization before renaming topics")
	}
	var tagID int64
	var old, aliasesRaw string
	if err = tx.QueryRowContext(ctx, `SELECT t.id,t.name,p.aliases_json FROM library_topics p JOIN tags t ON t.id=p.tag_id AND t.type='user' WHERE p.id=?`, id).Scan(&tagID, &old, &aliasesRaw); err != nil {
		return err
	}
	if old != expectedName {
		return ErrAIWriteConflict
	}
	if old == name {
		return tx.Commit()
	}
	var conflicts int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE type='user' AND lower(name)=lower(?) AND id!=?`, name, tagID).Scan(&conflicts); err != nil {
		return err
	}
	if conflicts > 0 {
		return fmt.Errorf("user label already exists")
	}
	// A different topic's alias must not become an ambiguous canonical label.
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM library_topics p,json_each(p.aliases_json) a WHERE p.id!=? AND lower(a.value)=lower(?)`, id, name).Scan(&conflicts); err != nil {
		return err
	}
	if conflicts > 0 {
		return fmt.Errorf("topic alias conflict")
	}
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM movie_user_tag_decisions WHERE name=?`, name).Scan(&conflicts); err != nil {
		return err
	}
	if conflicts > 0 {
		return fmt.Errorf("manual label decision conflict")
	}
	var aliases []string
	if err = json.Unmarshal([]byte(aliasesRaw), &aliases); err != nil {
		return err
	}
	nextAliases := []string{}
	for _, alias := range append(aliases, old) {
		if alias == name {
			continue
		}
		found := false
		for _, seen := range nextAliases {
			if strings.EqualFold(alias, seen) {
				found = true
			}
		}
		if !found {
			nextAliases = append(nextAliases, alias)
		}
	}
	if len(nextAliases) > 20 {
		return fmt.Errorf("too many topic aliases")
	}
	encoded, _ := json.Marshal(nextAliases)
	if _, err = tx.ExecContext(ctx, `UPDATE tags SET name=? WHERE id=? AND type='user'`, name, tagID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE library_topics SET aliases_json=? WHERE id=?`, string(encoded), id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE movie_user_tag_decisions SET name=? WHERE name=?`, name, old); err != nil {
		return err
	}
	// Revisions remain unchanged: the same tag IDs are attached and no movie decision changes.
	// Transform only structured label fields, never evidence quotes or NFO/source fields.
	type record struct{ job, movie, before, after, evidence string }
	records := []record{}
	rows, err := tx.QueryContext(ctx, `SELECT job_id,movie_id,before_json,after_json FROM ai_tag_organization_changes`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var r record
		if err = rows.Scan(&r.job, &r.movie, &r.before, &r.after); err != nil {
			rows.Close()
			return err
		}
		records = append(records, r)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	replace := func(raw string) (string, error) {
		var names []string
		if err := json.Unmarshal([]byte(raw), &names); err != nil {
			return "", err
		}
		for i, n := range names {
			if n == old {
				names[i] = name
			}
		}
		b, err := json.Marshal(names)
		return string(b), err
	}
	for _, r := range records {
		before, e := replace(r.before)
		if e != nil {
			return e
		}
		after, e := replace(r.after)
		if e != nil {
			return e
		}
		if _, err = tx.ExecContext(ctx, `UPDATE ai_tag_organization_changes SET before_json=?,after_json=? WHERE job_id=? AND movie_id=?`, before, after, r.job, r.movie); err != nil {
			return err
		}
	}
	records = nil
	rows, err = tx.QueryContext(ctx, `SELECT job_id,movie_id,evidence_json FROM ai_tag_organization_items`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var r record
		if err = rows.Scan(&r.job, &r.movie, &r.evidence); err != nil {
			rows.Close()
			return err
		}
		records = append(records, r)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, r := range records {
		var evidence []contracts.TopicEvidenceDTO
		if err = json.Unmarshal([]byte(r.evidence), &evidence); err != nil {
			return err
		}
		changed := false
		for i := range evidence {
			if evidence[i].Topic == old {
				evidence[i].Topic = name
				changed = true
			}
		}
		if changed {
			b, _ := json.Marshal(evidence)
			if _, err = tx.ExecContext(ctx, `UPDATE ai_tag_organization_items SET evidence_json=? WHERE job_id=? AND movie_id=?`, string(b), r.job, r.movie); err != nil {
				return err
			}
		}
	}
	type vocabulary struct{ id, raw string }
	vocabs := []vocabulary{}
	rows, err = tx.QueryContext(ctx, `SELECT id,vocabulary_json FROM ai_tag_organization_jobs`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var v vocabulary
		if err = rows.Scan(&v.id, &v.raw); err != nil {
			rows.Close()
			return err
		}
		vocabs = append(vocabs, v)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, v := range vocabs {
		var defs []TopicDefinition
		if err = json.Unmarshal([]byte(v.raw), &defs); err != nil {
			return err
		}
		changed := false
		for i := range defs {
			if defs[i].Name == old {
				defs[i].Name = name
				defs[i].Aliases = nextAliases
				changed = true
			}
		}
		if changed {
			b, _ := json.Marshal(defs)
			if _, err = tx.ExecContext(ctx, `UPDATE ai_tag_organization_jobs SET vocabulary_json=?,revision=revision+1 WHERE id=?`, string(b), v.id); err != nil {
				return err
			}
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO topic_label_renames(topic_id,old_name,new_name,created_at) VALUES(?,?,?,?)`, id, old, name, nowUTC()); err != nil {
		return err
	}
	return tx.Commit()
}
