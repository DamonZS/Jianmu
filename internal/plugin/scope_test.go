package plugin

import "testing"

// The community's packages are published under two scopes: an install made
// before the scope moved and one made after it are the same plugin, so a
// move, a listing and an update find either one.
func TestSamePackageAcrossTheCommunityScopes(t *testing.T) {
	old, moved := "@magpie-community/opencode-zed-auth", "@jianmu-plugins/opencode-zed-auth"
	if !SamePackage(old, moved) || !SamePackage(moved, old) {
		t.Fatalf("%s and %s are one plugin", old, moved)
	}
	for _, other := range []string{"opencode-zed-auth", "@example/opencode-zed-auth", "@jianmu-plugins/opencode-zed"} {
		if SamePackage(old, other) {
			t.Fatalf("%s and %s are not one plugin", old, other)
		}
	}
}

func TestOfficialCoversBothCommunityScopes(t *testing.T) {
	for _, pkg := range []string{"@magpie-community/opencode-zed-auth", "@jianmu-plugins/opencode-zed-auth"} {
		if !Official(pkg) {
			t.Fatalf("%s is the community's, which jianmu keeps up to date", pkg)
		}
	}
	if Official("@example/opencode-zed-auth") {
		t.Fatal("another scope's package is not jianmu's to update")
	}
}