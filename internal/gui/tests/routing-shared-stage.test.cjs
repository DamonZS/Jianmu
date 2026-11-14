const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const seat = (id) => ({ id, provider: id, name: id, kind: "key", model: "model" });
const a = seat("Alpha"), b = seat("Beta");
const request = (tries, done = false, id = 2) => ({
  id, seq: id, time: new Date().toISOString(), agent: "codex", model: "Alpha/model", provider: "Alpha",
  order: [a, b], tries: tries.map((tr) => ({ model: "model", start: new Date().toISOString(), ...tr })), done,
  status: done ? tries.at(-1).status : 0, ms: 1000, tokens: done ? 100 : 0,
});

function serve(lang, feed) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs={lang:${JSON.stringify(lang)},theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [{ id: "codex", name: "Codex", path: "/test/codex", fields: [] }], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/groups") return json({ groups: [], models: [], pools: [] });
    if (url.pathname === "/api/gateway/history") return json({ days: [], routes: [] });
    if (url.pathname === "/api/usage/quotas") return json([]);
    if (url.pathname === "/api/gateway/trace") {
      const reply = (routes) => json({ mine: true, now: new Date().toISOString(), seq: ++feed.seq, routes, totals: { requests: 1, rerouted: 0, errors: 0 } }).catch(() => {});
      if (!url.searchParams.get("wait")) return reply([request([{ id: a.id, done: true, status: 200 }], true, 1)]);
      if (feed.next) { const next = feed.next; feed.next = null; return reply(next); }
      feed.send = reply;
      return;
    }
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    try {
      await route.fulfill({ body: await fs.readFile(file), contentType: { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml" }[path.extname(file)] });
    } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

for (const engine of process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"]) {
  for (const lang of ["en", "zh", "zh-TW", "ja", "de"]) for (const mode of ["window", "panel"]) {
    test(`${engine} ${lang} ${mode}: the shared tray stage waits, fails over and returns its reply`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      t.after(() => browser.close());
      const page = await (await browser.newContext({ viewport: { width: mode === "panel" ? 440 : 1100, height: 800 } })).newPage();
      page.setDefaultTimeout(8000);
      const errors = [], feed = { seq: 1, send: null, next: null };
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve(lang, feed));
      await page.goto(mode === "panel" ? "http://jianmu.test/?mode=panel" : "http://jianmu.test/?view=routing");
      if (mode === "panel") await page.locator('[data-ptab="routing"]').click();
      const stage = page.locator(mode === "panel" ? "#panelRouting .pr-stage" : "#view-routing .pr-stage");
      await stage.waitFor();
      if (mode === "window") {
        assert.equal(await page.locator(".rt-stage.shared").isVisible(), true, "the account/key list stays under the shared stage");
        assert.equal(await page.locator(".rt-stage.shared .rt-accts").isVisible(), true);
        assert.equal(await page.locator(".rt-stage.shared .rt-hub").isVisible(), false, "the duplicate graph is hidden");
        assert.equal(await page.locator(".rt-shared-meta b").textContent(), "jianmu");
      }
      assert.deepEqual(await stage.evaluate((el) => {
        const hub = el.querySelector(".ps-hub"), s = getComputedStyle(hub);
        return [s.width, s.height, el.querySelectorAll(".rt-flier").length];
      }), ["40px", "40px", 0]);
      const send = async (r) => {
        if (feed.send) { const reply = feed.send; feed.send = null; await reply([r]); }
        else feed.next = [r];
      };
      await send(request([{ id: a.id }]));
      await stage.locator(".ps-dot.wait").waitFor({ state: "attached" });
      assert.equal(await stage.locator(".ps-hub.busy").count(), 1);
      assert.equal(await stage.locator(".ps-dot").getAttribute("r"), "3.5");
      assert.equal(await stage.locator(".ps-hub").evaluate((el) => getComputedStyle(el).animationName), "ps-ring");
      await send(request([{ id: a.id, done: true, status: 429, fail: "rate", again: true }, { id: b.id }]));
      await stage.locator(".ps-dot.bad").waitFor({ state: "attached" });
      await page.waitForFunction(() => document.querySelector(".pr-stage .ps-dot.wait:not(.bad)"));
      await send(request([{ id: a.id, done: true, status: 429, fail: "rate", again: true }, { id: b.id, done: true, status: 200 }], true));
      await stage.locator(".ps-dot.ok").waitFor({ state: "attached" });
      await stage.locator(".ps-dot").waitFor({ state: "detached" });
      assert.equal(await stage.locator(".ps-hub.busy, .ps-wire.on").count(), 0);
      await page.emulateMedia({ reducedMotion: "reduce" });
      await send(request([{ id: a.id }], false, 3));
      await page.waitForTimeout(300);
      assert.equal(await stage.locator(".ps-dot").count(), 0, "reduced motion does not launch a dot");
      assert.deepEqual(errors, []);
      if (process.env.ARTIFACT_DIR) {
        await fs.mkdir(process.env.ARTIFACT_DIR, { recursive: true });
        await page.screenshot({ path: path.join(process.env.ARTIFACT_DIR, `${engine}-${lang}-${mode}-shared-stage.png`) });
      }
    });
  }
}
