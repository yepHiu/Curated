package contracts

// NativePlaybackSource is a source-specific decoder query. Token binds the
// response to the selected file/version; it grants no access privileges.
type NativePlaybackSource struct {
	Token           string  `json:"token"`
	Container       string  `json:"container"`
	VideoCodec      string  `json:"videoCodec"`
	VideoProfile    string  `json:"videoProfile"`
	VideoLevel      int     `json:"videoLevel"`
	PixelFormat     string  `json:"pixelFormat"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	FrameRate       float64 `json:"frameRate"`
	BitRate         int64   `json:"bitRate"`
	AudioCodec      string  `json:"audioCodec"`
	AudioProfile    string  `json:"audioProfile"`
	AudioChannels   int     `json:"audioChannels"`
	AudioSampleRate int     `json:"audioSampleRate"`
	VideoStreams    int     `json:"videoStreams"`
	AudioStreams    int     `json:"audioStreams"`
}

type NativePlaybackPlanDTO struct {
	FileID string               `json:"fileId"`
	Source NativePlaybackSource `json:"source"`
}

// NativePlaybackSelection is accepted only by the versioned native protocol.
type NativePlaybackSelection struct {
	Version        int    `json:"version"`
	SourceToken    string `json:"sourceToken"`
	DirectPlay     bool   `json:"directPlay"`
	ForceTranscode bool   `json:"forceTranscode,omitempty"`
}
