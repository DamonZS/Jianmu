package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/DamonZS/Jianmu/internal/catalog"
	"github.com/DamonZS/Jianmu/internal/provider"
)

// yuyong_id on Discord: jianmu1 holds a Codex (ChatGPT) sign-in, jianmu2
// uses jianmu1 as a Remote jianmu, and jianmu3 uses jianmu2. jianmu2
// showed jianmu1's Codex card; jianmu3 showed none of it, as jianmu2
// told another jianmu only the cards of its own accounts. Every kind of
// card jianmu1 tells comes through to jianmu3 now, named after both
// jianmus; a refresh pressed at jianmu3 reaches jianmu1; a card that
// comes round to a jianmu it went through is left out.
func TestRemoteChainPassesOnCards(t *testing.T) {
	fresh(t)
	t.Setenv("JIANMU_ADDR", "")
	// jianmu1, as its gateway answers (CachedCards): its own cards, each
	// with the way it came by, jianmu1 alone
	var mu sync.Mutex
	var refreshes []string
	m1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-jianmu-m1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "GET " + provider.RemoteCardsPath:
		case "POST " + provider.RemoteRefreshPath:
			mu.Lock()
			refreshes = append(refreshes, r.URL.Query().Get("provider")+"|"+r.URL.Query().Get("user"))
			mu.Unlock()
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		io.WriteString(w, `{"object":"list","data":[
{"provider":"codex","name":"Codex","icon":"codex-color","plan":"plus","user":"a@x.com","kind":"subscription","via":["m1"],
 "windows":[{"name":"5h","used":37,"resetSecs":3600},{"name":"Week","used":12,"resetSecs":86400}],"resets":{"count":1}},
{"provider":"antigravity","name":"Antigravity","icon":"antigravity","user":"g@x.com","kind":"subscription","via":["m1"],
 "windows":[{"name":"Gemini","used":5}]},
{"provider":"zhipu","name":"GLM Coding","kind":"plan","via":["m1"],"windows":[{"name":"5h","used":10}]},
{"provider":"deepseek","name":"DeepSeek","kind":"balance","balance":"¥12.00","via":["m1"],"windows":[]}]}`)
	}))
	defer m1.Close()

	// jianmu2: this process's gateway, jianmu1 its Remote jianmu
	gw := httptest.NewServer(lanGuard(New().Handler()))
	defer gw.Close()
	if err := provider.Save(provider.Provider{ID: "m1", Name: "M1", Preset: provider.RemoteJianmuPreset, Key: "sk-jianmu-m1", Chat: m1.URL}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// jianmu2's Usage page reads jianmu1's cards, as reported
	var m2Codex bool
	for _, q := range provider.RemoteCards(ctx) {
		m2Codex = m2Codex || q.Provider == "m1/codex" && q.User == "a@x.com"
	}
	if !m2Codex {
		t.Fatal("jianmu2 doesn't show jianmu1's Codex card")
	}
	m2Self := catalog.Self()
	get := func(path string) []byte {
		t.Helper()
		res, err := http.Get(gw.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		if res.StatusCode != 200 {
			t.Fatalf("%s: %d %s", path, res.StatusCode, b)
		}
		return b
	}
	// what jianmu2 tells jianmu3, through its real handler
	told := get(provider.RemoteCardsPath)

	// jianmu3: another jianmu, with jianmu2 as its Remote jianmu, told
	// what jianmu2's gateway told
	defer catalog.SetSelfForTest("m3")()
	m2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method+" "+r.URL.Path != "GET "+provider.RemoteCardsPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(told)
	}))
	defer m2.Close()
	// one process plays both jianmus; jianmu3 knows jianmu2 alone, as the
	// same account by two paths would be shown once
	m1Saved := provider.Provider{ID: "m1", Name: "M1", Preset: provider.RemoteJianmuPreset, Key: "sk-jianmu-m1", Chat: m1.URL}
	if err := provider.Delete("m1"); err != nil {
		t.Fatal(err)
	}
	provider.ForgetRemoteCardsForTest()
	if err := provider.Save(provider.Provider{ID: "m2", Name: "M2", Preset: provider.RemoteJianmuPreset, Key: "sk-jianmu-m2", Chat: m2.URL}); err != nil {
		t.Fatal(err)
	}
	byID := map[string]provider.SubscriptionQuota{}
	for _, q := range remoteOnly(provider.RemoteCards(ctx)) {
		byID[q.Provider] = q
	}
	c, ok := byID["m2/m1/codex"]
	if !ok || c.Name != "Codex · M1 · M2" || c.From != "M2" || c.User != "a@x.com" || c.Kind != "subscription" ||
		len(c.Windows) != 2 || c.Windows[0].Used != 37 || c.Windows[1].Used != 12 {
		t.Fatalf("jianmu1's Codex card at jianmu3: %+v (all: %+v)", c, byID)
	}
	// a reset is spent on the jianmu next door only (UseRemoteCodexReset)
	if c.Resets != nil {
		t.Errorf("a reset two jianmus away is offered: %+v", c.Resets)
	}
	if !slices.Equal(c.Via, []string{"m1", m2Self}) {
		t.Errorf("the Codex card came by %v, want [m1 %s]", c.Via, m2Self)
	}
	// every kind passes on alike
	for id, kind := range map[string]string{"m2/m1/antigravity": "subscription", "m2/m1/zhipu": "plan", "m2/m1/deepseek": "balance"} {
		if q, ok := byID[id]; !ok || q.Kind != kind {
			t.Errorf("%s at jianmu3: %+v", id, q)
		}
	}
	if q := byID["m2/m1/deepseek"]; q.Balance != "¥12.00" {
		t.Errorf("jianmu1's DeepSeek balance at jianmu3: %+v", q)
	}
	// jianmu2's own stand-in for jianmu1, and its key's limit there, are
	// its own business with jianmu1
	var cards []provider.SubscriptionQuota
	json.Unmarshal(told, &struct {
		Data *[]provider.SubscriptionQuota `json:"data"`
	}{&cards})
	for _, q := range cards {
		if q.Provider == "m1" || q.GatewayKey != nil {
			t.Errorf("jianmu2 told jianmu3 its own card of jianmu1: %+v", q)
		}
	}

	if err := provider.Delete("m2"); err != nil {
		t.Fatal(err)
	}
	if err := provider.Save(m1Saved); err != nil {
		t.Fatal(err)
	}
	// a refresh pressed at jianmu3 is read again at jianmu1, by jianmu2
	res, err := http.Post(gw.URL+provider.RemoteRefreshPath+"?provider=m1/codex&user=a@x.com", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	mu.Lock()
	got := slices.Clone(refreshes)
	mu.Unlock()
	if res.StatusCode != 200 || !slices.Equal(got, []string{"codex|a@x.com"}) {
		t.Errorf("jianmu3's refresh of m1/codex: %d, jianmu1 asked %v", res.StatusCode, got)
	}

	// a card that went through this jianmu before is left out: jianmu3
	// told a card that came m3 → m2 shows nothing of it
	told = []byte(`{"object":"list","data":[{"provider":"m2/codex","name":"Codex · M2","from":"M2","kind":"subscription","user":"loop@x.com","via":["m3","m2x"],"windows":[]}]}`)
	provider.ForgetRemoteCardsForTest()
	for _, q := range provider.RemoteCards(ctx) {
		if q.User == "loop@x.com" {
			t.Errorf("a card that came back round: %+v", q)
		}
	}
}

