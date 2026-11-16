// MCP and skill market categories are derived locally, clickable, and filter the
// current search results without another market request. No live registry is hit.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const { reach } = require("./reader.cjs");

const assets = path.resolve(__dirname, "../assets");
const categories = ["all", "browse", "development", "data", "knowledge", "productivity", "media", "security", "other"];
const words = {
  en: ["All categories", "Browse & search", "Development & testing", "Data & databases", "Knowledge & memory", "Office & automation", "Media & design", "Security & operations", "Other resources"],
  zh: ["全部", "浏览与搜索", "开发与测试", "数据与数据库", "知识与记忆", "办公与自动化", "媒体与设计", "安全与运维", "其他资源"],
  "zh-TW": ["全部", "瀏覽與搜尋", "開發與測試", "資料與資料庫", "知識與記憶", "辦公與自動化", "媒體與設計", "安全與維運", "其他資源"],
  ja: ["すべて", "閲覧・検索", "開発・テスト", "データ・データベース", "知識・記憶", "事務・自動化", "メディア・デザイン", "セキュリティ・運用", "その他のリソース"],
  de: ["Alle Kategorien", "Browsen & Suche", "Entwicklung & Tests", "Daten & Datenbanken", "Wissen & Gedächtnis", "Büro & Automatisierung", "Medien & Design", "Sicherheit & Betrieb", "Weitere Ressourcen"],
};
const mcpExamples = [
  { id: "playwright", name: "Playwright", title: "Playwright", description: "Control a browser and web pages.", transport: "stdio", runs: "npx", inputs: [] },
  { id: "postgres", name: "PostgreSQL", title: "PostgreSQL database", description: "Run SQL queries against a database.", transport: "stdio", runs: "npx", inputs: [] },
  { id: "memory", name: "Memory", title: "Memory", description: "Keep a knowledge graph across conversations.", transport: "stdio", runs: "npx", inputs: [] },
  { id: "calendar", name: "Calendar", title: "Calendar", description: "Manage events and tasks.", transport: "stdio", runs: "npx", inputs: [] },
  { id: "imagegen", name: "Image Generator", title: "Image generation", description: "Create images from a prompt.", transport: "stdio", runs: "npx", inputs: [] },
  { id: "sentry", name: "Sentry", title: "Sentry", description: "Review security issues and error reports.", transport: "http", inputs: [] },
  { id: "mcp-utility", name: "Utility", title: "General utility", description: "A general-purpose helper.", transport: "stdio", runs: "npx", inputs: [] },
];
const mcpItems = [
  ...mcpExamples,
  ...Array.from({ length: 65 }, (_, i) => ({ id: `git-tool-${i}`, name: `Git tool ${i}`, title: `Git repository helper ${i}`, description: "Manage source code repositories.", transport: "stdio", runs: "npx", inputs: [] })),
  ...Array.from({ length: 58 }, (_, i) => ({ id: `utility-${i}`, name: `Utility ${i}`, title: `General helper ${i}`, description: "A general-purpose utility.", transport: "stdio", runs: "npx", inputs: [] })),
];
const skillExamples = [
  { id: "acme/skills/playwright", name: "playwright", source: "acme/skills", skillId: "playwright", installs: 500, icon: "" },
  { id: "acme/skills/xlsx", name: "xlsx", source: "acme/skills", skillId: "xlsx", installs: 450, icon: "" },
  { id: "acme/skills/research", name: "research", source: "acme/skills", skillId: "research", installs: 400, icon: "" },
  { id: "acme/skills/docx", name: "docx", source: "acme/skills", skillId: "docx", installs: 350, icon: "" },
  { id: "acme/skills/ai-video-generation", name: "ai-video-generation", source: "acme/skills", skillId: "ai-video-generation", installs: 300, icon: "" },
  { id: "acme/skills/security-audit", name: "security-audit", source: "acme/skills", skillId: "security-audit", installs: 250, icon: "" },
  // Its description arrives later and says PostgreSQL; the card's category
  // remains stable for the current result set.
  { id: "acme/skills/mystery-skill", name: "mystery-skill", source: "acme/skills", skillId: "mystery-skill", installs: 200, icon: "" },
];
const skillItems = [
  ...skillExamples,
  ...Array.from({ length: 65 }, (_, i) => ({ id: `acme/skills/code-review-${i}`, name: `code-review-${i}`, source: "acme/skills", skillId: `code-review-${i}`, installs: 100 - i, icon: "" })),
  ...Array.from({ length: 58 }, (_, i) => ({ id: `acme/skills/helper-${i}`, name: `helper-${i}`, source: "acme/skills", skillId: `helper-${i}`, installs: 0, icon: "" })),
];
const catalogs = { mcp: mcpItems, skills: skillItems };
const categoryExample = { browse: "playwright", development: "git-tool-0", data: "postgres", knowledge: "memory", productivity: "calendar", media: "imagegen", security: "sentry", other: "mcp-utility" };
const skillCategoryExample = { browse: "acme/skills/playwright", development: "acme/skills/code-review-0", data: "acme/skills/xlsx", knowledge: "acme/skills/research", productivity: "acme/skills/docx", media: "acme/skills/ai-video-generation", security: "acme/skills/security-audit", other: "acme/skills/mystery-skill" };

