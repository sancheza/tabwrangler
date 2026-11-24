//go:build windows

// Package browser provides cross-platform window activation and host automation utilities.
package browser

import (
	"strings"
	"syscall"
	"unsafe"

	"github.com/sancheza/tabwrangler/core/types"
	"golang.org/x/sys/windows"
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetForegroundWindow       = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procAllowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procShowWindow               = user32.NewProc("ShowWindow")
)

const (
	swRestore = 9
	asfwAny   = 0xFFFFFFFF
)

// ActivateBrowserApplication raises the specified browser application window to the foreground on Windows.
func ActivateBrowserApplication(b types.BrowserType) {
	var targetSubstrings []string
	switch b {
	case types.BrowserChrome:
		targetSubstrings = []string{"chrome", "google chrome"}
	case types.BrowserFirefox:
		targetSubstrings = []string{"firefox", "mozilla firefox"}
	default:
		return
	}

	var targetHwnd windows.HWND

	cb := windows.NewCallback(func(hwnd windows.HWND, lParam uintptr) uintptr {
		// Only check visible windows
		ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if ret == 0 {
			return 1
		}

		length, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
		if length == 0 {
			return 1
		}

		buf := make([]uint16, length+1)
		procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		title := strings.ToLower(syscall.UTF16ToString(buf))

		for _, sub := range targetSubstrings {
			if strings.Contains(title, sub) {
				targetHwnd = hwnd
				return 0 // Stop enumeration on match
			}
		}
		return 1
	})

	procEnumWindows.Call(cb, 0)

	if targetHwnd == 0 {
		return
	}

	forceWindowForeground(targetHwnd)
}

// forceWindowForeground brings the target window to the front, bypassing Windows focus restrictions.
func forceWindowForeground(hwnd windows.HWND) {
	fgWnd, _, _ := procGetForegroundWindow.Call()
	if windows.HWND(fgWnd) == hwnd {
		return
	}

	// Restore window if minimized
	procShowWindow.Call(uintptr(hwnd), uintptr(swRestore))

	currentThreadID := windows.GetCurrentThreadId()
	fgThreadID, _, _ := procGetWindowThreadProcessId.Call(fgWnd, 0)
	targetThreadID, _, _ := procGetWindowThreadProcessId.Call(uintptr(hwnd), 0)

	attached := false
	if fgThreadID != 0 && uint32(fgThreadID) != currentThreadID {
		procAttachThreadInput.Call(uintptr(currentThreadID), fgThreadID, 1)
		procAllowSetForegroundWindow.Call(uintptr(asfwAny))
		if fgThreadID != targetThreadID {
			procAttachThreadInput.Call(targetThreadID, fgThreadID, 1)
		}
		attached = true
	}

	procBringWindowToTop.Call(uintptr(hwnd))
	procSetForegroundWindow.Call(uintptr(hwnd))

	if attached {
		procAttachThreadInput.Call(uintptr(currentThreadID), fgThreadID, 0)
		if fgThreadID != targetThreadID {
			procAttachThreadInput.Call(targetThreadID, fgThreadID, 0)
		}
	}
}
