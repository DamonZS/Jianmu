// The shared tray stage keeps simultaneous requests off the node labels.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const assets = path.resolve(__dirname, "../assets");
const order = [{ id: "b", provider: "b", name: "B", kind: "account", model: "opus" },
  { id: "a", provider: "a", name: "A", kind: "account", model: "fast" }];
const routes = () => [
  { id: 7, seq: 7, time: new Date().toISOString(), agent: "claude", model: "b/opus", provider: "b", order,
    tries: [{ id: "b", model: "opus", start: new Date().toISOString() }], done: false },
  { id: 8, seq: 8, time: new Date().toISOString(), agent: "codex", model: "a/fast", provider: "a", order: order.slice(1),
    tries: [{ id: "a", model: "fast", start: new Date().toISOString() }], done: false },
  { id: 9, seq: 9, time: new Date().toISOString(), agent: "kimi", model: "a/fast", provider: "", order: [], tries: [], done: false },
];
function serve(lang, live) {
  return async (route) => {
    const url = new URL(route.request().url()), json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs={lang:${JSON.stringify(lang)},theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: ["claude", "codex", "kimi"].map((id) => ({ id, name: id, path: "/test/" + id, fields: [] })), profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/gateway/trace") {
      if (url.searchParams.get("wait")) { live.pending.push(route); return; }
      return json({ mine: true, now: new Date().toISOString(), seq: 9, routes: routes(), totals: { requests: 3, rerouted: 0, errors: 0 } });
    }
    if (url.pathname === "/api/gateway/history") return json({ days: [], routes: [] });
    if (url.pathname === "/api/groups") return json({ groups: [], models: [], pools: [] });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    await route.fulfill({ body: await fs.readFile(file), contentType: { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml" }[path.extname(file)] });
  };
}
for (const engine of process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"]) {
  for (const lang of ["en", "zh", "zh-TW", "ja", "de"]) for (const width of [1600, 1100, 560, 390]) {
    test(`${engine} ${lang} ${width}px: the shared stage holds three request dots beside its cards`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      t.after(() => browser.close());
      const page = await (await browser.newContext({ viewport: { width, height: 1000 } })).newPage();
      page.setDefaultTimeout(8000);
      const errors = [], live = { pending: [] };
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve(lang, live));
      await page.goto("http://jianmu.test/?view=routing");
      await page.waitForFunction(() => document.querySelectorAll("#view-routing .ps-dot.wait").length === 2);
      assert.equal(await page.locator("#view-routing .ps-dot").count(), 3);
      assert.equal(await page.locator(".rt-flier").count(), 0);
      assert.equal(await page.locator("#view-routing .ps-hub.busy").count(), 1);
      const covered = await page.evaluate(() => {
        const labels = [...document.querySelectorAll("#view-routing .ps-name")].map((e) => e.getBoundingClientRect());
        return [...document.querySelectorAll("#view-routing .ps-dot")].filter((d) => d.getAttribute("visibility") !== "hidden").some((d) => {
          const a = d.getBoundingClientRect();
          return labels.some((b) => a.right > b.left + 1 && a.left < b.right - 1 && a.bottom > b.top + 1 && a.top < b.bottom - 1);
        });
      });
      assert.equal(covered, false, "dots do not cover labels while waiting");
      const next = { ...routes()[2], order: order.slice(1), provider: "a", tries: [{ id: "a", model: "fast", start: new Date().toISOString() }] };
      await page.waitForFunction(() => document.querySelectorAll(".rt-req").length === 3);
      assert(live.pending.length, "the trace is waiting for its next update");
      await live.pending.shift().fulfill({ json: { mine: true, now: new Date().toISOString(), seq: 10, routes: [next], totals: { requests: 3, rerouted: 0, errors: 0 } } });
      await page.waitForFunction(() => document.querySelectorAll("#view-routing .ps-dot.wait").length === 3);
      assert.equal(await page.locator(".rt-stage.shared .rt-hub").isVisible(), false);
      assert.equal(await page.locator(".rt-stage.shared .rt-accts").isVisible(), true);
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth));
      assert.deepEqual(errors, []);
    });
  }
}
