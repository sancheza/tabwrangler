//go:build windows

// Package browser provides cross-platform window activation and host automation utilities.
package browser

import (
	"runtime"
	"strings"
	"syscall"
	"time"
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
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
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
	swRestore      = 9
	asfwAny        = 0xFFFFFFFF
	vkMenu         = 0x12 // VK_MENU (Alt key)
	keyeventfKeyup = 0x0002
	hwndTopmost    = ^uintptr(0) // -1
	hwndNotopmost  = ^uintptr(1) // -2
	hwndTop        = 0
	swpNosize      = 0x0001
	swpNomove      = 0x0002
	swpShowwindow  = 0x0040
)

// ActivateBrowserApplication raises the frontmost window of the specified browser to the foreground on Windows.
func ActivateBrowserApplication(b types.BrowserType) {
	activateWindow(b, "")
}

// processImageName returns the lower-case executable file name of the process
// that owns hwnd, or "" when it cannot be read (for example an elevated process).
func processImageName(hwnd windows.HWND) string {
	var pid uint32
	procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	full := syscall.UTF16ToString(buf[:size])
	if i := strings.LastIndexAny(full, `\/`); i >= 0 {
		full = full[i+1:]
	}
	return strings.ToLower(full)
}

// listWindows returns the visible, titled top-level windows from front to back.
func listWindows() []WindowCandidate {
	var out []WindowCandidate
	cb := windows.NewCallback(func(hwnd windows.HWND, lParam uintptr) uintptr {
		if ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd)); ret == 0 {
			return 1
		}
		titleBuf := make([]uint16, 512)
		tLen, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len(titleBuf)))
		if tLen == 0 {
			return 1
		}
		classBuf := make([]uint16, 256)
		cLen, _, _ := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&classBuf[0])), uintptr(len(classBuf)))
		out = append(out, WindowCandidate{
			Handle: uintptr(hwnd),
			Class:  syscall.UTF16ToString(classBuf[:cLen]),
			Title:  syscall.UTF16ToString(titleBuf[:tLen]),
			Exe:    processImageName(hwnd),
		})
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return out
}

// activateWindow raises the browser window chosen by ChooseWindow. The browser
// updates its window title a moment after a tab switch, so the search repeats
// briefly until a window title contains the hint.
func activateWindow(b types.BrowserType, titleHint string) {
	if b != types.BrowserChrome && b != types.BrowserFirefox {
		return
	}
	var cands []WindowCandidate
	idx := -1
	for attempt := 0; attempt < 6; attempt++ {
		cands = listWindows()
		idx = ChooseWindow(b, titleHint, cands)
		if idx < 0 || titleHint == "" {
			break
		}
		if strings.Contains(strings.ToLower(cands[idx].Title), strings.ToLower(strings.TrimSpace(titleHint))) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if idx < 0 {
		Debug("activate %s: no matching window among %d visible windows", b, len(cands))
		return
	}
	target := windows.HWND(cands[idx].Handle)
	ok := forceWindowForeground(target)
	Debug("activate %s: window %d of %d (exe=%q class=%q) foreground=%v", b, idx+1, len(cands), cands[idx].Exe, cands[idx].Class, ok)
}

// forceWindowForeground brings the target window to the front, bypassing Windows focus restrictions.
//
// AttachThreadInput and SetForegroundWindow act on the calling OS thread, so the
// goroutine stays on one thread for the whole sequence. It returns whether the
// window is in the foreground afterward.
func forceWindowForeground(hwnd windows.HWND) bool {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	fgWnd, _, _ := procGetForegroundWindow.Call()
	if windows.HWND(fgWnd) == hwnd {
		return true
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

	now, _, _ := procGetForegroundWindow.Call()
	return windows.HWND(now) == hwnd
}
