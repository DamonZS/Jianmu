// A panel on Settings' Plugins tab: how many notes there are, and a way to
// clear them.
(async () => {
  await jianmu.ready;
  let opts = await jianmu.options.get();
  const paint = () => {
    const n = Object.values(opts.notes || {}).filter(Boolean).length;
    document.getElementById("count").textContent = n === 1 ? "1 note" : `${n} notes`;
  };
  paint();
  document.getElementById("clear").onclick = async () => {
    opts = await jianmu.options.set({ ...opts, notes: {} });
    paint();
  };
})();
