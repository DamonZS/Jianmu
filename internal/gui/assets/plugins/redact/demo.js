// Insights › Masking's demo (client.js has its row): the masking gateway
// takes one request through. The request comes to it from the agent on
// this computer, the gateway reads it, sets the private values aside in
// its ledger (on this computer), and sends on what's left — placeholders.
// The model answers with placeholders, the gateway puts the values back,
// and the agent signs in on this computer.
//
// It is a pure SVG/CSS animation, no media. The stage is drawn at
// 1000×380 and scaled to the card's width; a light pulse runs the wire
// out and back, and the gateway's two rings (orange outside, cyan
// inside) say what it's doing. It plays while the row is open and on
// screen, once through, and rests on its last step; with reduced motion
// it starts there.

const XQD_END = 18;
const XQD_STEPS = [
  [0, "You send a request", "You ask the agent to sign in to github.com with a password. The request goes to the masking gateway first and hasn't left your computer yet."],
  [1.9, "She reads it through", "The gateway reads the request line by line for what shouldn't be sent: emails, passwords."],
  [3.5, "Tearing out what's private", "The real values are set aside in its ledger. The ledger stays on your computer."],
  [4.7, "Sticking on placeholders", "Placeholders fill the gaps. A value always gets the same one, so the model still understands."],
  [5.6, "Off to the provider", "The gateway sends what's left. The model gets only placeholders."],
  [9.2, "The answer calls the browser", "The model decides to call browser.login, and its arguments are placeholders too."],
  [13.2, "Putting the values back", "When the answer comes, the gateway consults its ledger and turns each placeholder back into its value."],
  [15.4, "Signing in on your computer", "The tool signs in with the real password on your computer. The password never left it."],
];
const XQD_MAIL = "demo@example.com", XQD_PASS = "hunter2-prod";
const XQD_PM = "{{EMAIL_r3tvb2ce}}", XQD_PP = "{{PASSWORD_k7q2m4xa}}";

let xqd = null;   // the demo, once its row has been opened

document.addEventListener("click", (e) => {
  if (e.target.closest?.("#mdRow")) maskDemoToggle();
});

function maskDemoToggle() {
  const row = $("#mdRow"), body = $("#mdBody");
  if (!row || !body) return;
  const open = row.getAttribute("aria-expanded") !== "true";
  row.setAttribute("aria-expanded", String(open));
  body.hidden = !open;
  // the kernel can draw the page again, and an old demo goes with it
  if (open && !xqd?.root.isConnected) xqd = maskDemoBuild(body);
  xqd?.wake();
}

// What a label says: data-t, so a change of language says it again.
const xqdSay = (cls, text) => { const s = el("span", cls, t(text)); s.dataset.t = ""; s.dataset.en = text; return s; };
const xqdIcon = {
  you: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><circle cx="12" cy="8" r="4"/><path d="M4 21c1.5-4 4.5-6 8-6s6.5 2 8 6"/></svg>',
  book: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 4h11a3 3 0 0 1 3 3v13H8a3 3 0 0 1-3-3z"/><path d="M5 17a3 3 0 0 1 3-3h11"/></svg>',
  lock: '<svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round"><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></svg>',
  check: (n) => `<svg width="${n}" height="${n}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12.5l4.5 4.5L19 7.5"/></svg>`,
  server: '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="4" width="16" height="7" rx="2"/><rect x="4" y="13" width="16" height="7" rx="2"/><path d="M8 7.5h.01M8 16.5h.01"/></svg>',
  pause: '<svg width="12" height="12" viewBox="0 0 12 12"><rect x="2" y="1.5" width="2.6" height="9" rx="1" fill="currentColor"/><rect x="7.4" y="1.5" width="2.6" height="9" rx="1" fill="currentColor"/></svg>',
  play: '<svg width="12" height="12" viewBox="0 0 12 12"><path d="M3 1.8v8.4a.6.6 0 0 0 .9.5l7-4.2a.6.6 0 0 0 0-1l-7-4a.6.6 0 0 0-.9.5z" fill="currentColor"/></svg>',
};

