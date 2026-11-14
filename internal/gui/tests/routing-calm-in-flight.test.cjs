// Run with Node's test runner and Playwright on the module path; see README.md.
// Reduced motion is honored by the shared tray Routing stage in both places
// it appears: system preference and Settings' Reduce motion. The request rows
// still show active work; dots and the hub pulse stay static. No backend.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const now = new Date();
const ago = (s) => new Date(now.getTime() - s * 1e3).toISOString();
const M = "gpt-5.5";
// the reporter's: one provider's two accounts, round-robin
const order = [1, 2].map((i) => ({ id: `codex#a${i}`, provider: "codex", name: "Codex", who: `codex-${i}`, kind: "account", model: M }));
const route = (id, tries) => ({ id, seq: id, time: ago(60 - id), agent: "codex", model: M, provider: "codex", done: tries.every((x) => x.done),
  status: 200, ms: 900, tokens: 1200, order, tries });
const at = (n, done) => ({ id: `codex#a${n}`, model: M, start: ago(5), done, ...(done ? { status: 200, ms: 900 } : {}) });

function serve(lang, settingsCalm, live) {
  return async (request) => {
    const url = new URL(request.request().url());
    const json = (data) => request.fulfill({ json: data });
    if (url.pathname === "/boot.js") return request.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true,reduceMotion:${settingsCalm}};` });
    if (url.pathname === "/wails/runtime.js") return request.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [{ id: "codex", name: "Codex", path: "/test/config.toml", fields: [] }], profiles: [], settings: { lang, theme: "light", reduceMotion: settingsCalm } });
    if (url.pathname === "/api/gateway/trace") {
      if (url.searchParams.get("wait")) {
        if (live.queued.length) return json(live.queued.shift());
        live.pending.push(request);
        return;
      }
      return json({ mine: true, now: now.toISOString(), seq: 1, totals: { requests: 1, rerouted: 0, errors: 0 }, routes: [route(1, [at(1, true)])] });
    }
    if (url.pathname === "/api/gateway/history") return json({ cut: false, days: [], routes: [] });
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname === "/api/providers") return json({ providers: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await request.fulfill({ body: await fs.readFile(file), contentType });
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const calm of ["system", "settings"]) {
    for (const width of [1100, 420]) {
      test(`${engine} ${calm} reduced motion ${width}px: a request in flight fades, nothing moves`, async (t) => {
        const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
        t.after(() => browser.close());
        const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: calm === "system" ? "reduce" : "no-preference" });
        page.setDefaultTimeout(5000);
        page.setDefaultNavigationTimeout(60000);
        const errors = [];
        page.on("pageerror", (error) => errors.push(error.message));
        const live = { pending: [], queued: [], seq: 1 };
        await page.route("**/*", serve("en", calm === "settings", live));
        const send = (...routes) => {
          const d = { mine: true, now: new Date().toISOString(), seq: ++live.seq, totals: { requests: live.seq }, routes };
          const r = live.pending.shift();
          if (r) r.fulfill({ json: d }); else live.queued.push(d);
        };
        await page.goto("http://jianmu.test/?view=routing");
        await page.waitForFunction(() => document.querySelector(".rt-accts li.on em")?.textContent.trim());
        assert.equal(await page.evaluate(() => calm()), true, "motion is reduced");

        // A request in flight remains visible; reduced motion keeps the
        // shared stage static while the trace updates it.
        send(route(2, [at(1, false)]));
        await page.waitForFunction(() => document.querySelector(".rt-accts li.now"));
        await page.waitForTimeout(300);
        assert.equal(await page.locator(".ps-sky .ps-dot").count(), 0);
        assert.equal(await page.locator(".ps-hub").evaluate((e) => getComputedStyle(e).animationName), "none");
        assert.equal(await page.locator(".rt-flier").count(), 0);

        send(route(2, [at(1, true)]));
        await page.waitForFunction(() => !document.querySelector(".rt-accts li.now"));
        assert.equal(await page.locator(".ps-dot, .ps-wire.on").count(), 0);
        assert.equal(await page.locator(".ps-hub.busy").count(), 0);
        assert.deepEqual(errors, []);
      });
    }
  }
}
