package contracts

// WishlistMetadata 是与本地文件无关的影片资料，字段来源由 provider 标识。
type WishlistMetadata struct {
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	Actors         []string `json:"actors"`
	Tags           []string `json:"tags"`
	Studio         string   `json:"studio"`
	ReleaseDate    string   `json:"releaseDate"`
	RuntimeMinutes int      `json:"runtimeMinutes"`
	Provider       string   `json:"provider"`
	Homepage       string   `json:"homepage"`
}

// WishlistAssetDTO 只公开受保护的资源地址，不返回本地路径。
type WishlistAssetDTO struct {
	ID           string `json:"id"`
	Role         string `json:"role"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

// WishlistItemDTO 将用户完成意愿、入库关联和资料处理状态分开。
type WishlistItemDTO struct {
	ID              string             `json:"id"`
	Code            string             `json:"code"`
	Metadata        WishlistMetadata   `json:"metadata"`
	Note            string             `json:"note"`
	Completed       bool               `json:"completed"`
	Status          string             `json:"status"`
	EnrichmentState string             `json:"enrichmentState"`
	Error           string             `json:"error"`
	Version         int                `json:"version"`
	Generation      int                `json:"-"`
	CreatedAt       string             `json:"createdAt"`
	UpdatedAt       string             `json:"updatedAt"`
	MovieIDs        []string           `json:"movieIds"`
	Assets          []WishlistAssetDTO `json:"assets"`
}

// WishlistPageDTO 返回分页条目与全局待入库数量。
type WishlistPageDTO struct {
	Items        []WishlistItemDTO `json:"items"`
	Total        int               `json:"total"`
	PendingCount int               `json:"pendingCount"`
	NextCursor   string            `json:"nextCursor,omitempty"`
}

// WishlistPatch 显式版本防止备注、番号和完成意愿被并发覆盖。
type WishlistPatch struct {
	Version   int     `json:"version"`
	Code      *string `json:"code,omitempty"`
	Note      *string `json:"note,omitempty"`
	Completed *bool   `json:"completed,omitempty"`
}

// WishlistTokenDTO 列表只返回凭证描述，创建时才携带一次性明文。
type WishlistTokenDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	Token     string `json:"token,omitempty"`
	Origin    string `json:"origin"`
}
