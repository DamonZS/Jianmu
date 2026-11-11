package gui

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// The Dock tile's pulse is the header logo's (assets/app.css,
// .logo.wag svg): two concentric rings have no tail to flick or wing to
// flap, so the mark swells and settles instead, over the same .9s,
// cubic-bezier(.3, 0, .3, 1) between each two keys. TestDockFlapIsTheLogos
// holds it to app.css.
const (
	flapSecs  = 0.9
	flapEvery = 0.03
)

// A keyframe: at is the fraction of the animation, size the mark's scale.
type flapKey struct{ at, size float64 }

var markKeys = []flapKey{{0, 1}, {.22, .90}, {.46, 1}, {.70, .93}, {.86, .98}, {1, 1}}

// markAt is the mark's scale at t, 0..1 of the pulse.
func markAt(t float64) float64 { return keyed(markKeys, t) }

// keyed is the scale at t of keyframes played as CSS plays them, each step
// eased by cubic-bezier(.3, 0, .3, 1).
func keyed(ks []flapKey, t float64) float64 {
	for i := 1; i < len(ks); i++ {
		if t <= ks[i].at {
			a, b := ks[i-1], ks[i]
			return a.size + (b.size-a.size)*flapEase((t-a.at)/(b.at-a.at))
		}
	}
	return ks[len(ks)-1].size
}

// flapEase is cubic-bezier(.3, 0, .3, 1): x found for the curve's parameter
// by bisection, and its y.
func flapEase(x float64) float64 {
	bez := func(u, p1, p2 float64) float64 { return 3*u*(1-u)*(1-u)*p1 + 3*u*u*(1-u)*p2 + u*u*u }
	lo, hi := 0.0, 1.0
	for range 40 {
		m := (lo + hi) / 2
		if bez(m, .3, .3) < x {
			lo = m
		} else {
			hi = m
		}
	}
	return bez((lo+hi)/2, 0, 1)
}

// A path for the Dock's drawing: ops are 'M', 'L', 'C' and 'Z', pts the
// points they take (one each, three for C), in the viewBox.
type flapPath struct {
	ops []byte
	pts []float64
}

// logoPaths reads the header logo's mark: the rings index.html draws
// under #bird, one path each, outer first. ok is false when none is
// there or one has a command it doesn't read.
func logoPaths(page string) (rings []flapPath, ok bool) {
	m := regexp.MustCompile(`(?s)<g id="bird">(.*?)</g>`).FindStringSubmatch(page)
	if m == nil {
		return nil, false
	}
	for _, d := range regexp.MustCompile(` d="([^"]+)"`).FindAllStringSubmatch(m[1], -1) {
		p, good := parseFlapPath(d[1])
		if !good {
			return nil, false
		}
		rings = append(rings, p)
	}
	return rings, len(rings) > 0
}

// parseFlapPath reads absolute M, L, H, V, C and Z, H and V written as L.
func parseFlapPath(s string) (flapPath, bool) {
	var p flapPath
	toks := regexp.MustCompile(`[MLHVCZ]|-?(?:\d+\.?\d*|\.\d+)`).FindAllString(s, -1)
	if strings.TrimSpace(regexp.MustCompile(`[MLHVCZ]|-?(?:\d+\.?\d*|\.\d+)|[\s,]`).ReplaceAllString(s, "")) != "" {
		return p, false
	}
	var cmd byte
	var x, y float64
	num := func(i *int) (float64, bool) {
		if *i >= len(toks) {
			return 0, false
		}
		v, err := strconv.ParseFloat(toks[*i], 64)
		*i++
		return v, err == nil && !math.IsNaN(v)
	}
	for i := 0; i < len(toks); {
		if c := toks[i][0]; c >= 'A' && c <= 'Z' {
			cmd = c
			i++
			if cmd == 'Z' {
				p.ops = append(p.ops, 'Z')
				continue
			}
		} else if cmd == 0 || cmd == 'Z' {
			return p, false
		}
		n := map[byte]int{'M': 2, 'L': 2, 'H': 1, 'V': 1, 'C': 6}[cmd]
		v := make([]float64, n)
		for j := range v {
			var good bool
			if v[j], good = num(&i); !good {
				return p, false
			}
		}
		switch cmd {
		case 'M', 'L':
			x, y = v[0], v[1]
			p.ops = append(p.ops, cmd)
			p.pts = append(p.pts, x, y)
			if cmd == 'M' {
				cmd = 'L' // more pairs after M are lines
			}
		case 'H', 'V':
			if cmd == 'H' {
				x = v[0]
			} else {
				y = v[0]
			}
			p.ops = append(p.ops, 'L')
			p.pts = append(p.pts, x, y)
		case 'C':
			x, y = v[4], v[5]
			p.ops = append(p.ops, 'C')
			p.pts = append(p.pts, v...)
		}
	}
	return p, len(p.ops) > 0
}
