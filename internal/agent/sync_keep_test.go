package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// providerAt is the object at providers.<id> (or provider.<id>) of a JSON
// file, decoded.
func providerAt(t *testing.T, path, top string) map[string]any {
	t.Helper()
	var f map[string]json.RawMessage
	var ps map[string]map[string]any
	if err := json.Unmarshal([]byte(readFile(path)), &f); err != nil || json.Unmarshal(f[top], &ps) != nil {
		t.Fatalf("%s: %v\n%s", path, err, readFile(path))
	}
	return ps[jianmuID]
}

// #1103: pi-cache-optimizer's /cache-optimizer fix adds
// providers.jianmu.compat.sendSessionAffinityHeaders to Pi's models.json.
// A catalog sync, and a pick of a jianmu model in Pi, keep it; the model
// list is still jianmu's, so a model gone from the catalog leaves it, and
// a sync with nothing new writes nothing.
func TestPiKeepsTheirKeysInJianmusProvider(t *testing.T) {
	home := syncHome(t)
	piModels := filepath.Join(home, ".pi", "agent", "models.json")
	// the block as jianmu wrote it, with the optimizer's key added
	writeFile(t, piModels, `{
  "providers": {
    "jianmu": {
      "api": "openai-completions",
      "apiKey": "jianmu",
      "baseUrl": "http://127.0.0.1:3425/v1",
      "compat": {
        "sendSessionAffinityHeaders": true
      },
      "models": [
        {
          "id": "gone/old-model",
          "name": "Old"
        }
      ],
      "name": "jianmu"
    }
  }
}
`)
	// Pi runs a model of jianmu's: connected, so its list follows
	writeFile(t, filepath.Join(home, ".pi", "agent", "settings.json"), `{"defaultProvider":"jianmu","defaultModel":"relay/glm-4.6"}`)
	setUpInJianmu("pi")
	SyncCatalog()
	p := providerAt(t, piModels, "providers")
	compat, _ := p["compat"].(map[string]any)
	if compat["sendSessionAffinityHeaders"] != true {
		t.Fatalf("compat.sendSessionAffinityHeaders dropped by the sync:\n%s", readFile(piModels))
	}
	ms, _ := json.Marshal(p["models"])
	if strings.Contains(string(ms), "gone/old-model") || !strings.Contains(string(ms), "relay/glm-4.6") {
		t.Fatalf("models not jianmu's list: %s", ms)
	}

	// synced already: the next sync writes nothing
	st, _ := os.Stat(piModels)
	old := st.ModTime().Add(-1e12)
	os.Chtimes(piModels, old, old)
	SyncCatalog()
	if now, _ := os.Stat(piModels); !now.ModTime().Equal(old) {
		t.Errorf("models.json rewritten though nothing changed:\n%s", readFile(piModels))
	}

	// picking a jianmu model in Pi writes the block too
	if err := pi(home).Fields[0].Set("jianmu/relay/glm-4.6"); err != nil {
		t.Fatal(err)
	}
	compat, _ = providerAt(t, piModels, "providers")["compat"].(map[string]any)
	if compat["sendSessionAffinityHeaders"] != true {
		t.Fatalf("compat.sendSessionAffinityHeaders dropped by a pick:\n%s", readFile(piModels))
	}
}

// The same for OpenCode's provider.jianmu (an options key of the user's
// own) and Crush's providers.jianmu.
func TestOpenCodeAndCrushKeepTheirKeysInJianmusProvider(t *testing.T) {
	home := syncHome(t)
	cfg := filepath.Join(home, ".config", "opencode", "opencode.json")
	writeFile(t, cfg, `{"model":"jianmu/relay/glm-4.6","provider":{"jianmu":{"name":"jianmu","npm":"@ai-sdk/openai-compatible","options":{"baseURL":"http://127.0.0.1:3425/v1","apiKey":"jianmu","setCacheKey":true},"models":{"gone/old-model":{"name":"Old"}}}}}`)
	oc := opencode(home, filepath.Join(home, ".config"))
	if err := oc.Sync(); err != nil {
		t.Fatal(err)
	}
	p := providerAt(t, cfg, "provider")
	opts, _ := p["options"].(map[string]any)
	ms, _ := p["models"].(map[string]any)
	if opts["setCacheKey"] != true || ms["gone/old-model"] != nil || ms["relay/glm-4.6"] == nil {
		t.Fatalf("opencode sync:\n%s", readFile(cfg))
	}
	if err := oc.Fields[0].Set("jianmu/relay/glm-4.6"); err != nil {
		t.Fatal(err)
	}
	if opts, _ := providerAt(t, cfg, "provider")["options"].(map[string]any); opts["setCacheKey"] != true {
		t.Fatalf("opencode pick:\n%s", readFile(cfg))
	}

	cr := filepath.Join(home, ".config", "crush", "crush.json")
	writeFile(t, cr, `{"providers":{"jianmu":{"type":"openai","name":"jianmu","base_url":"http://127.0.0.1:3425/v1","api_key":"jianmu","extra_headers":{"X-Mine":"1"},"models":[]}}}`)
	c := crushAt(here(home), cr, filepath.Join(home, ".local", "share", "crush", "crush.json"))
	if err := c.Sync(); err != nil {
		t.Fatal(err)
	}
	if h, _ := providerAt(t, cr, "providers")["extra_headers"].(map[string]any); h["X-Mine"] != "1" {
		t.Fatalf("crush sync:\n%s", readFile(cr))
	}
	if err := c.Fields[0].Set("jianmu/relay/glm-4.6"); err != nil {
		t.Fatal(err)
	}
	if h, _ := providerAt(t, cr, "providers")["extra_headers"].(map[string]any); h["X-Mine"] != "1" {
		t.Fatalf("crush pick:\n%s", readFile(cr))
	}
}
