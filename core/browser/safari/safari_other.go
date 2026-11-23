//go:build !darwin

// Package safari provides stub implementation for platforms without AppleScript support.
package safari

import (
	"context"
	"errors"

	"github.com/sancheza/tabwrangler/core/types"
)

// Adapter provides Safari stubs for non-macOS systems.
type Adapter struct{}

// NewAdapter constructs a stub adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Browser returns types.BrowserSafari.
func (a *Adapter) Browser() types.BrowserType {
	return types.BrowserSafari
}

// IsAvailable returns false on non-macOS platforms.
func (a *Adapter) IsAvailable() bool {
	return false
}

// ListTabs returns nil on non-macOS platforms.
func (a *Adapter) ListTabs(ctx context.Context) ([]types.Tab, error) {
	return nil, nil
}

// FocusTab returns an error on non-macOS platforms.
func (a *Adapter) FocusTab(ctx context.Context, tab types.Tab) error {
	return errors.New("safari adapter is only available on macOS")
}

// DiscardTabs returns skipped on non-macOS platforms.
func (a *Adapter) DiscardTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	return 0, len(tabs), nil
}

// CloseTabs returns an error on non-macOS platforms.
func (a *Adapter) CloseTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	return 0, len(tabs), errors.New("safari adapter is only available on macOS")
}
