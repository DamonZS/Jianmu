package gui

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestDockFlapIsTheLogos holds the Dock tile's pulse to the header logo's:
// the same keyframes, length and easing as app.css plays, and the mark
// read from index.html.
func TestDockFlapIsTheLogos(t *testing.T) {
	css, err := assets.ReadFile("assets/app.css")
	if err != nil {
		t.Fatal(err)
	}
	body := regexp.MustCompile(`@keyframes mark-pulse \{\n((?:  .*\n)+\})`).FindSubmatch(css)
	if body == nil {
		t.Fatal("no @keyframes mark-pulse in app.css")
	}
	// a keyframe line names one or two fractions: "0%, 100% { … }", so the
	// page's keys are read as a set and the Dock's order checked on its own
	got := map[flapKey]bool{}
	n := 0
	for _, m := range regexp.MustCompile(`(?m)^  ([\d%, ]+) \{ transform: scale\((-?[\d.]+)\); \}$`).FindAllSubmatch(body[1], -1) {
		size, _ := strconv.ParseFloat(string(m[2]), 64)
		for _, f := range strings.Split(string(m[1]), ",") {
			at, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(f), "%"), 64)
			got[flapKey{at / 100, size}] = true
			n++
		}
	}
	if n != len(markKeys) {
		t.Errorf("app.css has %d keys, the Dock plays %d", n, len(markKeys))
	}
	for _, k := range markKeys {
		if !got[k] {
			t.Errorf("app.css has no key %v", k)
		}
	}
	for i := 1; i < len(markKeys); i++ {
		if markKeys[i].at < markKeys[i-1].at {
			t.Errorf("the Dock's keys are out of order at %d", i)
		}
	}
	origin := `.logo svg { display: block; overflow: visible; transform-box: view-box; transform-origin: 22px 22px; }`
	if !regexp.MustCompile(regexp.QuoteMeta(origin)).Match(css) {
		t.Errorf("app.css has no %q", origin)
	}
	anim := `.logo.wag svg { animation: mark-pulse .9s cubic-bezier(.3, 0, .3, 1); }`
	if !regexp.MustCompile(regexp.QuoteMeta(anim)).Match(css) {
		t.Errorf("app.css has no %q", anim)
	}
	if flapSecs != 0.9 {
		t.Errorf("the Dock plays %v s, app.css .9s", flapSecs)
	}
	for x, prev := 0.0, -1.0; x <= 1; x += .01 {
		if e := flapEase(x); e < prev || e < -1e-9 || e > 1+1e-9 {
			t.Fatalf("the easing at %v is %v, after %v", x, e, prev)
		} else {
			prev = e
		}
	}
	page, err := assets.ReadFile("assets/index.html")
	if err != nil {
		t.Fatal(err)
	}
	rings, ok := logoPaths(string(page))
	if !ok {
		t.Fatal("index.html's mark didn't parse")
	}
	if len(rings) != 2 {
		t.Fatalf("the mark read as %d paths, want the two rings", len(rings))
	}
	for i, r := range rings {
		if len(r.ops) < 4 || r.ops[0] != 'M' {
			t.Errorf("ring %d read as %d ops", i, len(r.ops))
		}
	}
	if _, ok := parseFlapPath("M1 2Q3 4 5 6"); ok {
		t.Error("a path with Q read as good")
	}
}
