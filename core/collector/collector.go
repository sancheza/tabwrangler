// Package collector orchestrates tab data gathering and lifecycle actions
// across multiple browser adapters.
package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sancheza/tabwrangler/core/types"
)

// BrowserAdapter defines the contract required to interact with a specific browser.
type BrowserAdapter interface {
	Browser() types.BrowserType
	IsAvailable() bool
	ListTabs(ctx context.Context) ([]types.Tab, error)
	FocusTab(ctx context.Context, tab types.Tab) error
	DiscardTabs(ctx context.Context, tabs []types.Tab) (discarded int, skipped int, err error)
	CloseTabs(ctx context.Context, tabs []types.Tab) (closed int, failed int, err error)
	ReopenTabs(ctx context.Context, tabs []types.Tab) (opened int, failed int, err error)
	WakeTabs(ctx context.Context, tabs []types.Tab) (woken int, failed int, err error)
}

// Collector aggregates tab data from all registered browser adapters.
type Collector struct {
	mu       sync.RWMutex
	adapters map[types.BrowserType]BrowserAdapter
}

// NewCollector constructs a new Collector instance.
func NewCollector() *Collector {
	return &Collector{
		adapters: make(map[types.BrowserType]BrowserAdapter),
	}
}

// RegisterAdapter adds a browser adapter to the collection pipeline.
func (c *Collector) RegisterAdapter(adapter BrowserAdapter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.adapters[adapter.Browser()] = adapter
}

// GetAdapter retrieves the adapter registered for a specific browser type.
func (c *Collector) GetAdapter(b types.BrowserType) (BrowserAdapter, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	a, ok := c.adapters[b]
	return a, ok
}

// RegisteredBrowsers returns a list of all browser types currently registered.
func (c *Collector) RegisteredBrowsers() []types.BrowserType {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var list []types.BrowserType
	for b := range c.adapters {
		list = append(list, b)
	}
	return list
}

// FetchAll queries all available browser adapters concurrently with a per-adapter timeout.
func (c *Collector) FetchAll(ctx context.Context, timeout time.Duration) ([]types.Tab, map[types.BrowserType]error) {
	c.mu.RLock()
	adapters := make(map[types.BrowserType]BrowserAdapter, len(c.adapters))
	for k, v := range c.adapters {
		adapters[k] = v
	}
	c.mu.RUnlock()

	var (
		wg         sync.WaitGroup
		resMu      sync.Mutex
		allTabs    []types.Tab
		adapterErr = make(map[types.BrowserType]error)
	)

	for bType, adapter := range adapters {
		if !adapter.IsAvailable() {
			resMu.Lock()
			if ext, ok := adapter.(interface{ ServerError() error }); ok && ext.ServerError() != nil {
				adapterErr[bType] = fmt.Errorf("bridge socket error: %w", ext.ServerError())
			} else {
				adapterErr[bType] = fmt.Errorf("bridge disconnected")
			}
			resMu.Unlock()
			continue
		}

		wg.Add(1)
		go func(bt types.BrowserType, a BrowserAdapter) {
			defer wg.Done()

			fetchCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			tabs, err := a.ListTabs(fetchCtx)
			resMu.Lock()
			defer resMu.Unlock()

			if err != nil {
				adapterErr[bt] = err
			} else {
				allTabs = append(allTabs, tabs...)
			}
		}(bType, adapter)
	}

	wg.Wait()
	return allTabs, adapterErr
}

// FocusTab delegates focus execution to the appropriate browser adapter.
func (c *Collector) FocusTab(ctx context.Context, tab types.Tab) error {
	adapter, ok := c.GetAdapter(tab.Browser)
	if !ok {
		return fmt.Errorf("no adapter registered for browser %s", tab.Browser)
	}
	return adapter.FocusTab(ctx, tab)
}

// DiscardTabs batches tabs by browser and executes discard operations.
func (c *Collector) DiscardTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	byBrowser := make(map[types.BrowserType][]types.Tab)
	for _, t := range tabs {
		byBrowser[t.Browser] = append(byBrowser[t.Browser], t)
	}

	totalDiscarded := 0
	totalSkipped := 0

	for bt, bTabs := range byBrowser {
		adapter, ok := c.GetAdapter(bt)
		if !ok {
			totalSkipped += len(bTabs)
			continue
		}
		discarded, skipped, err := adapter.DiscardTabs(ctx, bTabs)
		totalDiscarded += discarded
		totalSkipped += skipped
		if err != nil {
			return totalDiscarded, totalSkipped, err
		}
	}

	return totalDiscarded, totalSkipped, nil
}

// CloseTabs batches tabs by browser and executes close operations.
func (c *Collector) CloseTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	byBrowser := make(map[types.BrowserType][]types.Tab)
	for _, t := range tabs {
		byBrowser[t.Browser] = append(byBrowser[t.Browser], t)
	}

	totalClosed := 0
	totalFailed := 0

	for bt, bTabs := range byBrowser {
		adapter, ok := c.GetAdapter(bt)
		if !ok {
			totalFailed += len(bTabs)
			continue
		}
		closed, failed, err := adapter.CloseTabs(ctx, bTabs)
		totalClosed += closed
		totalFailed += failed
		if err != nil {
			return totalClosed, totalFailed, err
		}
	}

	return totalClosed, totalFailed, nil
}

// ReopenTabs batches tabs by browser and executes reopen/create operations.
func (c *Collector) ReopenTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	byBrowser := make(map[types.BrowserType][]types.Tab)
	for _, t := range tabs {
		byBrowser[t.Browser] = append(byBrowser[t.Browser], t)
	}

	totalOpened := 0
	totalFailed := 0

	for bt, bTabs := range byBrowser {
		adapter, ok := c.GetAdapter(bt)
		if !ok {
			totalFailed += len(bTabs)
			continue
		}
		opened, failed, err := adapter.ReopenTabs(ctx, bTabs)
		totalOpened += opened
		totalFailed += failed
		if err != nil {
			return totalOpened, totalFailed, err
		}
	}

	return totalOpened, totalFailed, nil
}

// WakeTabs batches tabs by browser and executes wake/reload operations.
func (c *Collector) WakeTabs(ctx context.Context, tabs []types.Tab) (int, int, error) {
	byBrowser := make(map[types.BrowserType][]types.Tab)
	for _, t := range tabs {
		byBrowser[t.Browser] = append(byBrowser[t.Browser], t)
	}

	totalWoken := 0
	totalFailed := 0

	for bt, bTabs := range byBrowser {
		adapter, ok := c.GetAdapter(bt)
		if !ok {
			totalFailed += len(bTabs)
			continue
		}
		woken, failed, err := adapter.WakeTabs(ctx, bTabs)
		totalWoken += woken
		totalFailed += failed
		if err != nil {
			return totalWoken, totalFailed, err
		}
	}

	return totalWoken, totalFailed, nil
}
