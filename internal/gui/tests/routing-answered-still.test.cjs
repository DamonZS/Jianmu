// Run with Node's test runner and Playwright on the module path; see README.md.
// A request that is over doesn't look called on the Routing page (#2386,
// Moody-Sin: jianmu on a server, the agent long stopped, and the page and
// the app still showed the account that answered the last request as being
// called, however often reloaded). The row that answered the request shown
// stays lit (its dot filled, its wire drawn in the agent's colour), but
// only a request in flight moves: the row is `now`, its dot pulses and its
// wire flows. The same for a fold of keys and for the row it folds. Once
// the request is done nothing on the stage flows. English, Chinese
// (simplified and traditional), Japanese and German, at 1100px and 420px;
// no backend, the API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const now = new Date();
const ago = (s) => new Date(now.getTime() - s * 1e3).toISOString();
const M = "gemini-3.8-flash";
// the reporter's: a provider's accounts (three, so they fold) and one more
// account of another beside them in a group
const order = [
  ...[1, 2, 3].map((i) => ({ id: `ag#a${i}`, provider: "ag", name: "Antigravity", who: `ag-${i}`, kind: "key", model: M })),
  { id: "solo#s1", provider: "solo", name: "Solo", who: "solo-1", kind: "key", model: M },
];
const group = { id: "flash", name: "Flash", routing: "", members: ["ag/" + M, "solo/" + M] };
const route = (id, tries) => ({ id, seq: id, time: ago(60 - id), agent: "zcode", model: "flash", provider: tries[0].id.split("#")[0], done: tries.every((x) => x.done),
  status: 200, ms: 900, tokens: 1200, group, order, tries });
const solo = (done) => ({ id: "solo#s1", model: M, start: ago(50), done, ...(done ? { status: 200, ms: 900 } : {}) });
const ag = (done) => ({ id: "ag#a2", model: M, start: ago(5), done, ...(done ? { status: 200, ms: 900 } : {}) });

function serve(lang, live) {
  return async (request) => {
    const url = new URL(request.request().url());
    const json = (data) => request.fulfill({ json: data });
    if (url.pathname === "/boot.js") return request.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return request.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [{ id: "zcode", name: "ZCode", path: "/test/config.json", fields: [] }], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/gateway/trace") {
      if (url.searchParams.get("wait")) {
        if (live.queued.length) return json(live.queued.shift());
        live.pending.push(request);
        return;
      }
      return json({ mine: true, now: now.toISOString(), seq: 1, totals: { requests: 1, rerouted: 0, errors: 0 }, routes: [route(1, [solo(true)])] });
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
  for (const lang of ["en", "zh", "zh-TW", "ja", "de"]) {
    for (const width of [1100, 420]) {
      test(`${engine} ${lang} ${width}px: the row that answered stays lit and still; only a request in flight flows`, async (t) => {
        const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
        t.after(() => browser.close());
        // motion on: what is checked is whether the dot pulses
        const page = await browser.newPage({ viewport: { width, height: 900 } });
        page.setDefaultTimeout(5000);
        page.setDefaultNavigationTimeout(60000);
        const errors = [];
        page.on("pageerror", (error) => errors.push(error.message));
        const live = { pending: [], queued: [], seq: 1 };
        await page.route("**/*", serve(lang, live));
        const send = (...routes) => {
          const d = { mine: true, now: new Date().toISOString(), seq: ++live.seq, totals: { requests: live.seq }, routes };
          const r = live.pending.shift();
          if (r) r.fulfill({ json: d }); else live.queued.push(d);
        };
        const say = (key, vars = {}) => page.evaluate(([lang, key, vars]) => {
          const s = (lang !== "en" && I18N[lang]?.[key]) || key;
          return s.replace(/\{(\w+)\}/g, (_, k) => vars[k] ?? `{${k}}`);
        }, [lang, key, vars]);
        const row = page.locator(".rt-accts li:not(.rt-fold)", { has: page.locator(".who", { hasText: /^solo-1$/ }) });
        const fold = page.locator(".rt-accts li.rt-fold");
        await page.goto("http://jianmu.test/?view=routing");
        await row.waitFor();
        await page.waitForFunction(() => document.querySelector(".rt-accts li.on em")?.textContent.trim());
        assert.equal((await row.locator("em").textContent()).trim(), await say("answered this request"));
        assert.equal(await page.locator(".ps-dot").count(), 0);

        send(route(2, [ag(false)]));
        await page.locator(".ps-dot.wait").waitFor();
        assert.equal(await page.locator(".ps-hub.busy").count(), 1);

        send(route(2, [ag(true)]));
        await page.locator(".ps-dot.ok").waitFor();
        await page.locator(".ps-dot").waitFor({ state: "detached" });
        assert.equal(await page.locator(".ps-hub.busy, .ps-wire.on").count(), 0);
        await page.reload();
        await row.waitFor();
        await page.waitForFunction(() => document.querySelector(".rt-accts li.on em")?.textContent.trim());
        assert.equal(await page.locator(".ps-dot").count(), 0);
        assert.equal(await page.locator(".ps-hub.busy").count(), 0);
        assert.deepEqual(errors, []);
      });
    }
  }
}
