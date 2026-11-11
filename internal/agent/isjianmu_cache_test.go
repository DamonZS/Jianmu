package agent

import (
	"testing"

	"github.com/DamonZS/Jianmu/internal/provider"
)

// isJianmu answers again from what it kept while nothing the catalog is
// built from changed: the window's drift poll asked it for every agent's
// values every 15s, outside any request holding the catalog, and each
// answer built every provider's entries again (~6 MB a poll). A provider
// saved since is seen at once.
func TestIsJianmuKeptWhileCatalogStays(t *testing.T) {
	syncHome(t)
	if err := provider.Save(provider.Provider{ID: "oa", Name: "OA", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"gpt-5.5"}}); err != nil {
		t.Fatal(err)
	}
	if !isJianmu("oa/gpt-5.5") {
		t.Fatal("oa/gpt-5.5 not jianmu's")
	}
	if isJianmu("nb/x") {
		t.Fatal("nb/x jianmu's before it is added")
	}
	allocs := testing.AllocsPerRun(20, func() {
		isJianmu("oa/gpt-5.5")
		isJianmu("nb/x")
	})
	// StampNow's look at three files; a catalog built was thousands
	if allocs > 200 {
		t.Fatalf("%.0f allocations for two answers already given", allocs)
	}

	if err := provider.Save(provider.Provider{ID: "nb", Name: "NB", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	if !isJianmu("nb/x") {
		t.Fatal("a provider saved since not seen")
	}
}
