// TabWrangler Companion Extension Service Worker
// Connects to local native messaging host "com.tabwrangler.bridge" and executes tab queries and actions.

const HOST_NAME = "com.tabwrangler.bridge";
let nativePort = null;
let reconnectTimer = null;

function detectBrowser() {
  if (typeof browser !== "undefined" && browser.runtime && browser.runtime.getBrowserInfo) {
    return "firefox";
  }
  return "chrome";
}

function connectNative() {
  if (nativePort) {
    try { nativePort.disconnect(); } catch (e) {}
    nativePort = null;
  }

  try {
    const api = (typeof browser !== "undefined" && browser.runtime) ? browser.runtime : chrome.runtime;
    nativePort = api.connectNative(HOST_NAME);

    nativePort.onMessage.addListener((message) => {
      handleRequest(message);
    });

    nativePort.onDisconnect.addListener(() => {
      nativePort = null;
      // Exponential retry
      clearTimeout(reconnectTimer);
      reconnectTimer = setTimeout(connectNative, 5000);
    });

    // Send initial registration handshake
    nativePort.postMessage({
      action: "register",
      browser: detectBrowser()
    });
  } catch (err) {
    clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(connectNative, 5000);
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
        if (req.tab_id) {
          await api.tabs.update(req.tab_id, { active: true });
        }
        if (req.window_id) {
          await api.windows.update(req.window_id, { focused: true });
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
