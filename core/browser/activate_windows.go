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
	procGetClassNameW            = user32.NewProc("GetClassNameW")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetForegroundWindow       = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procAllowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSwitchToThisWindow       = user32.NewProc("SwitchToThisWindow")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procKeybdEvent               = user32.NewProc("keybd_event")
)

const (
	swRestore       = 9
	asfwAny         = 0xFFFFFFFF
	vkMenu          = 0x12 // VK_MENU (Alt key)
	keyeventfKeyup  = 0x0002
	hwndTopmost     = ^uintptr(0) // -1
	hwndNotopmost   = ^uintptr(1) // -2
	hwndTop         = 0
	swpNosize       = 0x0001
	swpNomove       = 0x0002
	swpShowwindow   = 0x0040
)

// ActivateBrowserApplication raises the specified browser application window to the foreground on Windows.
func ActivateBrowserApplication(b types.BrowserType) {
	var targetClasses []string
	var targetSubstrings []string

	switch b {
	case types.BrowserChrome:
		targetClasses = []string{"chrome_widgetwin_1"}
		targetSubstrings = []string{"chrome", "google chrome"}
	case types.BrowserFirefox:
		targetClasses = []string{"mozillawindowclass"}
		targetSubstrings = []string{"firefox", "mozilla firefox"}
	default:
		return
	}

	var targetHwnd windows.HWND

	cb := windows.NewCallback(func(hwnd windows.HWND, lParam uintptr) uintptr {
		ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if ret == 0 {
			return 1
		}

		// Check class name
		classBuf := make([]uint16, 256)
		cLen, _, _ := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&classBuf[0])), uintptr(len(classBuf)))
		className := ""
		if cLen > 0 {
			className = strings.ToLower(syscall.UTF16ToString(classBuf[:cLen]))
		}

		// Check window title
		titleBuf := make([]uint16, 512)
		tLen, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len(titleBuf)))
		title := ""
		if tLen > 0 {
			title = strings.ToLower(syscall.UTF16ToString(titleBuf[:tLen]))
		}

		// Match either by browser window class or title substring
		matchClass := false
		for _, tc := range targetClasses {
			if className == tc {
				matchClass = true
				break
			}
		}

		matchTitle := false
		for _, ts := range targetSubstrings {
			if strings.Contains(title, ts) {
				matchTitle = true
				break
			}
		}

		if matchClass || matchTitle {
			// Prefer window with non-empty title (the main browser frame)
			targetHwnd = hwnd
			if title != "" {
				return 0 // Found ideal top-level browser window, stop enumeration
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

	// 1. Restore window if minimized
	procShowWindow.Call(uintptr(hwnd), uintptr(swRestore))

	// 2. Temporarily bring to TOPMOST then NOTOPMOST to force Z-order elevation over terminal
	procSetWindowPos.Call(uintptr(hwnd), hwndTopmost, 0, 0, 0, 0, swpNomove|swpNosize|swpShowwindow)
	procSetWindowPos.Call(uintptr(hwnd), hwndNotopmost, 0, 0, 0, 0, swpNomove|swpNosize|swpShowwindow)

	// 3. AttachThreadInput & AllowSetForegroundWindow bypass
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

	// 4. Simulate ALT keypress to unlock Windows foreground lock timer
	procKeybdEvent.Call(vkMenu, 0, 0, 0)

	// 5. Bring to top, set foreground, and invoke SwitchToThisWindow
	procBringWindowToTop.Call(uintptr(hwnd))
	procSetForegroundWindow.Call(uintptr(hwnd))
	procSwitchToThisWindow.Call(uintptr(hwnd), 1)

	// Release ALT keypress
	procKeybdEvent.Call(vkMenu, 0, keyeventfKeyup, 0)

	if attached {
		procAttachThreadInput.Call(uintptr(currentThreadID), fgThreadID, 0)
		if fgThreadID != targetThreadID {
			procAttachThreadInput.Call(targetThreadID, fgThreadID, 0)
		}
	}
}
