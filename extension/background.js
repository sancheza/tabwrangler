// TabWrangler Companion Extension Service Worker
// Connects to local native messaging host "com.tabwrangler.bridge" and executes tab queries and actions.

const HOST_NAME = "com.tabwrangler.bridge";
let nativePort = null;
let reconnectTimer = null;

function detectBrowser() {
  if (typeof navigator !== "undefined" && navigator.userAgent && navigator.userAgent.toLowerCase().includes("firefox")) {
    return "firefox";
  }
  if (typeof browser !== "undefined" && browser.runtime && browser.runtime.getBrowserInfo) {
    return "firefox";
  }
  return "chrome";
}

function connectNative() {
  if (nativePort) {
    return;
  }

  try {
    const api = (typeof browser !== "undefined" && browser.runtime) ? browser.runtime : chrome.runtime;
    console.log("[TabWrangler] Attempting connectNative to:", HOST_NAME);
    nativePort = api.connectNative(HOST_NAME);

    nativePort.onMessage.addListener((message) => {
      console.log("[TabWrangler] Received native message:", message);
      handleRequest(message);
    });

    nativePort.onDisconnect.addListener(() => {
      // Access lastError to clear error state and prevent Chrome "Unchecked runtime.lastError" errors
      const lastErr = (typeof chrome !== "undefined" && chrome.runtime && chrome.runtime.lastError) ||
                      (typeof browser !== "undefined" && browser.runtime && browser.runtime.lastError);
      console.warn("[TabWrangler] Native bridge disconnected. lastError message:", lastErr ? lastErr.message : "none");
      nativePort = null;
      clearTimeout(reconnectTimer);
      reconnectTimer = setTimeout(connectNative, 3000);
    });

    // Send initial registration handshake
    const bType = detectBrowser();
    console.log("[TabWrangler] Sending registration handshake for:", bType);
    nativePort.postMessage({
      action: "register",
      browser: bType
    });
  } catch (err) {
    console.error("[TabWrangler] Exception in connectNative:", err);
    clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(connectNative, 2000);
  }
}

async function handleRequest(req) {
  if (!req || !req.action) return;

  const api = (typeof browser !== "undefined" && browser.tabs) ? browser : chrome;

  try {
    switch (req.action) {
      case "list_tabs": {
        const tabs = await api.tabs.query({});
        const mapped = tabs.map(t => ({
          id: t.id,
          windowId: t.windowId,
          title: t.title || "",
          url: t.url || "",
          favIconUrl: t.favIconUrl || "",
          active: !!t.active,
          pinned: !!t.pinned,
          discarded: !!t.discarded,
          lastAccessed: t.lastAccessed || 0,
          incognito: !!t.incognito,
          status: t.status || "complete"
        }));

        sendResponse({
          id: req.id,
          action: req.action,
          success: true,
          tabs: mapped
        });
        break;
      }

      case "focus_tab": {
        let focused = false;
        if (req.tab_id) {
          try {
            await api.tabs.update(req.tab_id, { active: true });
            focused = true;
          } catch (e) {}
        }
        if (!focused && req.url) {
          try {
            const matches = await api.tabs.query({ url: req.url });
            if (matches && matches.length > 0) {
              await api.tabs.update(matches[0].id, { active: true });
              if (matches[0].windowId) {
                await api.windows.update(matches[0].windowId, { focused: true });
              }
              focused = true;
            }
          } catch (e) {}
        }
        if (req.window_id) {
          try {
            await api.windows.update(req.window_id, { focused: true });
          } catch (e) {}
        }
        sendResponse({
          id: req.id,
          action: req.action,
          success: true
        });
        break;
      }

      case "discard_tabs": {
        const tabIds = req.tab_ids || (req.tab_id ? [req.tab_id] : []);
        let discarded = 0;
        let skipped = 0;

        for (const tid of tabIds) {
          try {
            // Discarding active tabs or already discarded tabs will throw or fail
            await api.tabs.discard(tid);
            discarded++;
          } catch (e) {
            skipped++;
          }
        }

        sendResponse({
          id: req.id,
          action: req.action,
          success: true,
          discarded,
          skipped
        });
        break;
      }

      case "close_tabs": {
        const tabIds = req.tab_ids || (req.tab_id ? [req.tab_id] : []);
        let closed = 0;
        let failed = 0;

        try {
          await api.tabs.remove(tabIds);
          closed = tabIds.length;
        } catch (e) {
          // Fall back to closing one by one to determine partial failures
          for (const tid of tabIds) {
            try {
              await api.tabs.remove(tid);
              closed++;
            } catch (err) {
              failed++;
            }
          }
        }

        sendResponse({
          id: req.id,
          action: req.action,
          success: true,
          closed,
          failed
        });
        break;
      }

      default:
        sendResponse({
          id: req.id,
          action: req.action,
          success: false,
          error: "Unknown action: " + req.action
        });
    }
  } catch (err) {
    sendResponse({
      id: req.id,
      action: req.action,
      success: false,
      error: err.message || String(err)
    });
  }
}

function sendResponse(resp) {
  if (nativePort) {
    try {
      nativePort.postMessage(resp);
    } catch (e) {}
  }
}

// Connect immediately on startup
connectNative();

// Reconnect whenever user switches or updates tabs if disconnected
try {
  const tabsAPI = (typeof browser !== "undefined" && browser.tabs) ? browser.tabs : (typeof chrome !== "undefined" ? chrome.tabs : null);
  if (tabsAPI && tabsAPI.onActivated) {
    tabsAPI.onActivated.addListener(() => {
      if (!nativePort) connectNative();
    });
  }
} catch (e) {}
