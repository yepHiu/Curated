package storagehealth

// NewDefaultProbe returns the best platform-specific storage probe for the current runtime.
func NewDefaultProbe() Probe {
	return defaultProbe{}
}
