const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const { fixture } = require("./fixtures/caller-keys.cjs");

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(`${engine}: the brand animation finishes and can play again`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || "chromium" }));
    t.after(() => browser.close());
    for (const lang of ["en", "zh", "zh-TW", "ja", "de"]) {
      for (const mode of ["window", "panel"]) {
        await t.test(`${lang} ${mode}: the ring finishes on its SVG`, async () => {
          const context = await browser.newContext({ reducedMotion: "no-preference" });
          try {
            const page = await context.newPage();
            page.setDefaultTimeout(6000);
            const errors = [];
            page.on("pageerror", (e) => errors.push(e.message));
            const base = fixture(lang, "light", []);
            await page.route("**/*", (route) => new URL(route.request().url()).host === "jianmu.test" ? base(route) : route.abort());
            await page.addInitScript(() => {
              // Windows hides the window brand; exercise its visible Mac layout.
              Object.defineProperty(navigator, "platform", { value: "MacIntel" });
              window.brandEnds = 0;
              document.addEventListener("animationend", (e) => {
                if (e.target.matches(".brand .logo svg") && e.animationName === "mark-pulse") window.brandEnds++;
              });
            });
            await page.goto(`http://jianmu.test/?mode=${mode}`);
            assert.equal(await page.locator(".brand .logo .tail").count(), 0);
            await page.waitForFunction(() => window.brandEnds === 1);
            assert.deepEqual(errors, [], "opening the ring logo must not throw");
            assert.equal(await page.locator(".brand .logo").evaluate((e) => e.classList.contains("wag")), false, "the opening animation clears wag");
            await page.locator(".brand").dispatchEvent("mouseenter");
            assert.equal(await page.locator(".brand .logo").evaluate((e) => e.classList.contains("wag")), true);
            await page.waitForFunction(() => window.brandEnds === 2);
            assert.equal(await page.locator(".brand .logo").evaluate((e) => e.classList.contains("wag")), false, "hover can complete another animation");
            assert.deepEqual(errors, []);
          } finally {
            await context.close();
          }
        });
      }
    }
    await t.test("the legacy bird still finishes on its tail", async () => {
      const context = await browser.newContext({ reducedMotion: "no-preference" });
      try {
        const page = await context.newPage();
        page.setDefaultTimeout(6000);
        const errors = [];
        page.on("pageerror", (e) => errors.push(e.message));
        const base = fixture("en", "light", []);
        await page.route("**/*", async (route) => {
          const url = new URL(route.request().url());
          if (url.host !== "jianmu.test") return route.abort();
          if (url.pathname !== "/") return base(route);
          const html = await fs.readFile(path.resolve(__dirname, "../assets/index.html"), "utf8");
          // Only the legacy animation target is needed; app.js stays intact.
          return route.fulfill({ contentType: "text/html", body: html.replace(/(<span class="logo"[^>]*>\s*<svg[^>]*>)/, '$1<g class="tail"></g>') });
        });
        await page.goto("http://jianmu.test/?mode=panel");
        assert.equal(await page.locator(".brand .logo .tail").count(), 1);
        await page.waitForFunction(() => document.querySelector(".brand .logo").classList.contains("wag"));
        const result = await page.evaluate(() => {
          const logo = document.querySelector(".brand .logo");
          const svg = logo.querySelector("svg"), tail = logo.querySelector(".tail");
          svg.dispatchEvent(new AnimationEvent("animationend", { bubbles: true }));
          const beforeTail = logo.classList.contains("wag");
          tail.dispatchEvent(new AnimationEvent("animationend", { bubbles: true }));
          const afterTail = logo.classList.contains("wag");
          logo.closest(".brand").dispatchEvent(new MouseEvent("mouseenter"));
          const again = logo.classList.contains("wag");
          tail.dispatchEvent(new AnimationEvent("animationend", { bubbles: true }));
          return { beforeTail, afterTail, again, finished: !logo.classList.contains("wag") };
        });
        assert.deepEqual(result, { beforeTail: true, afterTail: false, again: true, finished: true });
        assert.deepEqual(errors, []);
      } finally {
        await context.close();
      }
    });
  });
}
