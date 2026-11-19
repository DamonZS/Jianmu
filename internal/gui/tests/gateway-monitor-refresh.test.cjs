const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { test } = require('node:test');

const source = fs.readFileSync(path.join(__dirname, '../assets/plugins/gateway/client.js'), 'utf8');
function fixture() {
  const timers = new Map(), reads = [];
  let serial = 0, renders = 0;
  const page = { hidden: false };
  const ctx = vm.createContext({
    URLSearchParams, AbortController, location: { search: '' }, navigator: { platform: 'Win32' },
    localStorage: { getItem: () => null }, document: { hidden: false },
    jianmu: { plugin() {} }, view: 'gateway', providers: { gateway: { calls: [], mine: false } },
    $: () => page,
    setTimeout(fn, delay) { const id = ++serial; timers.set(id, { fn, delay }); return id; },
    clearTimeout(id) { timers.delete(id); },
    api(url, body, heads, signal) {
      return new Promise((resolve, reject) => reads.push({ url, signal, resolve, reject }));
    },
    rendered() { renders++; },
  });
  vm.runInContext(source, ctx);
  // Rendering has separate browser coverage; this test drives the actual request lifecycle.
  vm.runInContext('renderActivity = rendered; renderGatewayCallsNote = () => {};', ctx);
  return { ctx, page, reads, timers, renders: () => renders,
    run: (code) => vm.runInContext(code, ctx) };
}

test('recent calls refresh uses only metadata and does not repaint unchanged rows', async () => {
  const f = fixture();
  const pending = f.run('refreshGatewayCalls()');
  assert.equal(f.reads[0].url, 'gateway/recent');
  await f.run('refreshGatewayCalls()');
  assert.equal(f.reads.length, 1);
  f.reads[0].resolve({ calls: [{ id: 1 }], mine: true });
  await pending;
  assert.equal(f.renders(), 1);
  assert.equal(f.ctx.providers.gateway.mine, false, 'peer metadata does not claim local gateway ownership');
  assert.equal([...f.timers.values()][0].delay, 2000);
  const next = f.run('refreshGatewayCalls()');
  f.reads[1].resolve({ calls: [{ id: 1 }] });
  await next;
  assert.equal(f.renders(), 1);
});

test('leaving cancels requests and a stale completion cannot overwrite a new visit', async () => {
  const f = fixture();
  const old = f.run('refreshGatewayCalls()');
  f.run('stopGatewayCalls()');
  assert.equal(f.reads[0].signal.aborted, true);
  assert.equal(f.timers.size, 0);
  await f.run('refreshGatewayCalls()');
  assert.equal(f.reads.length, 1, 'a new visit waits for the aborted read to settle');
  f.reads[0].resolve({ calls: [{ id: 1 }] });
  await old;
  assert.equal(f.renders(), 0, 'the stale answer was discarded');
  const [timerId, timer] = [...f.timers.entries()][0];
  f.timers.delete(timerId);
  const current = timer.fn();
  assert.equal(f.reads.length, 2);
  f.reads[1].resolve({ calls: [{ id: 2 }] });
  await current;
  assert.equal(f.ctx.providers.gateway.calls[0].id, 2);
  f.ctx.document.hidden = true;
  await f.run('refreshGatewayCalls()');
  assert.equal(f.reads.length, 2);
  assert.equal(f.timers.size, 0);
});

test('failed metadata reads retain previous rows and report failure', async () => {
  const f = fixture();
  f.ctx.providers.gateway.calls = [{ id: 7 }];
  const pending = f.run('refreshGatewayCalls()');
  f.reads[0].reject(new Error('unavailable'));
  await pending;
  assert.equal(f.ctx.providers.gateway.calls[0].id, 7);
  assert.equal(f.run('gatewayCallsFailed'), true);
  assert.equal(f.renders(), 0);
});
