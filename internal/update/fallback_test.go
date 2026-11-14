package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// fallbackWorld is a fake GitHub with a signed release and a feed host
// that refuses every connection, the way a company allow list that lets
// only github.com through does (ssharpcc on X).
type fallbackWorld struct {
	srv           *httptest.Server
	body          []byte
	asset         string
	limited       bool // the API answers 403, as when anonymous asks run out
	unsigned      bool // current Jianmu releases publish checksums without a signature
	apiSigLimited bool // the API denies the signature asset even if metadata lists it
	sums, sig     string
	feedHits      atomic.Int32
	apiHits       atomic.Int32
	dlHits        atomic.Int32
}

func newFallbackWorld(t *testing.T, version string) *fallbackWorld {
	t.Helper()
	w := &fallbackWorld{body: []byte("the new jianmu"), asset: BinaryAsset()}
	h := sha256.Sum256(w.body)
	w.sums = hex.EncodeToString(h[:]) + "  " + w.asset + "\n" + strings.Repeat("ab", 32) + "  other-file\n"
	pub, priv, _ := ed25519.GenerateKey(nil)
	old := signingKey
	signingKey = base64.StdEncoding.EncodeToString(pub)
	t.Cleanup(func() { signingKey = old })
	w.sig = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte("jianmu release "+version+"\n"+w.sums)))

	tag := "v" + version
	const publishedRepo = "DamonZS/Jianmu"
	dl := "/" + publishedRepo + "/releases/download/" + tag + "/"
	w.srv = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/" + publishedRepo + "/releases/latest":
			w.apiHits.Add(1)
			if w.limited {
				http.Error(rw, "rate limit exceeded", 403)
				return
			}
			fmt.Fprintf(rw, `{"tag_name":%q,"body":"notes here","html_url":"https://github.com/%s/releases/tag/%s","assets":[{"name":%q,"size":%d,"url":%q},{"name":"SHA256SUMS.sig","size":%d,"url":%q}]}`,
				tag, publishedRepo, tag, w.asset, len(w.body), w.srv.URL+"/repos/"+publishedRepo+"/releases/assets/1", len(w.sig), w.srv.URL+"/repos/"+publishedRepo+"/releases/assets/2")
		case "/" + publishedRepo + "/releases/latest":
			http.Redirect(rw, r, "/"+publishedRepo+"/releases/tag/"+tag, http.StatusFound)
		case dl + "SHA256SUMS":
			rw.Write([]byte(w.sums))
		case "/repos/" + publishedRepo + "/releases/assets/2":
			if w.unsigned {
				http.NotFound(rw, r)
				return
			}
			if r.Header.Get("Accept") == "application/octet-stream" {
				if w.apiSigLimited {
					http.Error(rw, "rate limit exceeded", http.StatusForbidden)
					return
				}
				rw.Write([]byte(w.sig + "\n"))
				return
			}
			http.NotFound(rw, r)
		case dl + "SHA256SUMS.sig":
			if w.unsigned || w.apiSigLimited {
				http.NotFound(rw, r)
				return
			}
			rw.Write([]byte(w.sig + "\n"))
		case dl + w.asset:
			w.dlHits.Add(1)
			rw.Write(w.body)
		default:
			http.NotFound(rw, r)
		}
	}))
	t.Cleanup(w.srv.Close)

	oldWeb, oldAPI, oldFeed := githubWeb, githubAPI, siteFeed
	githubWeb, githubAPI = w.srv.URL, w.srv.URL
	// The feed closes the connection without a response, as in the reported EOF.
	feed := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		w.feedHits.Add(1)
		conn, _, err := rw.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		conn.Close()
	}))
	t.Cleanup(feed.Close)
	siteFeed = feed.URL + "/api/latest"
	t.Cleanup(func() { githubWeb, githubAPI, siteFeed = oldWeb, oldAPI, oldFeed })
	t.Setenv("JIANMU_UPDATE_FEED", "")
	stableFor(t, "")
	quickRetries(t)
	return w
}

