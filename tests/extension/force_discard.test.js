// Tests the force-hibernate logic in extension/background.js with a stubbed browser API.
// Run: node tests/extension/force_discard.test.js
const fs = require("fs");
const path = require("path");
const vm = require("vm");
const assert = require("assert");

// Objects created inside the sandbox have a different prototype, so compare them as JSON.
const same = (actual, expected) => assert.strictEqual(JSON.stringify(actual), JSON.stringify(expected));

const code = fs.readFileSync(path.join(__dirname, "..", "..", "extension", "background.js"), "utf8");

function run(tabs, request) {
  const calls = [];
  const sent = [];
  const noop = { addListener() {} };
  const port = { onMessage: noop, onDisconnect: noop, postMessage: (m) => sent.push(m) };
  const byId = new Map(tabs.map((t) => [t.id, t]));
  const chrome = {
    runtime: { connectNative: () => port, onInstalled: noop, onStartup: noop, getManifest: () => ({ version: "0.0.0" }) },
    tabs: {
      onActivated: noop, onCreated: noop, onUpdated: noop,
      get: async (id) => { if (!byId.has(id)) throw new Error("no tab"); return byId.get(id); },
      query: async (q) => tabs.filter((t) => q.windowId === undefined || t.windowId === q.windowId),
      update: async (id, props) => {
        calls.push(["update", id, props]);
        if (props.active) {
          // Like a browser: exactly one active tab per window, and showing a tab loads it.
          const target = byId.get(id);
          tabs.filter((t) => t.windowId === target.windowId).forEach((t) => { t.active = t.id === id; });
          target.discarded = false;
        }
      },
      discard: async (id) => {
        const t = byId.get(id);
        if (t.active) throw new Error("cannot discard the active tab");
        if (t.discarded) throw new Error("already discarded");
        calls.push(["discard", id]);
        t.discarded = true;
        return t;
      },
    },
    windows: { onFocusChanged: noop },
  };
  const sandbox = { chrome, console: { log() {}, warn() {}, error() {} }, setTimeout: () => 0, clearTimeout() {}, navigator: { userAgent: "test" } };
  vm.createContext(sandbox);
  vm.runInContext(code, sandbox);
  return sandbox.handleRequest(request).then(() => ({ calls, response: sent[sent.length - 1] }));
}

const tab = (id, extra = {}) => ({ id, windowId: 1, active: false, discarded: false, lastAccessed: id, ...extra });
const discardReq = (ids, force) => ({ id: "r", action: "discard_tabs", tab_ids: ids, force });

(async () => {
  // Without force, the active tab is skipped and nothing is switched.
  let r = await run([tab(1, { active: true }), tab(2)], discardReq([1], false));
  same(r.calls, []);
  assert.strictEqual(r.response.discarded, 0);
  assert.strictEqual(r.response.skipped, 1);

  // Force: switch to the most recently used loaded tab, then discard the active one.
  r = await run([tab(1, { active: true }), tab(2, { lastAccessed: 50 }), tab(3, { lastAccessed: 90 })], discardReq([1], true));
  same(r.calls, [["update", 3, { active: true }], ["discard", 1]]);
  assert.strictEqual(r.response.discarded, 1);

  // Force never switches to an already discarded tab (that would reload it).
  r = await run([tab(1, { active: true }), tab(2, { discarded: true, lastAccessed: 99 }), tab(3, { lastAccessed: 10 })], discardReq([1], true));
  same(r.calls, [["update", 3, { active: true }], ["discard", 1]]);

  // Force never switches to a tab that is also being discarded.
  r = await run([tab(1, { active: true }), tab(2, { lastAccessed: 99 }), tab(3, { lastAccessed: 10 })], discardReq([1, 2], true));
  same(r.calls, [["update", 3, { active: true }], ["discard", 1], ["discard", 2]]);
  assert.strictEqual(r.response.discarded, 2);

  // No loaded tab to switch to: skip, and do not touch the window.
  r = await run([tab(1, { active: true }), tab(2, { discarded: true })], discardReq([1], true));
  same(r.calls, []);
  assert.strictEqual(r.response.skipped, 1);

  // A window's only tab cannot be hibernated.
  r = await run([tab(1, { active: true })], discardReq([1], true));
  same(r.calls, []);
  assert.strictEqual(r.response.skipped, 1);

  // Only the window of the active tab is searched for a replacement.
  r = await run([tab(1, { active: true }), tab(2, { windowId: 2, lastAccessed: 99 }), tab(3, { lastAccessed: 5 })], discardReq([1], true));
  same(r.calls, [["update", 3, { active: true }], ["discard", 1]]);

  // A background tab is discarded without switching anything, with or without force.
  r = await run([tab(1, { active: true }), tab(2)], discardReq([2], true));
  same(r.calls, [["discard", 2]]);

  console.log("force_discard: all checks passed");
})().catch((e) => { console.error(e); process.exit(1); });
