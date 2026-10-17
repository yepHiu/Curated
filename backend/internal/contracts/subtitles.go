package contracts

type SubtitleAudioTrack struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Codec    string `json:"codec"`
	Channels int    `json:"channels"`
}

// SubtitleSettings never returns the provider credential.
type SubtitleSettings struct {
	RecognitionModel string `json:"recognitionModel"`
	CanManage        bool   `json:"canManage"`
	Enabled          bool   `json:"enabled"`
	APIKeyConfigured bool   `json:"apiKeyConfigured"`
	TranslationModel string `json:"translationModel"`
	ConfiguredModel  string `json:"configuredModel"`
}
type SubtitleSettingsPatch struct {
	RecognitionModel *string `json:"recognitionModel,omitempty"`
	Enabled          *bool   `json:"enabled,omitempty"`
	APIKey           *string `json:"apiKey,omitempty"`
	TranslationModel *string `json:"translationModel,omitempty"`
}
type SubtitleRequest struct {
	RequestID        string `json:"requestId"`
	FileID           string `json:"fileId"`
	TranslationModel string `json:"translationModel"`
	AudioTrack       int    `json:"audioTrack"`
}
type SubtitleAsset struct {
	ID       string `json:"id"`
	FileID   string `json:"fileId"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Format   string `json:"format"`
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	Missing  bool   `json:"missing"`
}
type SubtitleJob struct {
	RecognitionModel string          `json:"recognitionModel"`
	ID               string          `json:"id"`
	MovieID          string          `json:"movieId"`
	FileID           string          `json:"fileId"`
	FileName         string          `json:"fileName"`
	State            string          `json:"state"`
	Stage            string          `json:"stage"`
	TranslationModel string          `json:"translationModel"`
	Completed        int             `json:"completed"`
	Total            int             `json:"total"`
	Error            string          `json:"error,omitempty"`
	ProviderError    string          `json:"providerError,omitempty"`
	SupersededBy     string          `json:"supersededBy,omitempty"`
	CreatedAt        string          `json:"createdAt"`
	UpdatedAt        string          `json:"updatedAt"`
	ExpiresAt        string          `json:"expiresAt,omitempty"`
	Assets           []SubtitleAsset `json:"assets"`
}
