const assert = require("node:assert/strict");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const { fixture } = require("./fixtures/caller-keys.cjs");
const { openPart } = require("./reader.cjs");

// Settings › About names and opens toporeduce.cn, removes the Discord invite,
// and keeps the GitHub entry, in every language and at the tray panel's width.
const words = {
  en: { website: "Website", community: "Community", github: "questions, ideas and feedback, on GitHub" },
  zh: { website: "官网", community: "社区", github: "提问、想法和反馈，可在 GitHub 上交流" },
  "zh-TW": { website: "官網", community: "社群", github: "提問、想法和反饋，可在 GitHub 上交流" },
  ja: { website: "公式サイト", community: "コミュニティ", github: "質問、アイデア、フィードバックは GitHub で" },
  de: { website: "Website", community: "Community", github: "Fragen, Ideen und Feedback auf GitHub" },
};

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of Object.keys(words)) {
    for (const width of [1100, 420]) {
      test(`${engine} ${lang} ${width}: About links the website`, async (t) => {
        const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
        t.after(() => browser.close());
        const page = await (await browser.newContext({ viewport: { width, height: 800 }, reducedMotion: "reduce" })).newPage();
        page.setDefaultTimeout(6000);
        page.setDefaultNavigationTimeout(60000);
        const errors = [];
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", fixture(lang, "light", []));
        await page.goto("http://jianmu.test/");
        await page.locator("#prefs").click();
        await openPart(page, "about");
        const row = page.locator("#about .row", { has: page.locator(".who", { hasText: words[lang].website }) });
        await row.waitFor();
        assert.equal(await row.locator("code").textContent(), "toporeduce.cn");
        const opened = page.waitForRequest((r) => r.url().includes("/api/open"));
        await row.locator("button.text").click();
        assert.equal(JSON.parse((await opened).postData()).url, "https://toporeduce.cn");
        const community = page.locator("#about .row", { has: page.locator(".who", { hasText: words[lang].community }) });
        assert.equal(await community.locator("button.discord").count(), 0);
        assert.equal(await community.locator("button.github").count(), 1);
        assert.equal(await community.locator(".sub").textContent(), words[lang].github);
        assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, "About wider than the window");
        assert.deepEqual(errors, []);
      });
    }
  }
}