// The feed host refuses connections: the release is read from GitHub,
// checked against the embedded key, and its file downloaded from GitHub
// and installed by hash.
func TestUpdateWorksFromGitHubWhenTheFeedIsUnreachable(t *testing.T) {
	for _, limited := range []bool{false, true} {
		t.Run(fmt.Sprintf("api limited %v", limited), func(t *testing.T) {
			w := newFallbackWorld(t, "0.1.4000")
			w.limited = limited
			rel, err := Latest(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if rel.Version != "0.1.4000" || rel.Signed == nil {
				t.Fatalf("release %+v", rel)
			}
			if !limited && rel.Notes != "notes here" {
				t.Fatalf("notes %q", rel.Notes)
			}
			if _, ok := rel.Assets["other-file"]; !ok {
				t.Fatal("the signed list's files are all offered")
			}
			exe := filepath.Join(t.TempDir(), "jianmu")
			staged, err := stageBinaryFor(context.Background(), rel.Assets[w.asset], exe)
			if err != nil {
				t.Fatal(err)
			}
			if b, _ := os.ReadFile(staged); string(b) != string(w.body) {
				t.Fatalf("staged %q", b)
			}
			if w.dlHits.Load() != 1 {
				t.Fatalf("file asked %d times", w.dlHits.Load())
			}
		})
	}
}

// Jianmu's existing releases have SHA256SUMS but no signature. A failed
// feed and an API limit still leave the Windows download available by hash.
func TestFeedVerificationFailureDoesNotFallBack(t *testing.T) {
	w := newFallbackWorld(t, "0.1.12")
	good, err := Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	bad := *good
	bad.Assets = make(map[string]Asset, len(good.Assets))
	for name, asset := range good.Assets {
		asset.SHA256 = strings.Repeat("00", sha256.Size)
		bad.Assets[name] = asset
	}
	feed := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(rw).Encode(&bad)
	}))
	defer feed.Close()
	oldFeed := siteFeed
	siteFeed = feed.URL
	defer func() { siteFeed = oldFeed }()
	if _, err := Latest(context.Background()); err == nil || !errors.Is(err, errFeedVerification) {
		t.Fatalf("a bad feed checksum fell back to GitHub: %v", err)
	}
	if w.apiHits.Load() != 1 {
		t.Fatalf("GitHub was queried after a corrupt feed; API hits %d", w.apiHits.Load())
	}
}

func TestJianmuReleaseUpdatesWithoutTheFeed(t *testing.T) {
	for _, limited := range []bool{false, true} {
		t.Run(fmt.Sprintf("api limited %v", limited), func(t *testing.T) {
			wasGUI := GUI
			GUI = true
			t.Cleanup(func() { GUI = wasGUI })
			w := newFallbackWorld(t, "0.1.12")
			w.limited, w.unsigned = limited, true
			rel, err := LatestIn(context.Background(), "zh")
			if err != nil {
				t.Fatal(err)
			}
			if rel.Version != "0.1.12" || rel.Signed != nil {
				t.Fatalf("release %+v", rel)
			}
			asset, ok := rel.Assets[w.asset]
			if !ok || !strings.Contains(asset.URL, "/DamonZS/Jianmu/releases/download/v0.1.12/") {
				t.Fatalf("Windows update asset: %+v", rel.Assets)
			}
			exe := filepath.Join(t.TempDir(), "jianmu-windows-x64.exe")
			staged, err := stageBinaryFor(context.Background(), asset, exe)
			if err != nil {
				t.Fatal(err)
			}
			if b, _ := os.ReadFile(staged); string(b) != string(w.body) {
				t.Fatalf("staged %q", b)
			}
		})
	}
}

