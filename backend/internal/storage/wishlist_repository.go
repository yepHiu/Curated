package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/library/moviecode"
	"github.com/google/uuid"
)

var ErrWishlistConflict = errors.New("WISHLIST_VERSION_CONFLICT")

// AddWishlist 原子保存番号与待处理工作，重试不会重新激活或重刮已有条目。
func (s *SQLiteStore) AddWishlist(ctx context.Context, raw string) (string, bool, error) {
	code, key, err := moviecode.WishlistIdentity(raw)
	if err != nil {
		return "", false, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback()
	id := uuid.NewString()
	now := nowUTC()
	result, err := tx.ExecContext(ctx, `INSERT INTO wishlist_items(id,code,identity_key,created_at,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(identity_key) DO NOTHING`, id, code, key, now, now)
	if err != nil {
		return "", false, err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		err = tx.QueryRowContext(ctx, `SELECT id FROM wishlist_items WHERE identity_key=?`, key).Scan(&id)
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO wishlist_jobs(item_id,generation) VALUES(?,1)`, id)
	}
	if err != nil {
		return "", false, err
	}
	return id, count == 1, tx.Commit()
}

const wishlistSelect = `SELECT w.id,w.code,w.metadata_json,w.note,w.completed,w.state,w.error,w.version,w.generation,w.created_at,w.updated_at FROM wishlist_items w`

// scanWishlist 解码共享条目投影，空集合始终返回数组。
func scanWishlist(row interface{ Scan(...any) error }) (contracts.WishlistItemDTO, error) {
	item := contracts.WishlistItemDTO{MovieIDs: []string{}, Assets: []contracts.WishlistAssetDTO{}}
	var metadata string
	err := row.Scan(&item.ID, &item.Code, &metadata, &item.Note, &item.Completed, &item.EnrichmentState, &item.Error, &item.Version, &item.Generation, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	err = json.Unmarshal([]byte(metadata), &item.Metadata)
	if item.Metadata.Actors == nil {
		item.Metadata.Actors = []string{}
	}
	if item.Metadata.Tags == nil {
		item.Metadata.Tags = []string{}
	}
	item.Status = "pending"
	if item.Completed {
		item.Status = "completed"
	}
	return item, err
}

// GetWishlist 返回详情与当前有效的影片/图片关联。
func (s *SQLiteStore) GetWishlist(ctx context.Context, id string) (contracts.WishlistItemDTO, error) {
	item, err := scanWishlist(s.db.QueryRowContext(ctx, wishlistSelect+` WHERE w.id=?`, id))
	if err != nil {
		return item, err
	}
	err = s.hydrateWishlist(ctx, &item)
	return item, err
}

// hydrateWishlist 在主查询关闭后读取关联，避免单连接 SQLite 嵌套查询死锁。
func (s *SQLiteStore) hydrateWishlist(ctx context.Context, item *contracts.WishlistItemDTO) error {
	rows, err := s.db.QueryContext(ctx, `SELECT l.movie_id FROM wishlist_movie_links l JOIN movies m ON m.id=l.movie_id WHERE l.item_id=? AND l.excluded=0 AND COALESCE(m.trashed_at,'')=''`, item.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		item.MovieIDs = append(item.MovieIDs, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if len(item.MovieIDs) > 0 {
		item.Status = "in_library"
	}
	rows, err = s.db.QueryContext(ctx, `SELECT id,role,thumbnail_path FROM wishlist_assets WHERE item_id=? AND generation=? ORDER BY CASE role WHEN 'thumb' THEN 0 WHEN 'cover' THEN 1 ELSE 2 END,position`, item.ID, item.Generation)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var a contracts.WishlistAssetDTO
		var thumb string
		if err = rows.Scan(&a.ID, &a.Role, &thumb); err != nil {
			return err
		}
		a.URL = "/api/wishlist/items/" + item.ID + "/assets/" + a.ID
		a.ThumbnailURL = a.URL
		if thumb != "" {
			a.ThumbnailURL += "?thumbnail=1"
		}
		item.Assets = append(item.Assets, a)
	}
	return rows.Err()
}

// ListWishlist 使用绑定筛选的游标分页，统计不依赖前端遍历全部数据。
func (s *SQLiteStore) ListWishlist(ctx context.Context, status, q, cursor string, limit int) (contracts.WishlistPageDTO, error) {
	out := contracts.WishlistPageDTO{Items: []contracts.WishlistItemDTO{}}
	if limit < 1 || limit > 100 {
		limit = 60
	}
	linked := `EXISTS(SELECT 1 FROM wishlist_movie_links l JOIN movies m ON m.id=l.movie_id WHERE l.item_id=w.id AND l.excluded=0 AND COALESCE(m.trashed_at,'')='')`
	where := "1=1"
	switch status {
	case "pending":
		where = "w.completed=0 AND NOT " + linked
	case "completed":
		where = "w.completed=1 AND NOT " + linked
	case "in_library":
		where = linked
	case "all":
	default:
		return out, errors.New("invalid wishlist status")
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wishlist_items w WHERE w.completed=0 AND NOT `+linked).Scan(&out.PendingCount); err != nil {
		return out, err
	}
	args := []any{}
	if q != "" {
		where += ` AND (w.code LIKE ? OR w.metadata_json LIKE ?)`
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wishlist_items w WHERE `+where, args...).Scan(&out.Total); err != nil {
		return out, err
	}
	if cursor != "" {
		b, err := base64.RawURLEncoding.DecodeString(cursor)
		var c []string
		if err != nil || json.Unmarshal(b, &c) != nil || len(c) != 4 || c[0] != status || c[1] != q {
			return out, errors.New("invalid wishlist cursor")
		}
		where += ` AND (w.created_at < ? OR (w.created_at=? AND w.id<?))`
		args = append(args, c[2], c[2], c[3])
	}
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, wishlistSelect+` WHERE `+where+` ORDER BY w.created_at DESC,w.id DESC LIMIT ?`, args...)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		item, e := scanWishlist(rows)
		if e != nil {
			rows.Close()
			return out, e
		}
		out.Items = append(out.Items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	if len(out.Items) > limit {
		out.Items = out.Items[:limit]
		last := out.Items[limit-1]
		b, _ := json.Marshal([]string{status, q, last.CreatedAt, last.ID})
		out.NextCursor = base64.RawURLEncoding.EncodeToString(b)
	}
	for i := range out.Items {
		if err = s.hydrateWishlist(ctx, &out.Items[i]); err != nil {
			return out, err
		}
	}
	return out, nil
}

// PatchWishlist 原子应用用户修订，番号变化使旧任务结果失效。
func (s *SQLiteStore) PatchWishlist(ctx context.Context, id string, p contracts.WishlistPatch) error {
	if p.Note != nil && len([]rune(*p.Note)) > 10000 {
		return errors.New("note too long")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version, generation int
	var oldCode string
	if err = tx.QueryRowContext(ctx, `SELECT version,generation,code FROM wishlist_items WHERE id=?`, id).Scan(&version, &generation, &oldCode); err != nil {
		return err
	}
	if version != p.Version {
		return ErrWishlistConflict
	}
	if p.Code != nil && *p.Code != oldCode {
		code, key, e := moviecode.WishlistIdentity(*p.Code)
		if e != nil {
			return e
		}
		generation++
		_, err = tx.ExecContext(ctx, `UPDATE wishlist_items SET code=?,identity_key=?,generation=?,metadata_json='{}',state='queued',error='' WHERE id=?`, code, key, generation, id)
		if isSQLiteUniqueConstraint(err) {
			return errors.New("WISHLIST_CODE_CONFLICT")
		}
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM wishlist_movie_links WHERE item_id=?`, id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO wishlist_jobs(item_id,generation) VALUES(?,?) ON CONFLICT(item_id) DO UPDATE SET generation=excluded.generation,attempts=0,next_at=0,lease_until=0`, id, generation); err != nil {
			return err
		}
	}
	if p.Note != nil {
		if _, err = tx.ExecContext(ctx, `UPDATE wishlist_items SET note=? WHERE id=?`, *p.Note, id); err != nil {
			return err
		}
	}
	if p.Completed != nil {
		if _, err = tx.ExecContext(ctx, `UPDATE wishlist_items SET completed=? WHERE id=?`, *p.Completed, id); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE wishlist_items SET version=version+1,updated_at=? WHERE id=?`, nowUTC(), id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteWishlist 删除条目及其关系，保留文件供独立资产回收处理。
func (s *SQLiteStore) DeleteWishlist(ctx context.Context, id string) error {
	r, err := s.db.ExecContext(ctx, `DELETE FROM wishlist_items WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RefreshWishlist 仅为非在途条目入队，失败重试不破坏已保存图片。
func (s *SQLiteStore) RefreshWishlist(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var generation int
	if err = tx.QueryRowContext(ctx, `SELECT generation FROM wishlist_items WHERE id=?`, id).Scan(&generation); err != nil {
		return err
	}
	r, err := tx.ExecContext(ctx, `INSERT INTO wishlist_jobs(item_id,generation) VALUES(?,?) ON CONFLICT(item_id) DO NOTHING`, id, generation)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n > 0 {
		if _, err = tx.ExecContext(ctx, `UPDATE wishlist_items SET state='queued',error='' WHERE id=?`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ReconcileWishlist 根据统一规范键补关联；只读影片身份，不因离线盘删关联。
func (s *SQLiteStore) ReconcileWishlist(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id,code FROM movies WHERE COALESCE(trashed_at,'')=''`)
	if err != nil {
		return err
	}
	matches := map[string][]string{}
	for rows.Next() {
		var id, code string
		if err = rows.Scan(&id, &code); err != nil {
			rows.Close()
			return err
		}
		_, key, e := moviecode.WishlistIdentity(code)
		if e == nil {
			matches[key] = append(matches[key], id)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for key, ids := range matches {
		if len(ids) != 1 {
			continue
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO wishlist_movie_links(item_id,movie_id) SELECT id,? FROM wishlist_items WHERE identity_key=? ON CONFLICT(item_id,movie_id) DO NOTHING`, ids[0], key); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetWishlistLink 确认或排除单个匹配对，避免自动对账撤销用户选择。
func (s *SQLiteStore) SetWishlistLink(ctx context.Context, id, movieID string, excluded bool) error {
	var found int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM movies WHERE id=? AND COALESCE(trashed_at,'')=''`, movieID).Scan(&found); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO wishlist_movie_links(item_id,movie_id,excluded) VALUES(?,?,?) ON CONFLICT(item_id,movie_id) DO UPDATE SET excluded=excluded.excluded`, id, movieID, excluded)
	return err
}

// wishlistTokenHash 将高熵凭证转换为不可恢复的数据库索引。
func wishlistTokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// CreateWishlistToken 创建仅提交番号的凭证，明文只出现在本次回执。
func (s *SQLiteStore) CreateWishlistToken(ctx context.Context, name, origin string) (contracts.WishlistTokenDTO, error) {
	item := contracts.WishlistTokenDTO{ID: uuid.NewString(), Name: strings.TrimSpace(name), CreatedAt: nowUTC(), Token: uuid.NewString() + uuid.NewString(), Origin: origin}
	if item.Name == "" || len([]rune(item.Name)) > 80 {
		return item, errors.New("invalid token name")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO wishlist_tokens(id,name,hash,created_at,origin) VALUES(?,?,?,?,?)`, item.ID, item.Name, wishlistTokenHash(item.Token), item.CreatedAt, origin)
	return item, err
}

// ValidateWishlistToken 同时校验凭证及可选的扩展来源绑定。
func (s *SQLiteStore) ValidateWishlistToken(ctx context.Context, token, origin string) bool {
	if len(token) != 72 {
		return false
	}
	var expected string
	err := s.db.QueryRowContext(ctx, `SELECT origin FROM wishlist_tokens WHERE hash=?`, wishlistTokenHash(token)).Scan(&expected)
	return err == nil && (origin == "" || expected == "" || expected == origin)
}

// ListWishlistTokens 返回不含明文或哈希的凭证描述。
func (s *SQLiteStore) ListWishlistTokens(ctx context.Context) ([]contracts.WishlistTokenDTO, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,created_at,origin FROM wishlist_tokens ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []contracts.WishlistTokenDTO{}
	for rows.Next() {
		var item contracts.WishlistTokenDTO
		if err = rows.Scan(&item.ID, &item.Name, &item.CreatedAt, &item.Origin); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// DeleteWishlistToken 撤销凭证，后续提交立即失效。
func (s *SQLiteStore) DeleteWishlistToken(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM wishlist_tokens WHERE id=?`, id)
	return err
}

// ClaimWishlistJob 原子领取过期或未开始工作，进程退出后由租约恢复。
func (s *SQLiteStore) ClaimWishlistJob(ctx context.Context) (contracts.WishlistItemDTO, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return contracts.WishlistItemDTO{}, 0, err
	}
	defer tx.Rollback()
	var id string
	var attempt int
	now := time.Now().Unix()
	err = tx.QueryRowContext(ctx, `SELECT item_id,attempts FROM wishlist_jobs WHERE next_at<=? AND lease_until<=? ORDER BY next_at,item_id LIMIT 1`, now, now).Scan(&id, &attempt)
	if err != nil {
		return contracts.WishlistItemDTO{}, 0, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE wishlist_jobs SET lease_until=?,attempts=attempts+1 WHERE item_id=?`, now+600, id); err != nil {
		return contracts.WishlistItemDTO{}, 0, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE wishlist_items SET state='running',error='' WHERE id=?`, id); err != nil {
		return contracts.WishlistItemDTO{}, 0, err
	}
	if err = tx.Commit(); err != nil {
		return contracts.WishlistItemDTO{}, 0, err
	}
	item, err := s.GetWishlist(ctx, id)
	return item, attempt + 1, err
}

// SaveWishlistMetadata 仅保存当前版本资料，用户备注保持独立。
func (s *SQLiteStore) SaveWishlistMetadata(ctx context.Context, id string, generation int, m contracts.WishlistMetadata) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	r, err := s.db.ExecContext(ctx, `UPDATE wishlist_items SET metadata_json=?,updated_at=? WHERE id=? AND generation=?`, string(b), nowUTC(), id, generation)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// FinishWishlistJob 完成、有限重试或记录需要人工处理的状态。
func (s *SQLiteStore) FinishWishlistJob(ctx context.Context, id string, generation, attempt int, state, message string, retry bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if retry && attempt < 3 {
		_, err = tx.ExecContext(ctx, `UPDATE wishlist_jobs SET lease_until=0,next_at=? WHERE item_id=? AND generation=?`, time.Now().Add(time.Duration(attempt)*time.Minute).Unix(), id, generation)
		state = "queued"
	} else {
		_, err = tx.ExecContext(ctx, `DELETE FROM wishlist_jobs WHERE item_id=? AND generation=?`, id, generation)
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE wishlist_items SET state=?,error=?,updated_at=? WHERE id=? AND generation=?`, state, message, nowUTC(), id, generation)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// WishlistAssetFile 描述内部资产记录，仅服务端使用本地相对路径。
type WishlistAssetFile struct {
	ID, ItemID, Role, SourceURL, Path, ThumbnailPath, SHA256 string
	Generation, Position                                     int
}

// SaveWishlistAsset 校验 generation 后登记不可变文件，迟到下载不能挂到新条目版本。
func (s *SQLiteStore) SaveWishlistAsset(ctx context.Context, a WishlistAssetFile) error {
	r, err := s.db.ExecContext(ctx, `INSERT INTO wishlist_assets(id,item_id,generation,role,position,source_url,path,thumbnail_path,sha256) SELECT ?,id,?,?,?,?,?,?,? FROM wishlist_items WHERE id=? AND generation=? ON CONFLICT(item_id,generation,role,position) DO UPDATE SET source_url=excluded.source_url,path=excluded.path,thumbnail_path=excluded.thumbnail_path,sha256=excluded.sha256`, a.ID, a.Generation, a.Role, a.Position, a.SourceURL, a.Path, a.ThumbnailPath, a.SHA256, a.ItemID, a.Generation)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// WishlistAssetFiles 返回当前版本的已提交图片供读取和入库复制。
func (s *SQLiteStore) WishlistAssetFiles(ctx context.Context, id string) ([]WishlistAssetFile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,a.item_id,a.generation,a.role,a.position,a.source_url,a.path,a.thumbnail_path,a.sha256 FROM wishlist_assets a JOIN wishlist_items w ON w.id=a.item_id AND w.generation=a.generation WHERE a.item_id=? ORDER BY a.role,a.position`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WishlistAssetFile{}
	for rows.Next() {
		var a WishlistAssetFile
		if err = rows.Scan(&a.ID, &a.ItemID, &a.Generation, &a.Role, &a.Position, &a.SourceURL, &a.Path, &a.ThumbnailPath, &a.SHA256); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// WishlistAssetRoot 返回数据库旁的持久根，独立于可清理缓存。
func (s *SQLiteStore) WishlistAssetRoot() (string, error) {
	var seq int
	var name, path string
	err := s.db.QueryRow(`PRAGMA database_list`).Scan(&seq, &name, &path)
	if err != nil {
		return "", fmt.Errorf("wishlist database path: %w", err)
	}
	return filepath.Join(filepath.Dir(path), "assets", "wishlist"), nil
}
