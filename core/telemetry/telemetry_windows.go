//go:build windows

// Package telemetry provides Windows process discovery and resource measurement using gopsutil.
package telemetry

import (
	"strings"

	"github.com/sancheza/tabwrangler/core/types"
	"github.com/shirou/gopsutil/v4/process"
)

// OSCollector gathers browser process metrics on Windows.
type OSCollector struct{}

// NewOSCollector constructs a Windows process collector.
func NewOSCollector() *OSCollector {
	return &OSCollector{}
}

// CollectBrowserProcesses scans the Windows process table and returns metrics for Chrome and Firefox.
func (c *OSCollector) CollectBrowserProcesses() (map[types.BrowserType][]ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	result := make(map[types.BrowserType][]ProcessInfo)

	for _, p := range procs {
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}

		nameLower := strings.ToLower(name)
		if !strings.Contains(nameLower, "chrome") &&
			!strings.Contains(nameLower, "firefox") {
			continue
		}

		cmdline, _ := p.Cmdline()
		isRenderer, browser := IsRendererProcess(name, cmdline)
		if browser == "" {
			continue
		}

		memInfo, err := p.MemoryInfo()
		var rss uint64
		if err == nil && memInfo != nil {
			rss = memInfo.RSS
		}

		cpu, _ := p.CPUPercent()
		ppid, _ := p.Ppid()

		info := ProcessInfo{
			PID:        int(p.Pid),
			PPID:       int(ppid),
			Name:       name,
			Browser:    browser,
			CPUPercent: cpu,
			RAMBytes:   rss,
			IsRenderer: isRenderer,
		}

		result[browser] = append(result[browser], info)
	}

	return result, nil
}
