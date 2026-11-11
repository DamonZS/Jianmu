package cli

import "testing"

// jianmu web on loopback behind a reverse proxy (#1479) prints the proxy's
// link with the key when told the proxy's address, so nobody joins
// "/?k=…" onto it by hand: JIANMU_WEB_URL, else JIANMU_PUBLIC_URL with no
// port (one with a port isn't a proxy). None is printed without either.
func TestWebProxyLink(t *testing.T) {
	key := "/?k=0123456789abcdef"
	for _, c := range []struct{ env, want string }{
		{"https://jianmu.example.com", "https://jianmu.example.com/?k=0123456789abcdef"},
		{"https://jianmu.example.com/", "https://jianmu.example.com/?k=0123456789abcdef"},
		{"jianmu.example.com", "http://jianmu.example.com/?k=0123456789abcdef"},
		{"https://example.com/jianmu/", "https://example.com/jianmu/?k=0123456789abcdef"},
		{"http://192.168.1.20:3425", ""},
		{"", ""},
	} {
		t.Setenv("JIANMU_PUBLIC_URL", c.env)
		if got := proxyLink(key); got != c.want {
			t.Errorf("JIANMU_PUBLIC_URL=%q: %q, want %q", c.env, got, c.want)
		}
	}
	// the page's own proxy address, when the gateway's is elsewhere
	t.Setenv("JIANMU_PUBLIC_URL", "https://api.example.com")
	for _, c := range []struct{ env, want string }{
		{"https://jianmu.example.com/", "https://jianmu.example.com/?k=0123456789abcdef"},
		{"jianmu.example.com:8443", "https://jianmu.example.com:8443/?k=0123456789abcdef"},
		{"https://jianmu.example.com/?x=1", ""},
		{"ftp://jianmu.example.com", ""},
	} {
		t.Setenv("JIANMU_WEB_URL", c.env)
		if got := proxyLink(key); got != c.want {
			t.Errorf("JIANMU_WEB_URL=%q: %q, want %q", c.env, got, c.want)
		}
	}
}

// `jianmu web` opens the app's page when the app serves it on the port it
// would take, and starts its own on any other.
func TestWebSamePort(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"127.0.0.1:3430", "127.0.0.1:3430", true},
		{"0.0.0.0:3430", "127.0.0.1:3430", true},
		{"127.0.0.1:3430", "0.0.0.0:3431", false},
		{"", "127.0.0.1:3430", false},
		{"127.0.0.1:3430", "3430", false},
	} {
		if got := samePort(c.a, c.b); got != c.want {
			t.Errorf("samePort(%q, %q) = %v", c.a, c.b, got)
		}
	}
}
