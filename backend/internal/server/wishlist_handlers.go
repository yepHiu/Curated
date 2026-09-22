package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"curated-backend/internal/contracts"
)

const wishlistIntakePath = "/api/integrations/wishlist/items"

var wishlistRates = struct {
	sync.Mutex
	entries map[string]struct {
		at    time.Time
		count int
	}
}{entries: make(map[string]struct {
	at    time.Time
	count int
})}

// wishlistExtensionOrigin 只识别扩展 origin，不授权任何普通业务 API。
func wishlistExtensionOrigin(origin string) bool {
	u, e := url.Parse(origin)
	return e == nil && (u.Scheme == "chrome-extension" || u.Scheme == "moz-extension") && u.Host != "" && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == ""
}

// decodeWishlistJSON 限制请求大小，拒绝未知字段与追加 JSON。
func decodeWishlistJSON(w http.ResponseWriter, r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(out); e != nil {
		return e
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return errors.New("one JSON object required")
	}
	return nil
}

// wishlistError 将存储错误转换为稳定 HTTP 错误。
func wishlistError(w http.ResponseWriter, e error) {
	status := http.StatusInternalServerError
	code := "COMMON_INTERNAL"
	message := "wishlist operation failed"
	if errors.Is(e, sql.ErrNoRows) {
		status = 404
		code = "COMMON_NOT_FOUND"
		message = "wishlist item not found"
	} else if strings.Contains(e.Error(), "CONFLICT") {
		status = 409
		code = e.Error()
		message = "wishlist changed; reload before retrying"
	} else if strings.Contains(e.Error(), "INVALID") || strings.HasPrefix(e.Error(), "invalid") || e.Error() == "note too long" {
		status = 400
		code = "WISHLIST_INVALID_INPUT"
		message = e.Error()
	}
	writeAppError(w, status, code, message)
}

// handleAddWishlist 接收番号和可选来源网页，在持久提交后返回，不等待刮削。
func (h *Handler) handleAddWishlist(w http.ResponseWriter, r *http.Request) {
	if !h.browserPluginEnabled() {
		writeAppError(w, http.StatusForbidden, "BROWSER_PLUGIN_DISABLED", "browser plugin integration is disabled")
		return
	}
	if h.store == nil {
		writeAppError(w, 503, "COMMON_UNAVAILABLE", "wishlist unavailable")
		return
	}
	client, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		client = r.RemoteAddr
	}
	wishlistRates.Lock()
	now := time.Now()
	for key, v := range wishlistRates.entries {
		if now.Sub(v.at) > time.Minute {
			delete(wishlistRates.entries, key)
		}
	}
	entry := wishlistRates.entries[client]
	if entry.at.IsZero() {
		entry.at = now
	}
	entry.count++
	wishlistRates.entries[client] = entry
	wishlistRates.Unlock()
	if entry.count > 60 {
		w.Header().Set("Retry-After", "60")
		writeAppError(w, 429, "COMMON_RATE_LIMITED", "too many submissions")
		return
	}
	var body struct {
		Code      string `json:"code"`
		SourceURL string `json:"sourceUrl"`
	}
	if e := decodeWishlistJSON(w, r, &body); e != nil {
		writeAppError(w, 400, "WISHLIST_INVALID_INPUT", "expected code and optional sourceUrl")
		return
	}
	id, created, e := h.store.AddWishlist(r.Context(), body.Code, body.SourceURL)
	if e != nil {
		wishlistError(w, e)
		return
	}
	if e = h.store.ReconcileWishlist(r.Context()); e != nil {
		wishlistError(w, e)
		return
	}
	item, e := h.store.GetWishlist(r.Context(), id)
	if e != nil {
		wishlistError(w, e)
		return
	}
	status := 200
	result := "existing"
	if created {
		status = 201
		result = "created"
	}
	if item.Status == "in_library" {
		result = "in_library"
	}
	writeJSON(w, status, map[string]string{"id": id, "result": result})
}

// handleWishlistList 返回游标分页和待入库计数。
func (h *Handler) handleWishlistList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	if status == "" {
		status = "pending"
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if len(q.Get("q")) > 512 {
		writeAppError(w, 400, "WISHLIST_INVALID_INPUT", "query too long")
		return
	}
	if e := h.store.ReconcileWishlist(r.Context()); e != nil {
		wishlistError(w, e)
		return
	}
	page, e := h.store.ListWishlist(r.Context(), status, q.Get("q"), q.Get("cursor"), limit)
	if e != nil {
		wishlistError(w, e)
		return
	}
	writeJSON(w, 200, page)
}

