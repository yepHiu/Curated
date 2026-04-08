//go:build !windows

package devmetrics

func NewCPUSampler() CPUSampler {
	return unsupportedCPUSampler{}
}
