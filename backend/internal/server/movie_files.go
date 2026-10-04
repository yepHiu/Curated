package server

import (
	"net/http"
	"strings"

	"curated-backend/internal/contracts"
)

// withMovieFileSelection 将兼容的 fileId 查询参数投影到领域操作上下文。
func withMovieFileSelection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 文件归属在 storage 中按 movieId 校验，客户端不能用另一作品的文件播放。
		fileID := strings.TrimSpace(r.URL.Query().Get("fileId"))
		if len(fileID) > 128 {
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid fileId")
			return
		}
		next.ServeHTTP(w, r.WithContext(contracts.WithMovieFileSelection(r.Context(), fileID)))
	})
}
