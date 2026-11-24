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
    let extVersion = "";
    try {
      extVersion = api.getManifest().version || "";
    } catch (e) {}
    nativePort.postMessage({
      action: "register",
      browser: bType,
      version: extVersion
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
          lastAccessed: Math.round(t.lastAccessed || 0),
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
        let targetWindowId = req.window_id || null;
        if (req.tab_id) {
          try {
            const tab = await api.tabs.update(req.tab_id, { active: true });
            if (tab && tab.windowId) {
              targetWindowId = tab.windowId;
            }
            focused = true;
          } catch (e) {}
        }
        if (!focused && req.url) {
          try {
            const matches = await api.tabs.query({ url: req.url });
            if (matches && matches.length > 0) {
              const tab = await api.tabs.update(matches[0].id, { active: true });
              if (tab && tab.windowId) {
                targetWindowId = tab.windowId;
              } else if (matches[0].windowId) {
                targetWindowId = matches[0].windowId;
              }
              focused = true;
            }
          } catch (e) {}
        }
        if (targetWindowId) {
          try {
            await api.windows.update(targetWindowId, { focused: true });
          } catch (e) {}
          try {
            await api.windows.update(targetWindowId, { drawAttention: true });
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

        const selected = new Set(tabIds);

        for (const tid of tabIds) {
          try {
            // A browser cannot discard the tab showing in its window. Switch that window to the
            // most recently used loaded tab that is not being discarded, then discard this one.
            const tab = await api.tabs.get(tid);
            if (tab.active) {
              const others = (await api.tabs.query({ windowId: tab.windowId }))
                .filter(t => t.id !== tid && !selected.has(t.id) && !t.discarded)
                .sort((a, b) => (b.lastAccessed || 0) - (a.lastAccessed || 0));
              if (others.length === 0) {
                skipped++;
                continue;
              }
              await api.tabs.update(others[0].id, { active: true });
            }
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

      case "reopen_tabs": {
        const urls = req.urls || (req.url ? [req.url] : []);
        let opened = 0;
        let failed = 0;

        for (const u of urls) {
          try {
            const createOpts = { url: u, active: false };
            if (req.window_id) createOpts.windowId = req.window_id;
            await api.tabs.create(createOpts);
            opened++;
          } catch (e) {
            try {
              await api.tabs.create({ url: u, active: false });
              opened++;
            } catch (e2) {
              failed++;
            }
          }
        }

        sendResponse({
          id: req.id,
          action: req.action,
          success: true,
          opened,
          failed
        });
        break;
      }

      case "wake_tabs": {
        const tabIds = req.tab_ids || (req.tab_id ? [req.tab_id] : []);
        let woken = 0;
        let failed = 0;

        for (const tid of tabIds) {
          try {
            await api.tabs.reload(tid);
            woken++;
          } catch (e) {
            failed++;
          }
        }

        sendResponse({
          id: req.id,
          action: req.action,
          success: true,
          woken,
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

// Manifest V3 lifecycle hooks
const runtimeAPI = (typeof browser !== "undefined" && browser.runtime) ? browser.runtime : (typeof chrome !== "undefined" ? chrome.runtime : null);

if (runtimeAPI) {
  // Fires when unpacked extension is first loaded or re-installed
  if (runtimeAPI.onInstalled) {
    runtimeAPI.onInstalled.addListener(() => {
      console.log("[TabWrangler] onInstalled fired, connecting native host...");
      connectNative();
    });
  }

  // Fires when browser launches
  if (runtimeAPI.onStartup) {
    runtimeAPI.onStartup.addListener(() => {
      console.log("[TabWrangler] onStartup fired, connecting native host...");
      connectNative();
    });
  }
}

// Connect immediately on initial script evaluation
connectNative();

// Comprehensive browser activity hooks to reconnect if disconnected
try {
  const tabsAPI = (typeof browser !== "undefined" && browser.tabs) ? browser.tabs : (typeof chrome !== "undefined" ? chrome.tabs : null);
  if (tabsAPI) {
    if (tabsAPI.onActivated) {
      tabsAPI.onActivated.addListener(() => {
        if (!nativePort) connectNative();
      });
    }
    if (tabsAPI.onCreated) {
      tabsAPI.onCreated.addListener(() => {
        if (!nativePort) connectNative();
      });
    }
    if (tabsAPI.onUpdated) {
      tabsAPI.onUpdated.addListener(() => {
        if (!nativePort) connectNative();
      });
    }
  }

  const windowsAPI = (typeof browser !== "undefined" && browser.windows) ? browser.windows : (typeof chrome !== "undefined" ? chrome.windows : null);
  if (windowsAPI && windowsAPI.onFocusChanged) {
    windowsAPI.onFocusChanged.addListener(() => {
      if (!nativePort) connectNative();
    });
  }
} catch (e) {}
