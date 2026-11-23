//go:build !darwin && !windows

// Package telemetry provides fallback process discovery for unsupported platforms.
package telemetry

import (
	"github.com/sancheza/tabwrangler/core/types"
)

// OSCollector is a stub process collector.
type OSCollector struct{}

// NewOSCollector constructs a stub process collector.
func NewOSCollector() *OSCollector {
	return &OSCollector{}
}

// CollectBrowserProcesses returns an empty map on unsupported operating systems.
func (c *OSCollector) CollectBrowserProcesses() (map[types.BrowserType][]ProcessInfo, error) {
	return make(map[types.BrowserType][]ProcessInfo), nil
}
