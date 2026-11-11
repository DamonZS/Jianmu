package iconproxy

import (
	"net/url"
	"testing"
)

func TestURLCarriesThePictureWhole(t *testing.T) {
	t.Setenv("JIANMU_ICON_PROXY", "")
	pic := "https://github.com/magpie-community.png?size=96&x=a b#c"
	got := URL(pic)
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme+"://"+u.Host+u.Path != Base {
		t.Fatalf("URL(%q) = %q, not on %s", pic, got, Base)
	}
	if q := u.Query(); len(q) != 1 || q.Get("url") != pic {
		t.Fatalf("URL(%q) = %q: query %v", pic, got, q)
	}
}

func TestURLLeavesToporeduceAlone(t *testing.T) {
	t.Setenv("JIANMU_ICON_PROXY", "")
	for u, want := range map[string]string{
		"https://toporeduce.cn/partners/x.png":     "https://toporeduce.cn/partners/x.png",
		"https://toporeduce.cn.evil.example/x.png": Base + "?url=" + url.QueryEscape("https://toporeduce.cn.evil.example/x.png"),
		"https://evil.example/?toporeduce.cn":      Base + "?url=" + url.QueryEscape("https://evil.example/?toporeduce.cn"),
	} {
		if got := URL(u); got != want {
			t.Errorf("URL(%q) = %q, want %q", u, got, want)
		}
	}
}
