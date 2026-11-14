// Run with Node's test runner and Playwright on the module path; see README.md.
// One plugin is one card. The Jianmu catalog lists a package, and the same
// plugin's repository is also tagged jianmu-plugin on GitHub: Discover shows
// the catalog's card alone. Before this, the two were matched by the package
// name spelled exactly, so a repository whose package.json still says
// @magpie-community/… beside a migrated @jianmu/… listing (or a listing whose
// npm repository is the discovered one) showed the plugin twice.
// A repository that is not the listed one stays a card of its own, even
// when its display name matches. The API is faked.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

// the catalog: two packages this community lists, the first also published
// from a repository GitHub discovers
const LISTINGS = [
  { package: "@jianmu/opencode-zed-auth", name: "Zed", community: true,
    npm: { version: "0.1.9", repository: "https://github.com/acme/zed-plugin" } },
  { package: "@jianmu/middleware-legacy-auth", name: "Legacy", kind: "middleware", community: true },
];
// GitHub's tagged repositories: the listed plugin by its repository and by
// its old package name (npm says nothing of this one's repository), and
// another author's plugin by the same display name
const REPOS = [
  { spec: "github:acme/zed-plugin", repo: "acme/zed-plugin", url: "https://github.com/acme/zed-plugin",
    owner: "acme", description: "Zed models", package: "@magpie-community/opencode-zed-auth", version: "0.1.9" },
  { spec: "github:acme/legacy-plugin", repo: "acme/legacy-plugin", url: "https://github.com/acme/legacy-plugin",
    owner: "acme", description: "A legacy middleware", package: "@magpie-community/middleware-legacy-auth", kind: "middleware", version: "0.2.0" },
  { spec: "github:other/zed-plugin", repo: "other/zed-plugin", url: "https://github.com/other/zed-plugin",
    owner: "other", description: "Another Zed plugin", package: "other-zed-plugin", name: "Zed", version: "0.0.1" },
];

function server(fetched) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"en",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang: "en", theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true }, plugins: [] });
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname === "/api/gateway/trace") return json({ routes: [] });
    if (url.pathname === "/api/plugins") return json({ bun: true, bunVersion: "1.4.2", plugins: [], picker: false, movable: [] });
    if (url.pathname === "/api/plugins/listings") return json({ listings: LISTINGS.map((l) => ({ ...l, npm: undefined })) });
    // npm's answers, as the page merges them onto the listings: the first's
    // repository is the one GitHub discovers, the second's is unknown
    if (url.pathname === "/api/plugins/npm") return json({ npm: { [LISTINGS[0].package]: LISTINGS[0].npm } });
    if (url.pathname === "/api/plugins/github") return json({ repos: REPOS, topic: "jianmu-plugin" });
    if (url.pathname === "/api/plugins/search") return json({ hits: [] });
    if (url.pathname === "/api/plugins/updates") return json({ waiting: [], updated: [] });
    if (url.pathname === "/api/plugins/page") return json({ readme: "", updated: "0001-01-01T00:00:00Z" });
    if (url.pathname.startsWith("/api/")) return json({});
    if (url.host !== "jianmu.test" || !/^\/[\w./-]*$/.test(url.pathname)) { fetched.push(url.href); return route.fulfill({ status: 404, body: "" }); }
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium"])) {
  test(engine + ": a listed plugin's repository is not a second card", async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || "chromium" }));
    t.after(() => browser.close());
    await t.test("catalog and discovery show one Zed, and a stranger's Zed", async () => {
      const context = await browser.newContext({ viewport: { width: 1100, height: 860 } });
      t.after(() => context.close());
      const page = await context.newPage(), fetched = [], errors = [];
      page.setDefaultTimeout(8000);
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", server(fetched));
      await page.goto("http://jianmu.test/?view=plugins");
      const view = page.locator("#view-plugins");
      await view.locator(".pm-ghlook").waitFor({ state: "detached" });

      const catalog = view.locator('.pm-card[data-pkg="@jianmu/opencode-zed-auth"]');
      await catalog.waitFor();
      assert.equal(await catalog.locator(".pm-name b").innerText(), "Zed");
      // the listed plugin's repository, and the repository whose package
      // carries the old scope, are the catalog's cards alone
      assert.equal(await view.locator('.pm-card[data-pkg="github:acme/zed-plugin"]').count(), 0, "the listed repository is not shown again");
      assert.equal(await view.locator('.pm-card[data-pkg="github:acme/legacy-plugin"]').count(), 0, "the old scope is the listed package");
      // another author's plugin of the same display name is its own card
      const stranger = view.locator('.pm-card[data-pkg="github:other/zed-plugin"]');
      assert.equal(await stranger.count(), 1);
      assert.equal((await stranger.locator(".pm-name b").innerText()).trim(), "Zed");
      // one card each: the two listings and the stranger
      assert.equal(await view.locator(".pm-card").filter({ has: page.locator(".pm-name b", { hasText: /^Zed$/ }) }).count(), 2);
      assert.deepEqual(errors, []);
      assert.deepEqual(fetched, []);
    });
  });
}
