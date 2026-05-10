package storagehealth

// Binding stores the expected backing volume identity for one configured library path.
type Binding struct {
	LibraryPathID      string
	RootPath           string
	VolumeID           string
	VolumeLabel        string
	FileSystem         string
	DriveType          string
	IdentityConfidence string
}

// ProbeResult is the platform-neutral snapshot returned by a path storage probe.
type ProbeResult struct {
	RootPath           string
	RootAvailable      bool
	PathExists         bool
	PathIsDir          bool
	PathReadable       bool
	PermissionDenied   bool
	VolumeID           string
	VolumeLabel        string
	FileSystem         string
	DriveType          string
	IdentityConfidence string
	ErrorMessage       string
}
