package storage

import (
	"context"
	"errors"
	"strings"

	"curated-backend/internal/contracts"
	"curated-backend/internal/library/moviecode"
)

// WishlistStatus 通过同一去重键批量读取成员状态，包含已完成、已入库条目，不创建任务。
func (s *SQLiteStore) WishlistStatus(ctx context.Context, codes []string) (contracts.WishlistStatusDTO, error) {
	out := contracts.WishlistStatusDTO{StatusMap: make(map[string]contracts.WishlistMembershipDTO)}
	if len(codes) == 0 || len(codes) > 100 {
		return out, errors.New("WISHLIST_INVALID_CODES")
	}
	keys := make(map[string][]string)
	for _, code := range codes {
		_, key, err := moviecode.WishlistIdentity(code)
		if err != nil {
			return out, err
		}
		keys[key] = append(keys[key], code)
		out.StatusMap[code] = contracts.WishlistMembershipDTO{}
	}
	args := make([]any, 0, len(keys))
	for key := range keys {
		args = append(args, key)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")
	rows, err := s.db.QueryContext(ctx, `SELECT identity_key FROM wishlist_items WHERE identity_key IN (`+placeholders+`)`, args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return out, err
		}
		for _, code := range keys[key] {
			out.StatusMap[code] = contracts.WishlistMembershipDTO{Added: true}
		}
	}
	return out, rows.Err()
}