func TestGitHubFallbackStillRequiresSignatureAndVerifiedHash(t *testing.T) {
	t.Run("signature required after cutoff", func(t *testing.T) {
		w := newFallbackWorld(t, firstSigned)
		w.unsigned = true
		if _, err := Latest(context.Background()); err == nil || !strings.Contains(err.Error(), "has no signature") {
			t.Fatalf("missing signature: %v", err)
		}
	})
	t.Run("raw missing plus API forbidden is not missing", func(t *testing.T) {
		w := newFallbackWorld(t, "0.1.12")
		w.apiSigLimited = true
		api := w.srv.URL + "/repos/DamonZS/Jianmu/releases/assets/2"
		if _, err := githubFile(context.Background(), w.srv.URL+"/raw-signature", api); err == nil || errors.Is(err, errGitHubFileMissing) {
			t.Fatalf("raw 404 plus API 403 was treated as a missing file: %v", err)
		}
	})
	t.Run("all signature URLs missing is recognized", func(t *testing.T) {
		w := newFallbackWorld(t, "0.1.12")
		if _, err := githubFile(context.Background(), w.srv.URL+"/raw-signature", ""); err == nil || !errors.Is(err, errGitHubFileMissing) {
			t.Fatalf("raw 404 with no API URL was not recognized as missing: %v", err)
		}
	})
	t.Run("release page loc works without asset API metadata", func(t *testing.T) {
		wasGUI := GUI
		GUI = true
		t.Cleanup(func() { GUI = wasGUI })
		w := newFallbackWorld(t, "0.1.12")
		w.limited, w.unsigned = true, true
		rel, err := Latest(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := rel.Assets[w.asset]; !ok {
			t.Fatalf("assets from SHA256SUMS: %+v", rel.Assets)
		}
	})
	t.Run("bad signature is not treated as missing", func(t *testing.T) {
		w := newFallbackWorld(t, "0.1.12")
		w.sig = "not base64"
		if _, err := Latest(context.Background()); err == nil || !strings.Contains(err.Error(), "signature") {
			t.Fatalf("bad signature: %v", err)
		}
	})
	t.Run("bad unsigned file hash is not staged", func(t *testing.T) {
		wasGUI := GUI
		GUI = true
		t.Cleanup(func() { GUI = wasGUI })
		w := newFallbackWorld(t, "0.1.12")
		w.unsigned = true
		w.sums = strings.Repeat("00", 32) + "  " + w.asset + "\n"
		rel, err := Latest(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := stageBinaryFor(context.Background(), rel.Assets[w.asset], filepath.Join(t.TempDir(), w.asset)); err == nil {
			t.Fatal("staged a file whose bytes do not match SHA256SUMS")
		}
	})
}

// A release GitHub gives with a signature that isn't the key's is refused,
// on the API path and on the redirect path alike.
func TestGitHubFallbackRefusesABadSignature(t *testing.T) {
	for _, limited := range []bool{false, true} {
		for _, bad := range []string{"wrong key", "other sums", "no sig", "junk"} {
			t.Run(fmt.Sprintf("limited %v %s", limited, bad), func(t *testing.T) {
				w := newFallbackWorld(t, "0.1.4000")
				w.limited = limited
				switch bad {
				case "wrong key":
					_, priv, _ := ed25519.GenerateKey(nil)
					w.sig = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte("jianmu release 0.1.4000\n"+w.sums)))
				case "other sums": // a swapped file with a hash that matches it
					h := sha256.Sum256([]byte("evil"))
					w.sums = hex.EncodeToString(h[:]) + "  " + w.asset + "\n"
				case "no sig":
					w.sig = ""
				case "junk":
					w.sig = "not base64!!"
				}
				if rel, err := Latest(context.Background()); err == nil {
					t.Fatalf("accepted %+v", rel)
				}
				if w.dlHits.Load() != 0 {
					t.Fatal("a file was downloaded")
				}
			})
		}
	}
}

// A release signed for another version can't be passed off as the tag's.
func TestGitHubFallbackBindsTheVersion(t *testing.T) {
	w := newFallbackWorld(t, "0.1.4000")
	pub, priv, _ := ed25519.GenerateKey(nil)
	signingKey = base64.StdEncoding.EncodeToString(pub)
	w.sig = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte("jianmu release 0.1.3999\n"+w.sums)))
	if _, err := Latest(context.Background()); err == nil {
		t.Fatal("accepted a signature for another version")
	}
}

// An existing but empty signature is invalid even before the signing cutoff.
func TestGitHubFallbackRefusesEmptySignatureForOldVersions(t *testing.T) {
	w := newFallbackWorld(t, "0.1.100")
	w.sig = ""
	if _, err := Latest(context.Background()); err == nil {
		t.Fatal("accepted an empty signature from GitHub")
	}
}

// A feed that answers is used and GitHub isn't asked; a custom feed
// (JIANMU_UPDATE_FEED) never falls back.
func TestGitHubIsOnlyAFallback(t *testing.T) {
	w := newFallbackWorld(t, "0.1.4000")
	t.Setenv("JIANMU_UPDATE_FEED", siteFeed+"?mine")
	if _, err := Latest(context.Background()); err == nil {
		t.Fatal("a custom feed fell back")
	}
	if w.apiHits.Load() != 0 {
		t.Fatal("GitHub was asked")
	}
}

// On Stable the site's choice can't be made by GitHub: no fallback.
func TestGitHubFallbackSkipsStable(t *testing.T) {
	w := newFallbackWorld(t, "0.1.4000")
	stableFor(t, "stable")
	if _, err := Latest(context.Background()); err == nil {
		t.Fatal("Stable fell back to the newest release")
	}
	if w.apiHits.Load() != 0 {
		t.Fatal("GitHub was asked")
	}
}

// A feed that names its files on the site gets their github.com address
// too, so a network without the site downloads them from GitHub.
func TestFeedFilesGetTheirGitHubAddress(t *testing.T) {
	rel := &Release{Version: "0.1.4000", Assets: map[string]Asset{
		"a": {URL: Site + "/dl/v0.1.4000/a"},
		"b": {URL: "https://github.com/x/y/releases/download/v1/b"},
		"c": {URL: Site + "/dl/v0.1.4000/c", GitHub: "https://github.com/keep"},
	}}
	rel.githubAddresses()
	if got := rel.Assets["a"].GitHub; got != githubWeb+"/"+releaseRepo+"/releases/download/v0.1.4000/a" {
		t.Fatalf("a: %q", got)
	}
	if rel.Assets["b"].GitHub != "" || rel.Assets["c"].GitHub != "https://github.com/keep" {
		t.Fatalf("b, c: %+v %+v", rel.Assets["b"], rel.Assets["c"])
	}
}