// jianmu2's status line (GET /v1/jianmu/quotas?kept=1, KeptQuotaReport)
// is told its Remote jianmu's cards as last read, as its Usage page shows
// them: a jianmu whose Codex is another's had none on its status line.
func TestKeptReportHasRemoteCards(t *testing.T) {
	fresh(t)
	t.Setenv("JIANMU_ADDR", "")
	m1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"object":"list","data":[{"provider":"codex","name":"Codex","user":"a@x.com","kind":"subscription","via":["m1"],"windows":[{"name":"5h","used":37}]}]}`)
	}))
	defer m1.Close()
	if err := provider.Save(provider.Provider{ID: "m1", Name: "M1", Preset: provider.RemoteJianmuPreset, Key: "sk-jianmu-m1", Chat: m1.URL}); err != nil {
		t.Fatal(err)
	}
	provider.RemoteCards(context.Background())
	r := httptest.NewRequest("GET", "/v1/jianmu/quotas?kept=1", nil)
	r.RemoteAddr = "127.0.0.1:5000"
	w := httptest.NewRecorder()
	lanGuard(New().Handler()).ServeHTTP(w, r)
	var got struct {
		Data []provider.Quota `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &got)
	for _, q := range got.Data {
		if q.Provider == "m1/codex" && q.From == "M1" && len(q.Windows) == 1 && q.Windows[0].Used == 37 {
			return
		}
	}
	t.Fatalf("kept report: %d %s", w.Code, w.Body)
}
