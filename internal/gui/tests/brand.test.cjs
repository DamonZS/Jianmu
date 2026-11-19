const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const { test } = require("node:test");

const assets = process.env.JIANMU_ASSETS || path.resolve(__dirname, "../assets");
const source = fs.readFileSync(path.join(assets, "i18n.js"), "utf8");

function translations(pref, system = "en-US", query = "") {
  const context = vm.createContext({
    params: new URLSearchParams(query),
    navigator: { language: system },
    document: { documentElement: {}, querySelectorAll: () => [] },
    window: {},
  });
  vm.runInContext(source, context);
  vm.runInContext(`setLocale(${JSON.stringify(pref)})`, context);
  return (s, vars) => {
    context.input = s;
    context.vars = vars;
    return vm.runInContext("t(input, vars)", context);
  };
}

for (const lang of ["zh", "zh-TW", "en", "ja", "de"]) {
  test(`${lang}: only product labels are localized`, () => {
    const t = translations(lang);
    assert.equal(t("jianmu"), lang.startsWith("zh") ? "建木" : "jianmu");
    if (lang.startsWith("zh")) assert.equal(t("Quit jianmu"), lang === "zh" ? "退出 建木" : "結束 建木");
    for (const value of ["jianmu", "JIANMU", "jianmu web", "jianmu --help", "/home/jianmu", "C:\\jianmu", "https://jianmu.test/jianmu", "sk-jianmu-test", "@jianmu/plugin", "{jianmu}", "my jianmu 工作区"]) {
      assert.equal(t("{value}", { value }), value, "a substituted value is never renamed");
      assert.equal(t(value === "jianmu" ? `User says ${value}` : value), value === "jianmu" ? `User says ${value}` : value, "unknown text is never rewritten");
      assert.ok(t("Found by jianmu: {url}", { url: value }).endsWith(value), "a real translated template preserves the supplied value");
    }
    const command = t("{host} doesn’t answer as a jianmu page: check the address and port (jianmu web’s is 3430)", { host: "jianmu" });
    assert.ok(command.startsWith("jianmu"));
    assert.ok(command.includes("jianmu web"));
    if (lang.startsWith("zh")) assert.ok(command.includes("建木"));
    assert.ok(t("The encrypted backup goes in <folder>/jianmu/jianmu.jianmu-backup. Each update creates a commit.").includes("<folder>/jianmu/jianmu.jianmu-backup"));
    assert.ok(t("Switch an agent on and the models you set up in jianmu show up as the {jianmu} provider in its own model list; or pick its model right here.").includes("{jianmu}"));
  });
}

test("saved language and URL choice win over the system language", () => {
  assert.equal(translations("en", "zh-CN")("jianmu"), "jianmu");
  assert.equal(translations("ja", "zh-TW")("jianmu"), "jianmu");
  assert.equal(translations("system", "zh-Hant-HK")("jianmu"), "建木");
  assert.equal(translations("zh", "zh-CN", "locale=en")("jianmu"), "jianmu");
  assert.equal(translations("en", "en-US", "locale=zh-TW")("jianmu"), "建木");
});

test("static branding follows language changes without overwriting its English source", () => {
  const html = fs.readFileSync(path.join(assets, "index.html"), "utf8");
  const title = html.match(/<title data-t>([^<]+)<\/title>/);
  const brand = html.match(/<span data-t>(jianmu)<\/span>/);
  const meta = html.match(/<meta name="apple-mobile-web-app-title" content="([^"]+)" data-tc="([^"]+)">/);
  assert.ok(title && brand && meta, "the real page marks its product labels for translation");
  const text = [title, brand].map((m) => ({ textContent: m[1], dataset: {} }));
  const content = { content: meta[1], dataset: { tc: meta[2] } };
  const context = vm.createContext({
    params: new URLSearchParams(), navigator: { language: "zh-CN" }, window: {},
    document: { documentElement: {}, querySelectorAll: (selector) => selector === "[data-t]" ? text : selector === "[data-tc]" ? [content] : [] },
  });
  vm.runInContext(source, context);
  for (const lang of ["zh", "en", "zh-TW", "ja", "de", "zh"]) {
    vm.runInContext(`setLocale(${JSON.stringify(lang)})`, context);
    const want = lang.startsWith("zh") ? "建木" : "jianmu";
    assert.deepEqual(text.map((e) => e.textContent), [want, want]);
    assert.equal(content.content, want);
  }
});

test("the static install manifest keeps the language-neutral name", () => {
  const manifest = JSON.parse(fs.readFileSync(path.join(assets, "manifest.json"), "utf8"));
  assert.equal(manifest.name, "jianmu");
  assert.equal(manifest.short_name, "jianmu");
});