// the gateway's mark: 建木's two rings, orange outside, cyan inside; the
// dashed arc walks its ring while the demo moves
const RING = `<svg class="xq-ring" viewBox="0 0 100 100" aria-hidden="true"><circle cx="50" cy="50" r="44" fill="none" stroke="#ED9834" stroke-width="5"/><circle class="xq-ring-dash" cx="50" cy="50" r="33" fill="none" stroke="#66C2CE" stroke-width="4" stroke-linecap="round" stroke-dasharray="60 120"/><circle cx="50" cy="50" r="21" fill="none" stroke="#66C2CE" stroke-width="3" stroke-opacity=".5"/></svg>`;

// The stage is drawn at 1000×380 and scaled to the card's width.
function maskDemoStage() {
  const stage = el("div", "xq-stage");
  const at = (node, k) => { node.dataset.k = k; return node; };
  const box = (k, icon, title, sub) => {
    const b = at(el("div", "xq-box"), k), h = el("div", "xq-bh"), ic = el("div", "xq-ic" + (icon === "book" ? " xq-acc" : ""));
    ic.innerHTML = xqdIcon[icon];
    const words = el("div");
    words.append(xqdSay("xq-h", title));
    if (sub) words.append(xqdSay("xq-sub", sub));
    h.append(ic, words);
    b.append(h);
    return b;
  };
  const slot = (k, code, real) => {
    const s = at(el("div", "xq-slot xq-mv"), k);
    s.append(el("span", "xq-real xq-mv", real), el("span", "xq-code xq-mv", code));
    return s;
  };
  const mline = (label, code, real, k) => { const m = at(el("div", "xq-mline xq-mv"), k); m.append(xqdSay("xq-k", label), slot(k, code, real)); return m; };

  // left column: the request, then the ledger the gateway keeps on this
  // computer, with the two sitting against the wire
  const req = box("req", "you", "You tell the agent");
  const mjob = at(el("div", "xq-mline"), "mjob");
  mjob.append(xqdSay("xq-k", "The job"), xqdSay("", "Sign in to github.com"));
  req.append(mjob, mline("Account", XQD_PM, XQD_MAIL, "s0"), mline("Password", XQD_PP, XQD_PASS, "s1"));

  const ledger = box("ledger", "book", "Value ledger", "Kept on your computer only");
  ledger.append(at(xqdSay("xq-lempty", "Nothing in it yet"), "lempty"));
  // the entries follow the header, so a taller font moves them down with it
  const entries = el("div", "xq-llist");
  [[XQD_PM, XQD_MAIL], [XQD_PP, XQD_PASS]].forEach(([code, real], i) => {
    const r = at(el("div", "xq-lrow xq-mv"), "row" + i);
    r.append(el("span", "xq-lc xq-pillc", code), el("span", "xq-lv", real));
    entries.append(r);
  });
  ledger.append(entries);

  // the gateway between the two sides: 建木's two rings, its state's words
  const gw = at(el("div", "xq-gateway"), "gw");
  const ring = el("div", "xq-gw-ring");
  ring.innerHTML = RING;
  const cap = at(el("div", "xq-gwcap"), "gwcaps");
  ["Idle", "Scanning", "Held locally", "Restoring"].forEach((w, i) => { const s = xqdSay("", w); if (!i) s.classList.add("on"); cap.append(s); });
  gw.append(xqdSay("xq-h", "Masking gateway"), xqdSay("xq-sub", "Runs on your computer"), ring, cap);

  // right column: the model service, what it got and what it wrote
  const model = box("model", "server", "Model service", "Remote, not on your computer");
  const pin = at(el("div", "xq-psec xq-mv"), "pin");
  const pline = (...n) => { const p = el("div", "xq-pline"); p.append(...n); return p; };
  pin.append(xqdSay("xq-cap", "It received"), pline(xqdSay("", "Sign in to github.com")), pline(xqdSay("", "Account"), el("span", "xq-pillc", XQD_PM)), pline(xqdSay("", "Password"), el("span", "xq-pillc", XQD_PP)));
  const pout = at(el("div", "xq-psec xq-mv"), "pout"), think = at(el("div", "xq-think"), "think");
  think.append(el("i", "xq-mv"), el("i", "xq-mv"), el("i", "xq-mv"));
  const pcode = at(el("div", "xq-pcode xq-mv"), "pcode");
  pcode.append("browser.login(", el("div", "xq-ind", "username:"), el("span", "xq-pillc", XQD_PM), el("div", "xq-ind", "password:"), el("span", "xq-pillc", XQD_PP), el("div", "", ")"));
  pout.append(xqdSay("xq-cap", "It wrote"), think, pcode);
  const pnote = at(el("div", "xq-pnote xq-mv"), "pnote");
  pnote.innerHTML = xqdIcon.check(12);
  pnote.append(xqdSay("", "It saw only placeholders, never your password or email."));
  model.append(pin, pout, pnote);

  // the wire's two ends: the left side is on this computer, the right is not
  const laneL = at(el("div", "xq-lane xq-lane-l"), "laneL");
  laneL.append(xqdSay("", "Your computer"));
  const laneR = at(el("div", "xq-lane xq-lane-r"), "laneR");
  laneR.append(xqdSay("", "Model service"));
  // the divider between the two sides, with its own words and a small note
  const wire = at(el("div", "xq-wire"), "wire");
  wire.append(at(el("div", "xq-wire-lbl"), "lbl"), xqdSay("xq-wire-note", "Only placeholders cross the wire"));
  stage.append(laneL, laneR, wire, req, ledger, gw, model);
  // the light itself: a head and three beads, the pulse that runs the wire
  const mkPulse = (k, cls) => { const p = at(el("div", cls + " xq-mv"), k); p.append(el("i"), el("i"), el("i"), el("i")); return p; };
  stage.append(mkPulse("ps", "xq-pulse xq-pulse-out"), mkPulse("pr", "xq-pulse xq-pulse-in"));
  return stage;
}

