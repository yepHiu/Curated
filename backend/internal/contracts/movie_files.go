package contracts

import (
	"context"
	"strings"
)

// MovieFileDTO 是同一作品的一个视频分片，ID 不随分片显示顺序变化。
type MovieFileDTO struct {
	ID        string `json:"id"`
	PartIndex int    `json:"partIndex"`
	FileName  string `json:"fileName"`
	Location  string `json:"location"`
}

type movieFileContextKey struct{}

// WithMovieFileSelection 将可选视频文件选择传给播放、进度及媒体处理服务。
func WithMovieFileSelection(ctx context.Context, fileID string) context.Context {
	return context.WithValue(ctx, movieFileContextKey{}, strings.TrimSpace(fileID))
}

// MovieFileSelection 读取当前操作的分片；空值保留旧客户端默认播放行为。
func MovieFileSelection(ctx context.Context) string {
	id, _ := ctx.Value(movieFileContextKey{}).(string)
	return id
}