// handleWishlistItem 处理单条读取、用户修订与删除。
func (h *Handler) handleWishlistItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var e error
	switch r.Method {
	case "DELETE":
		e = h.store.DeleteWishlist(r.Context(), id)
	case "PATCH":
		var p contracts.WishlistPatch
		if e = decodeWishlistJSON(w, r, &p); e != nil {
			writeAppError(w, 400, "WISHLIST_INVALID_INPUT", "invalid patch")
			return
		}
		e = h.store.PatchWishlist(r.Context(), id, p)
	}
	if e != nil {
		wishlistError(w, e)
		return
	}
	if r.Method == "DELETE" {
		w.WriteHeader(204)
		return
	}
	item, e := h.store.GetWishlist(r.Context(), id)
	if e != nil {
		wishlistError(w, e)
		return
	}
	writeJSON(w, 200, item)
}

// handleWishlistRefresh 请求后台重试，重复调用合并到同一任务。
func (h *Handler) handleWishlistRefresh(w http.ResponseWriter, r *http.Request) {
	if e := h.store.RefreshWishlist(r.Context(), r.PathValue("id")); e != nil {
		wishlistError(w, e)
		return
	}
	w.WriteHeader(202)
}

// handleWishlistLink 显式确认或排除关联。
func (h *Handler) handleWishlistLink(w http.ResponseWriter, r *http.Request) {
	var p struct {
		MovieID  string `json:"movieId"`
		Excluded bool   `json:"excluded"`
	}
	if e := decodeWishlistJSON(w, r, &p); e != nil {
		writeAppError(w, 400, "WISHLIST_INVALID_INPUT", "invalid link")
		return
	}
	if e := h.store.SetWishlistLink(r.Context(), r.PathValue("id"), p.MovieID, p.Excluded); e != nil {
		wishlistError(w, e)
		return
	}
	w.WriteHeader(204)
}

// handleWishlistAsset 仅根据当前条目资产引用提供文件，拒绝跨根路径。
func (h *Handler) handleWishlistAsset(w http.ResponseWriter, r *http.Request) {
	files, e := h.store.WishlistAssetFiles(r.Context(), r.PathValue("id"))
	if e != nil {
		wishlistError(w, e)
		return
	}
	root, e := h.store.WishlistAssetRoot()
	if e != nil {
		wishlistError(w, e)
		return
	}
	for _, a := range files {
		if a.ID != r.PathValue("assetId") {
			continue
		}
		relative := a.Path
		if r.URL.Query().Get("thumbnail") == "1" && a.ThumbnailPath != "" {
			relative = a.ThumbnailPath
		}
		path := filepath.Join(root, filepath.FromSlash(relative))
		rel, e := filepath.Rel(root, path)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "private, max-age=3600")
		http.ServeFile(w, r, path)
		return
	}
	http.NotFound(w, r)
}

// registerWishlistRoutes 集中注册愿望单及有限权限的外部提交入口。
func (h *Handler) registerWishlistRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST "+wishlistIntakePath, h.handleAddWishlist)
	mux.HandleFunc("GET /api/wishlist/items", h.handleWishlistList)
	mux.HandleFunc("GET /api/wishlist/items/{id}", h.handleWishlistItem)
	mux.HandleFunc("PATCH /api/wishlist/items/{id}", h.handleWishlistItem)
	mux.HandleFunc("DELETE /api/wishlist/items/{id}", h.handleWishlistItem)
	mux.HandleFunc("POST /api/wishlist/items/{id}/refresh", h.handleWishlistRefresh)
	mux.HandleFunc("PUT /api/wishlist/items/{id}/library-links", h.handleWishlistLink)
	mux.HandleFunc("GET /api/wishlist/items/{id}/assets/{assetId}", h.handleWishlistAsset)
}

// browserPluginEnabled uses the live preference when a controller is available.
func (h *Handler) browserPluginEnabled() bool {
	if h.browserPluginCtl != nil {
		return h.browserPluginCtl.BrowserPluginEnabled()
	}
	return h.cfg.BrowserPluginEnabled
}
