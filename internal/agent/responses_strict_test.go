package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/DamonZS/Jianmu/internal/catalog"
	"github.com/DamonZS/Jianmu/internal/provider"
)

// dshModelOf is the model id of a route as dshPiProviders parsed it.
func dshModelOf(t *testing.T, ps map[string]dshPiRoute, key, id string) *dshPiModel {
	t.Helper()
	r, ok := ps[key]
	if !ok {
		t.Fatalf("no route %s", key)
	}
	for i := range r.Models {
		if r.Models[i].ID == id {
			return &r.Models[i]
		}
	}
	t.Fatalf("route %s has no %s: %+v", key, id, r.Models)
	return nil
}

// #3055 (ArakawaHenri): OpenAI's Responses API takes a function tool whose
// strict is left out as a strict-mode one and normalizes its schema, where
// every property is required, so a tool's optional parameters came back
// filled. pi-ai says strict: false on an ordinary tool only when the
// model's compat says supportsStrictMode, which it takes as false where
// the compat is left out — as jianmu's routes were. jianmu now says it on
// every model of a route whose api stands as openai-responses (the one a
// user picked in dsh's Models page kept), and on none of a Chat or
// Messages route.
func TestDshTellsAResponsesToolsStrictIsOff(t *testing.T) {
	home, dir, web := dshProtocolHome(t)
	dshKeyFixture(dir)
	os.WriteFile(web, []byte("# Your patch layer for this dsh profile.\n"), 0o644)
	a := dsh(home)
	if err := a.Field("model").Set("jianmu/resp/gpt-5.5"); err != nil {
		t.Fatal(err)
	}
	ps := dshPiProviders(t, web)
	if m := dshModelOf(t, ps, "jianmu-responses", "resp/gpt-5.5"); m.Compat == nil || !m.Compat.SupportsStrictMode {
		t.Fatalf("a model asked on Responses isn't told strict is off: %+v", m)
	}
	if m := dshModelOf(t, ps, "jianmu", "deepseek/pro"); m.Compat != nil && m.Compat.SupportsStrictMode {
		t.Fatalf("a Chat model was told strict is off: %+v", m)
	}
	if dshModelOf(t, ps, "jianmu", "deepseek/flash").Compat != nil {
		t.Fatal("a Chat model took a compat of its own")
	}
	if m := dshModelOf(t, ps, "jianmu-anthropic", "anth/claude-sonnet-5"); m.Compat == nil || !m.Compat.ForceAdaptiveThinking || m.Compat.SupportsStrictMode {
		t.Fatalf("a Claude on Messages: %+v", m)
	}

	// a round over what it wrote leaves it as it is
	was, _ := os.ReadFile(web)
	dshWiredOnce()
	if again, _ := os.ReadFile(web); string(again) != string(was) {
		t.Fatalf("written again:\n%s", again)
	}

	// the api a user picked for a route in dsh's Models page decides, not
	// the route's name: jianmu's first route on Responses tells its Chat
	// models strict is off and keeps a key of theirs beside jianmu's; the
	// responses route on Chat tells its model nothing there
	os.WriteFile(web, []byte("# Your patch layer for this dsh profile.\n"+
		"- id: llm-pi-ai # jianmu\n  name: \"@deepseek-ai/dsh-llm-pi-ai\"\n  config:\n    providers:\n"+
		"      jianmu:\n        displayName: Jianmu\n        apiKeyEnv: "+dshKeyRef+"\n        api: openai-responses\n        baseURL: "+gatewayV1()+"\n        models:\n"+
		"          - id: deepseek/pro\n            compat:\n              maxTokensField: max_tokens\n"+
		"      jianmu-responses:\n        displayName: Jianmu · Responses\n        apiKeyEnv: "+dshKeyRef+"\n        api: openai-completions\n        baseURL: "+gatewayV1()+"\n        models:\n"+
		"          - id: resp/gpt-5.5\n"), 0o644)
	if err := a.Sync(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(web)
	_, compat := dshRouteCompat(t, web, "jianmu")
	c := compat["deepseek/pro"]
	if c["supportsStrictMode"] != true || c["maxTokensField"] != "max_tokens" {
		t.Fatalf("a Chat model on a route switched to Responses: %v\n%s", c, b)
	}
	_, compat = dshRouteCompat(t, web, "jianmu-responses")
	if c = compat["resp/gpt-5.5"]; c != nil && c["supportsStrictMode"] == true {
		t.Fatalf("a model on a route switched to Chat was told strict is off: %v\n%s", c, b)
	}
}

// The same for Pi's and StepCode's model entries (piModelJSON): a model
// best served on Responses carries compat.supportsStrictMode, a Chat
// model's entry carries no compat at all. Pencil's entries stay on Chat
// throughout, so it takes none.
func TestPiTellsAResponsesToolsStrictIsOff(t *testing.T) {
	home := syncHome(t)
	if err := provider.Save(provider.Provider{ID: "resp", Name: "Resp", Key: "k", Responses: "http://127.0.0.1:1/v1", Models: []string{"gpt-5.5"}}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveLive("resp", "http://127.0.0.1:1/v1", []catalog.Model{{ID: "gpt-5.5", Efforts: []string{"high"}}}); err != nil {
		t.Fatal(err)
	}
	piModels := filepath.Join(home, ".pi", "agent", "models.json")
	// the provider block as jianmu wrote it before this provider was, with
	// a key of pi-cache-optimizer's in it
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
	// the models.json the test writes is one jianmu wrote while Pi was
	// connected: a sync only tends a ListOnly agent's file while it
	// reads as connected (Connect), and a block jianmu never set up is
	// the user's and is left alone
	if err := pi(home).Connect(); err != nil {
		t.Fatal(err)
	}
	SyncCatalog()
	b, err := os.ReadFile(piModels)
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Providers struct {
			Jianmu struct {
				Models []struct {
					ID     string         `json:"id"`
					API    string         `json:"api"`
					Compat map[string]any `json:"compat"`
				} `json:"models"`
			} `json:"jianmu"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("%v:\n%s", err, b)
	}
	strict := map[string]bool{}
	for _, m := range f.Providers.Jianmu.Models {
		if m.API == "openai-responses" && m.Compat["supportsStrictMode"] != true {
			t.Fatalf("%s is asked on Responses and isn't told strict is off:\n%s", m.ID, b)
		}
		strict[m.ID] = m.Compat["supportsStrictMode"] == true
	}
	if !strict["resp/gpt-5.5"] {
		t.Fatalf("no responses model of jianmu's carries supportsStrictMode:\n%s", b)
	}
	if strict["relay/glm-4.6"] {
		t.Fatalf("a Chat model was told strict is off:\n%s", b)
	}
}
