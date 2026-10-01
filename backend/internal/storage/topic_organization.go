package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"curated-backend/internal/contracts"
)

// TopicDefinition 是归纳后的词汇；仅通过 user 标签持久化。
type TopicDefinition struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
}

// TopicMovieInput 是不带路径和偏好信息的只读分析投影。
type TopicMovieInput struct {
	MovieID      string   `json:"movieId"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	MetadataTags []string `json:"metadataTags"`
	UserTags     []string `json:"userTags"`
	Revision     int64    `json:"-"`
	Fingerprint  string   `json:"-"`
}

// topicIdentifier 为名称生成稳定 ID；后续名称展示不依赖模型生成 ID。
func topicIdentifier(name string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(name))))
	return "topic-" + hex.EncodeToString(sum[:12])
}

// TopicMovieInput 读取原始资料和当前用户标签版本，避免使用展示覆盖冒充源资料。
func (s *SQLiteStore) TopicMovieInput(ctx context.Context, id string) (TopicMovieInput, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return TopicMovieInput{}, err
	}
	defer func() { _ = tx.Rollback() }()
	return topicMovieInputTx(ctx, tx, id)
}

// topicMovieInputTx 在同一事务中捕获资料与标签版本。
func topicMovieInputTx(ctx context.Context, tx *sql.Tx, id string) (TopicMovieInput, error) {
	v := TopicMovieInput{MovieID: id, MetadataTags: []string{}, UserTags: []string{}}
	err := tx.QueryRowContext(ctx, `SELECT m.title,m.summary,COALESCE(r.revision,0) FROM movies m
	LEFT JOIN movie_tag_revisions r ON r.movie_id=m.id WHERE m.id=? AND `+sqlMovieActiveClause, id).Scan(&v.Title, &v.Summary, &v.Revision)
	if err != nil {
		return v, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT t.name,t.type FROM tags t JOIN movie_tags mt ON mt.tag_id=t.id WHERE mt.movie_id=? ORDER BY t.name,t.type`, id)
	if err != nil {
		return v, err
	}
	for rows.Next() {
		var name, kind string
		if err = rows.Scan(&name, &kind); err != nil {
			_ = rows.Close()
			return v, err
		}
		if kind == "user" {
			v.UserTags = append(v.UserTags, name)
		} else if kind == "nfo" {
			v.MetadataTags = append(v.MetadataTags, name)
		}
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return v, err
	}
	_ = rows.Close()
	data, _ := json.Marshal([]any{v.Title, v.Summary, v.MetadataTags})
	sum := sha256.Sum256(data)
	v.Fingerprint = hex.EncodeToString(sum[:])
	return v, nil
}

// SaveTopicVocabulary 将受验证的规范词汇映射到用户标签，永不修改 NFO 行。
func (s *SQLiteStore) SaveTopicVocabulary(ctx context.Context, defs []TopicDefinition) error {
	return s.saveTopicVocabulary(ctx, "", defs)
}

// SaveTopicVocabularyForJob checks cancellation in the same transaction as vocabulary writes.
func (s *SQLiteStore) SaveTopicVocabularyForJob(ctx context.Context, jobID string, defs []TopicDefinition) error {
	return s.saveTopicVocabulary(ctx, jobID, defs)
}