function maskDemoBuild(body) {
  const root = el("div", "xqd"), vp = el("div", "xq-vp"), stage = maskDemoStage();
  vp.append(stage);
  const foot = el("div", "xq-foot"), capT = el("div", "xq-cap-t"), capB = el("div", "xq-cap-b");
  foot.append(capT, capB);
  const ctrl = el("div", "xq-ctrl"), play = el("button", "xq-play"), scrub = el("input", "xq-scrub");
  play.type = "button";
  scrub.type = "range"; scrub.min = "0"; scrub.max = "1000"; scrub.value = "0";
  ctrl.append(play, scrub);
  const steps = el("div", "xq-steps");
  root.append(vp, foot, ctrl, steps);
  body.append(root);

  const q = (k) => stage.querySelector(`[data-k="${k}"]`);
  const clamp = (v, a = 0, b = 1) => Math.min(b, Math.max(a, v));
  const seg = (t, a, b) => clamp((t - a) / (b - a));
  const lerp = (a, b, k) => a + (b - a) * k;
  const eo = (k) => 1 - Math.pow(1 - k, 3);
  const f2 = (v) => Math.round(v * 100) / 100;
  // style writes are skipped when the value is the same, so a frame
  // touches only what moved, and nothing per frame reads layout
  const css = (e, prop, v) => { const c = e._c || (e._c = {}); if (c[prop] === v) return; c[prop] = v; e.style[prop] = v; };
  const cls = (e, name, on) => { const c = e._c || (e._c = {}), k = "c:" + name; if (c[k] === on) return; c[k] = on; e.classList.toggle(name, on); };
  const op = (e, v) => css(e, "opacity", String(f2(v)));

  const slots = {};
  for (const k of ["s0", "s1"]) {
    const m = q(k);
    slots[k] = { m, real: m.querySelector(".xq-real"), code: m.querySelector(".xq-code") };
  }
  const rows = [q("row0"), q("row1")], rowCodes = rows.map((r) => r.querySelector(".xq-lc"));
  const e = {};
  for (const k of ["pin", "pout", "think", "pcode", "pnote", "lempty", "ps", "pr", "gwcaps", "wire"]) e[k] = q(k);
  const thinkDots = [...q("think").children];
  const gwCaps = [...e.gwcaps.children];

  // the pulses' lane, in the stage's 1000 space: off the left edge of the
  // gateway to the model's edge, and back, with a small lift over the wire
  const PULSE = { out: [352, 700], in: [700, 352] };
  const drawPulse = (n, p, t0, a, b) => {
    const k = seg(t0, a, b);
    const vis = k > 0 && k < 1 ? seg(k, 0, .1) * (1 - seg(k, .9, 1)) : 0;
    if (vis <= 0) return op(n, 0);
    const x = lerp(p[0], p[1], k), y = 210 - 16 * Math.sin(Math.PI * k);
    css(n, "transform", `translate(${f2(x)}px,${f2(y)}px)`);
    op(n, vis);
    // the trail: three beads behind the head, each a little back on the lane
    [...n.children].forEach((d, i) => {
      const k2 = k - (i + 1) * .06;
      css(d, "transform", `translate(${f2(lerp(p[0], p[1], clamp(k2)))}px,${f2(210 - 16 * Math.sin(Math.PI * clamp(k2)))}px)`);
      op(d, k2 > 0 ? (1 - i / 3.4) : 0);
    });
  };

  const stepBtns = XQD_STEPS.map(([at0, title]) => {
    const b = el("button", "", t(title));
    b.type = "button"; b.dataset.t = ""; b.dataset.en = title;
    b.onclick = () => { seek(at0 + .01); render(T); sync(); };
    steps.append(b);
    return b;
  });

  let lastSi = -1, lastCap = -1;

  function render(t0) {
    const lang = document.documentElement.lang;

    // step 1: the request, then its two values
    op(q("mjob"), seg(t0, .3, .6));
    op(q("s0"), seg(t0, .9, 1.15));
    op(q("s1"), seg(t0, 1.15, 1.4));
    cls(q("s0"), "read", t0 >= 1.9 && t0 < 3.3);
    cls(q("s1"), "read", t0 >= 2.3 && t0 < 3.7);
    // steps 3–4: the values come out of the request into the ledger,
    // placeholders stand in where they were, and stand down when their
    // value is back on the answer
    const RECV = [3.55, 3.85], STICK = [4.7, 5.05], BACK = [14.0, 14.55];
    ["s0", "s1"].forEach((k, i) => {
      const s = slots[k];
      const gone = eo(seg(t0, RECV[i], RECV[i] + .3));
      const stuck = eo(seg(t0, STICK[i], STICK[i] + .3));
      const back = eo(seg(t0, BACK[i], BACK[i] + .35));
      op(s.real, Math.max(1 - gone, back));
      cls(s.real, "ok", t0 >= BACK[i] + .35);
      op(s.code, stuck * (1 - back));
      css(s.code, "transform", `scale(${f2(lerp(1.25, 1, stuck))})`);
      op(rows[i], eo(seg(t0, RECV[i] + .25, RECV[i] + .55)));
      op(rowCodes[i], seg(t0, STICK[i] - .15, STICK[i] + .25));
      cls(rows[i], "hot", t0 >= BACK[i] && t0 < BACK[i] + 1.0);
    });
    op(e.lempty, 1 - seg(t0, RECV[0] + .5, RECV[0] + .7));

    // step 5: out over the wire; the model receives only placeholders
    drawPulse(e.ps, PULSE.out, t0, 5.7, 6.7);
    op(e.pin, seg(t0, 6.4, 6.8));
    cls(e.wire, "live", t0 >= 5.55 && t0 < 7.0);
    cls(e.wire, "live-in", t0 >= 13.05 && t0 < 14.5);

    // step 6: the model thinks and writes its answer, placeholders too
    op(e.pout, seg(t0, 9.4, 9.7));
    css(e.think, "display", t0 < 10.4 ? "flex" : "none");
    if (t0 >= 9.4 && t0 < 10.4) thinkDots.forEach((d, i) => op(d, .35 + .65 * Math.max(0, Math.sin(t0 * 7 - i * .9))));
    op(e.pcode, seg(t0, 10.4, 10.9));
    op(e.pnote, seg(t0, 11.2, 11.6));

    // step 7: the answer comes back over the same wire, the placeholders
    // turn into their values, and the values go back to the request
    drawPulse(e.pr, PULSE.in, t0, 13.3, 14.3);

    // the gateway's own words, with what it's doing
    const capIdx = t0 < 1.9 ? 0 : t0 < 3.5 ? 1 : t0 < 13.2 ? 2 : 3;
    if (capIdx !== lastCap) {
      lastCap = capIdx;
      gwCaps.forEach((c, i) => cls(c, "on", i === capIdx));
    }
    const ringMove = (t0 >= 5.55 && t0 < 7.0) || (t0 >= 13.05 && t0 < 14.5);
    if (root._ringOp !== (ringMove ? "1" : ".55")) {
      root._ringOp = ringMove ? "1" : ".55";
      root.style.setProperty("--xq-ring-op", ringMove ? "1" : ".55");
    }
    cls(stage, "xq-moving", ringMove);

    let si = 0;
    XQD_STEPS.forEach(([s], i) => { if (t0 >= s) si = i; });
    if (si !== lastSi || lang !== document.documentElement.lang) {
      lastSi = si;
      const n = el("span", "n", `${si + 1}/${XQD_STEPS.length}`);
      capT.replaceChildren(n, t(XQD_STEPS[si][1]));
      capB.textContent = t(XQD_STEPS[si][2]);
      stepBtns.forEach((b, i) => { b.classList.toggle("on", i === si); b.classList.toggle("done", i < si); });
    }
  }

  // T, the frame clock, runs the animation; there is no media to keep in
  // step with, the clock alone is the whole of it
  let T = 0, playing = !calm(), open = false, seen = false, raf = 0, last = 0;
  const seek = (t0) => { T = t0; };
  let lastScrub = -1;
  const sync = () => { const v = Math.round(T / XQD_END * 1000); if (v !== lastScrub) { lastScrub = v; scrub.value = v; } };
  const live = () => open && seen && !document.hidden && root.isConnected && !$("#maskDemo")?.hidden;
  const setPlaying = (p) => {
    playing = p;
    play.innerHTML = p ? xqdIcon.pause : xqdIcon.play;
    words();
    if (p && T >= XQD_END - .05) seek(0);
    wake();
  };
  // the controls' names, said again when the language changes
  const words = () => {
    play.title = t(playing ? "Pause the demo" : "Play the demo");
    play.setAttribute("aria-label", play.title);
    scrub.setAttribute("aria-label", t("Where the demo is"));
  };
  play.onclick = () => setPlaying(!playing);
  scrub.oninput = () => { setPlaying(false); seek(scrub.value / 1000 * XQD_END); render(T); };

  function frame(now) {
    raf = 0;
    if (!live() || !playing) return;
    T += Math.min(.05, (now - (last || now)) / 1000);
    last = now;
    // once through, then it rests on the last step
    if (T >= XQD_END) { T = XQD_END; render(T); sync(); return setPlaying(false); }
    render(T); sync();
    raf = requestAnimationFrame(frame);
  }
  // wake starts or stops it as the row, the page and the window say
  function wake() {
    open = $("#mdRow")?.getAttribute("aria-expanded") === "true";
    if (live() && playing) {
      if (!raf) { last = 0; raf = requestAnimationFrame(frame); }
    } else {
      if (raf) cancelAnimationFrame(raf), (raf = 0);
    }
  }
  new IntersectionObserver((es) => { seen = es[es.length - 1].isIntersecting; wake(); }).observe(vp);
  document.addEventListener("visibilitychange", wake);
  new ResizeObserver(() => { stage.style.transform = `scale(${vp.clientWidth / 1000})`; }).observe(vp);
  // a change of language redraws the words the frame writes itself
  new MutationObserver(() => { lastSi = lastCap = -1; render(T); words(); }).observe(document.documentElement, { attributeFilter: ["lang"] });

  if (!playing) seek(17.2);
  render(T); sync();
  setPlaying(playing);
  root.__state = () => ({ T, playing, raf: !!raf });
  return { wake, root };
}
