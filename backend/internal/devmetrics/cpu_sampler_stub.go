//go:build !windows

package devmetrics

// NewCPUSampler returns a no-op CPU sampler on non-Windows platforms.
func NewCPUSampler() CPUSampler {
	return unsupportedCPUSampler{}
}