// saveTopicVocabulary is the only writer of user-only topic mappings.
func (s *SQLiteStore) saveTopicVocabulary(ctx context.Context, jobID string, defs []TopicDefinition) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if jobID != "" {
		var status string
		if err = tx.QueryRowContext(ctx, `SELECT status FROM ai_tag_organization_jobs WHERE id=?`, jobID).Scan(&status); err != nil {
			return err
		}
		if status != "running" {
			return fmt.Errorf("organization is not running")
		}
	}
	for _, d := range defs {
		names, err := NormalizeUserTagsForPatch([]string{d.Name})
		if err != nil || len(names) != 1 {
			return ErrInvalidUserTags
		}
		name := names[0]
		id := topicIdentifier(name)
		// 名称大小写规范化后，优先复用既有题材和对应用户标签。
		var existing string
		err = tx.QueryRowContext(ctx, `SELECT p.id FROM library_topics p JOIN tags t ON t.id=p.tag_id AND t.type='user' WHERE lower(t.name)=lower(?)`, name).Scan(&existing)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		tagID, err := ensureUserTag(ctx, tx, name)
		if err != nil {
			return err
		}
		aliases, _ := json.Marshal(d.Aliases)
		if _, err = tx.ExecContext(ctx, `INSERT INTO library_topics(id,tag_id,description,aliases_json) VALUES(?,?,?,?)`, id, tagID, d.Description, string(aliases)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// TopicVocabulary 返回规范词汇以便各批次复用。
func (s *SQLiteStore) TopicVocabulary(ctx context.Context) ([]TopicDefinition, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.name,p.description,p.aliases_json FROM library_topics p JOIN tags t ON t.id=p.tag_id AND t.type='user' ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []TopicDefinition{}
	for rows.Next() {
		var d TopicDefinition
		var raw string
		if err = rows.Scan(&d.Name, &d.Description, &raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &d.Aliases); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// recordManualTopicDecisionsTx 记录真实人工增删，保留对 AI 后续整理的约束。
func recordManualTopicDecisionsTx(ctx context.Context, tx *sql.Tx, movieID string, names []string) error {
	if err := recordManualUserTagDecisionsTx(ctx, tx, movieID, names); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT p.id,t.name,EXISTS(SELECT 1 FROM movie_tags mt WHERE mt.movie_id=? AND mt.tag_id=t.id)
	FROM library_topics p JOIN tags t ON t.id=p.tag_id AND t.type='user'`, movieID)
	if err != nil {
		return err
	}
	set := map[string]bool{}
	for _, name := range names {
		set[name] = true
	}
	type decision struct{ id, value string }
	changes := []decision{}
	for rows.Next() {
		var id, name string
		var present bool
		if err = rows.Scan(&id, &name, &present); err != nil {
			_ = rows.Close()
			return err
		}
		if present != set[name] {
			value := "exclude"
			if set[name] {
				value = "keep"
			}
			changes = append(changes, decision{id, value})
		}
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()
	for _, d := range changes {
		if _, err = tx.ExecContext(ctx, `INSERT INTO movie_topic_decisions(movie_id,topic_id,decision) VALUES(?,?,?) ON CONFLICT(movie_id,topic_id) DO UPDATE SET decision=excluded.decision`, movieID, d.id, d.value); err != nil {
			return err
		}
	}
	return nil
}

// bumpUserTagRevisionTx 在事务中更新用户标签修订。
func bumpUserTagRevisionTx(ctx context.Context, tx *sql.Tx, id string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO movie_tag_revisions(movie_id,revision) VALUES(?,1) ON CONFLICT(movie_id) DO UPDATE SET revision=revision+1`, id)
	return err
}

// ApplyMovieTopics 只添加明确题材与替换等价别名，不按模型遗漏删除标签。
// job scope、资料指纹和人工版本在同一事务中核对，检查点与写入原子提交。
func (s *SQLiteStore) ApplyMovieTopics(ctx context.Context, jobID string, input TopicMovieInput, names []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var state, status string
	if err = tx.QueryRowContext(ctx, `SELECT i.status,j.status FROM ai_tag_organization_items i JOIN ai_tag_organization_jobs j ON j.id=i.job_id WHERE i.job_id=? AND i.movie_id=?`, jobID, input.MovieID).Scan(&state, &status); err != nil {
		return err
	}
	if state == "succeeded" {
		return nil
	}
	if status != "running" {
		return fmt.Errorf("organization is not running")
	}
	current, err := topicMovieInputTx(ctx, tx, input.MovieID)
	if err != nil {
		return err
	}
	if current.Revision != input.Revision || current.Fingerprint != input.Fingerprint {
		return ErrAIWriteConflict
	}
	set := map[string]bool{}
	for _, n := range current.UserTags {
		set[n] = true
	}
	for _, name := range names {
		var id, canonical, aliasesRaw string
		if err = tx.QueryRowContext(ctx, `SELECT p.id,t.name,p.aliases_json FROM library_topics p JOIN tags t ON t.id=p.tag_id AND t.type='user' WHERE lower(t.name)=lower(?)`, name).Scan(&id, &canonical, &aliasesRaw); err != nil {
			return fmt.Errorf("unknown user topic: %w", err)
		}
		var decision string
		err = tx.QueryRowContext(ctx, `SELECT decision FROM movie_topic_decisions WHERE movie_id=? AND topic_id=?`, input.MovieID, id).Scan(&decision)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var manualExclude int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM movie_user_tag_decisions WHERE movie_id=? AND name=? AND decision='exclude'`, input.MovieID, canonical).Scan(&manualExclude); err != nil {
			return err
		}
		if decision == "exclude" || manualExclude > 0 {
			continue
		}
		set[canonical] = true
		// 别名只转换本任务当前影片的 user 关联，不改全局标签行。
		var aliases []string
		if err = json.Unmarshal([]byte(aliasesRaw), &aliases); err != nil {
			return err
		}
		for _, alias := range aliases {
			if alias != canonical {
				var keep int
				if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM movie_topic_decisions d JOIN library_topics p ON p.id=d.topic_id JOIN tags t ON t.id=p.tag_id WHERE d.movie_id=? AND d.decision='keep' AND t.name=?`, input.MovieID, alias).Scan(&keep); err != nil {
					return err
				}
				var manualKeep int
				if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM movie_user_tag_decisions WHERE movie_id=? AND name=? AND decision='keep'`, input.MovieID, alias).Scan(&manualKeep); err != nil {
					return err
				}
				if keep == 0 && manualKeep == 0 {
					delete(set, alias)
				}
			}
		}
	}
	after := []string{}
	for n := range set {
		after = append(after, n)
	}
	sort.Strings(after)
	if _, err = NormalizeUserTagsForPatch(after); err != nil {
		return err
	}
	beforeJSON, _ := json.Marshal(current.UserTags)
	afterJSON, _ := json.Marshal(after)
	if string(beforeJSON) != string(afterJSON) {
		if err = replaceMovieUserTagsTx(ctx, tx, input.MovieID, after); err != nil {
			return err
		}
		if err = bumpUserTagRevisionTx(ctx, tx, input.MovieID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO ai_tag_organization_changes(job_id,movie_id,before_json,after_json,after_revision) VALUES(?,?,?,?,?)`, jobID, input.MovieID, string(beforeJSON), string(afterJSON), current.Revision+1); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE ai_tag_organization_items SET status='succeeded',reason='' WHERE job_id=? AND movie_id=?`, jobID, input.MovieID); err != nil {
		return err
	}
	return tx.Commit()
}

// UndoTopicOrganization 恢复未被后续编辑覆盖的用户标签，永不触碰 NFO。
func (s *SQLiteStore) UndoTopicOrganization(ctx context.Context, jobID string) (contracts.TagOrganizationUndoDTO, error) {
	out := contracts.TagOrganizationUndoDTO{}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback() }()
	var status string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM ai_tag_organization_jobs WHERE id=?`, jobID).Scan(&status); err != nil {
		return out, err
	}
	if status == "running" || status == "queued" {
		return out, fmt.Errorf("cancel organization before undo")
	}
	rows, err := tx.QueryContext(ctx, `SELECT movie_id,before_json,after_revision FROM ai_tag_organization_changes WHERE job_id=? AND undone=0`, jobID)
	if err != nil {
		return out, err
	}
	type change struct {
		id, before string
		rev        int64
	}
	changes := []change{}
	for rows.Next() {
		var c change
		if err = rows.Scan(&c.id, &c.before, &c.rev); err != nil {
			_ = rows.Close()
			return out, err
		}
		changes = append(changes, c)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return out, err
	}
	_ = rows.Close()
	for _, c := range changes {
		var rev int64
		err = tx.QueryRowContext(ctx, `SELECT revision FROM movie_tag_revisions WHERE movie_id=?`, c.id).Scan(&rev)
		if errors.Is(err, sql.ErrNoRows) || err == nil && rev != c.rev {
			out.Conflicts++
			continue
		}
		if err != nil {
			return out, err
		}
		var before []string
		if err = json.Unmarshal([]byte(c.before), &before); err != nil {
			return out, err
		}
		if err = replaceMovieUserTagsTx(ctx, tx, c.id, before); err != nil {
			return out, err
		}
		if err = bumpUserTagRevisionTx(ctx, tx, c.id); err != nil {
			return out, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE ai_tag_organization_changes SET undone=1 WHERE job_id=? AND movie_id=?`, jobID, c.id); err != nil {
			return out, err
		}
		out.Restored++
	}
	return out, tx.Commit()
}

// sameUserTagSet 避免仅顺序变化或无变化的人工提交破坏撤销版本。
func sameUserTagSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := map[string]bool{}
	for _, n := range a {
		set[n] = true
	}
	for _, n := range b {
		if !set[n] {
			return false
		}
	}
	return true
}

// recordManualUserTagDecisionsTx preserves later corrections, including aliases without their own topic.
// Existing tags before the first vocabulary remain eligible for initial normalization.
func recordManualUserTagDecisionsTx(ctx context.Context, tx *sql.Tx, movieID string, names []string) error {
	var topics int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM library_topics`).Scan(&topics); err != nil {
		return err
	}
	if topics == 0 {
		return nil
	}
	current, err := topicMovieInputTx(ctx, tx, movieID)
	if err != nil {
		return err
	}
	before, after := map[string]bool{}, map[string]bool{}
	for _, name := range current.UserTags {
		before[name] = true
	}
	for _, name := range names {
		after[name] = true
	}
	decisions := map[string]string{}
	for name := range before {
		if !after[name] {
			decisions[name] = "exclude"
		}
	}
	for name := range after {
		if !before[name] {
			decisions[name] = "keep"
		}
	}
	for name, decision := range decisions {
		if _, err = tx.ExecContext(ctx, `INSERT INTO movie_user_tag_decisions(movie_id,name,decision) VALUES(?,?,?) ON CONFLICT(movie_id,name) DO UPDATE SET decision=excluded.decision`, movieID, name, decision); err != nil {
			return err
		}
	}
	return nil
}