function matches(item, query) {
  const text = [item.id, item.name, item.title, item.description, item.source, item.skillId].filter(Boolean).join(" ").toLowerCase();
  return text.includes(query.toLowerCase());
}
function serve(lang, requests, about) {
  return async (route) => {
    const req = route.request(), url = new URL(req.url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:${JSON.stringify(lang)},theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname === "/api/library") return json({
      dir: "/Users/tester/.jianmu/library", backups: "/Users/tester/.jianmu/backups", home: "/Users/tester",
      agents: [{ id: "claude", name: "Claude Code", icon: "claudecode-color", skills: "/Users/tester/.claude/skills", mcp: "/Users/tester/.claude.json" }],
      instructions: { agents: [], sets: [] }, servers: [], foundServers: [], projects: [], foundSkills: [], skills: [], problems: [],
    });
    const market = url.pathname.match(/^\/api\/library\/market\/(servers|skills)$/);
    if (market) {
      const kind = market[1] === "servers" ? "mcp" : "skills";
      const q = url.searchParams.get("q") || "";
      requests.push({ kind, q });
      if (kind === "mcp" && q.startsWith("https://")) return json({ items: [], custom: { name: "Example", transport: "http", url: q } });
      const items = catalogs[kind].filter((item) => !q || matches(item, q));
      return json({ items });
    }
    if (url.pathname === "/api/library/market/about") {
      const ids = req.postDataJSON().ids;
      await new Promise((resolve) => setTimeout(resolve, 60));
      const result = Object.fromEntries(ids.map((id) => [id, id === "acme/skills/mystery-skill" ? "A PostgreSQL database integration skill." : `Description for ${id}.`]));
      for (const [id, text] of Object.entries(result)) about[id] = text;
      return json(result);
    }
    if (url.pathname === "/api/library/icon") return route.fulfill({ status: 404, body: "" });
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname === "/api/providers") return json({ providers: [], gateway: { running: true } });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    const body = await fs.readFile(file).catch(() => null);
    await (body ? route.fulfill({ body, contentType }) : route.fulfill({ status: 404, body: "" }));
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of Object.keys(words)) for (const kind of ["mcp", "skills"]) {
    test(`${engine} ${lang} ${kind}: category filters compose with search and pagination`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      const context = await browser.newContext({ viewport: { width: 520, height: 760 }, reducedMotion: "reduce" });
      const page = await context.newPage();
      page.setDefaultTimeout(6000);
      page.setDefaultNavigationTimeout(60000);
      const errors = [], requests = [], about = {};
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve(lang, requests, about));
      await page.addInitScript((tab) => localStorage.setItem("jianmu.libTab", tab), kind === "mcp" ? "mcp" : "skills");
      t.after(async () => { await context.close(); await browser.close(); });
      await page.goto("http://jianmu.test/?view=library");
      const box = page.locator(`#view-library .mk[data-market="${kind}"]`);
      await box.waitFor();
      const buttons = box.locator(".mk-categories button[data-market-category]");
      await buttons.first().waitFor();
      assert.deepEqual(await buttons.evaluateAll((bs) => bs.map((b) => b.textContent.trim())), words[lang]);
      assert.equal(await box.locator('[data-market-category="all"]').getAttribute("aria-pressed"), "true");

      const examples = kind === "mcp" ? categoryExample : skillCategoryExample;
      const calls = () => requests.filter((r) => r.kind === kind).length;
      await page.waitForFunction((sel) => {
        const box = document.querySelector(sel);
        return box && !box.classList.contains("loading") && box.querySelector(".mk-card:not(.mk-skel)");
      }, `#view-library .mk[data-market="${kind}"]`);
      const beforeCategoryClick = calls();
      if (kind === "skills") {
        const mystery = box.locator('.mk-card[data-id="acme/skills/mystery-skill"]');
        await page.waitForFunction(() => document.querySelector('.mk-card[data-id="acme/skills/mystery-skill"] .mk-desc')?.textContent.includes("PostgreSQL"));
        await box.locator('[data-market-category="data"]').click();
        assert.equal(await mystery.count(), 0, "late descriptions must not move a fetched skill between categories");
        await box.locator('[data-market-category="other"]').click();
        assert.equal(await mystery.count(), 1);
        await box.locator('[data-market-category="all"]').click();
      }
      for (const category of categories.slice(1)) {
        const chip = box.locator(`[data-market-category="${category}"]`);
        await reach(page, chip);
        await chip.click();
        await page.waitForFunction(([market, id]) => document.querySelector(`.mk[data-market="${market}"] [data-market-category="${id}"]`)?.getAttribute("aria-pressed") === "true", [kind, category]);
        const ids = await box.locator(".mk-card").evaluateAll((cards) => cards.map((card) => card.getAttribute("data-id")));
        assert(ids.includes(examples[category]), `${kind} did not classify the ${category} example`);
        assert.equal(calls(), beforeCategoryClick, "category click must not request the market again");
        assert.equal(await chip.getAttribute("aria-pressed"), "true");
        assert.equal(await box.locator('[data-market-category="all"]').getAttribute("aria-pressed"), "false");
        if (category === "development") {
          assert.equal(ids.length, 60);
          assert.match(await box.locator(".mk-more").textContent(), /5/);
          const more = box.locator(".mk-more");
          await reach(page, more);
          await more.click();
          await page.waitForFunction((market) => document.querySelectorAll(`.mk[data-market="${market}"] .mk-card`).length === 65, kind);
          const allDev = await box.locator(".mk-card").evaluateAll((cards) => cards.map((card) => card.getAttribute("data-id")));
          assert(allDev.every((id) => id.includes(kind === "mcp" ? "git-tool-" : "code-review-")));
          assert.equal(calls(), beforeCategoryClick, "show more is also local to fetched results");
        }
      }

      // Text search and a category form an intersection; clearing the category
      // restores the same query results without another network request.
      await box.locator('[data-market-category="all"]').click();
      const search = box.locator(`input[data-lib="market-${kind}"]`);
      await search.fill(kind === "mcp" ? "postgres" : "mystery-skill");
      await search.press("Enter");
      await page.waitForFunction(([market, needle]) => {
        const b = document.querySelector(`.mk[data-market="${market}"]`);
        return b && !b.classList.contains("loading") && [...b.querySelectorAll(".mk-card")].some((c) => c.dataset.id.includes(needle));
      }, [kind, kind === "mcp" ? "postgres" : "mystery-skill"]);
      const queryCalls = calls();
      await box.locator('[data-market-category="security"]').click();
      await page.waitForFunction((market) => document.querySelector(`.mk[data-market="${market}"] .mk-msg.mk-category-empty`), kind);
      assert.equal(await box.locator(".mk-card").count(), 0);
      assert.equal(calls(), queryCalls, "filtering a text query must not request it again");
      await box.locator('[data-market-category="all"]').click();
      await page.waitForFunction((market) => document.querySelector(`.mk[data-market="${market}"] .mk-card`), kind);
      assert.equal(await box.locator(".mk-card").count(), 1);

      if (kind === "mcp") {
        await search.fill("https://example.com/mcp");
        await search.press("Enter");
        await page.waitForFunction(() => document.querySelector(".mk-custom"));
        assert.equal(await box.locator(".mk-custom").count(), 1, "category UI leaves the add-by-address fallback intact");
      } else {
        await box.locator('[data-market-category="data"]').click();
        await search.fill("");
        await search.press("Enter");
        await page.waitForFunction(() => document.querySelector('.mk[data-market="skills"] .mk-card[data-id="acme/skills/xlsx"]'));
        assert.equal(await box.locator('[data-market-category="data"]').getAttribute("aria-pressed"), "true", "clearing search preserves the category");
      }
      const chipRow = box.locator(".mk-categories");
      assert(await chipRow.evaluate((row) => row.scrollWidth <= row.clientWidth || getComputedStyle(row).flexWrap === "wrap"), "category chips must fit or wrap");
      assert.deepEqual(errors, []);
    });
  }
}
