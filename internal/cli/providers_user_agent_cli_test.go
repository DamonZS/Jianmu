package cli

import (
	"strings"
	"testing"

	"github.com/DamonZS/Jianmu/internal/provider"
)

// A key+URL provider passes the agent's User-Agent on by default (#1851,
// nb5p on Discord): user-agent=jianmu sends jianmu's in its place,
// user-agent=agent goes back to the default, and a signed-in account, which
// goes as its own client, is refused agent.
func TestProviderUserAgentPair(t *testing.T) {
	p := provider.Provider{ID: "plan", Name: "Plan", Chat: "https://plan.example/v1", Key: "k"}
	if err := applyPairs(&p, []string{"user-agent=jianmu"}); err != nil || !p.JianmuUserAgent || p.PassesUserAgent() {
		t.Fatalf("jianmu: %v %v", p.JianmuUserAgent, err)
	}
	if err := applyPairs(&p, []string{"user-agent=agent"}); err != nil || p.JianmuUserAgent || !p.PassesUserAgent() {
		t.Fatalf("agent: %v %v", p.JianmuUserAgent, err)
	}
	if err := applyPairs(&p, []string{"user-agent=claude"}); err == nil || !strings.Contains(err.Error(), "agent|jianmu") {
		t.Fatalf("a made-up value: %v", err)
	}
	a := provider.Provider{ID: "zcode", Name: "ZCode", Account: &provider.Account{Agent: "zcode"}}
	if err := applyPairs(&a, []string{"user-agent=agent"}); err == nil || a.JianmuUserAgent {
		t.Fatalf("an account took it: %v", err)
	}
	if err := applyPairs(&a, []string{"user-agent=jianmu"}); err != nil || a.JianmuUserAgent {
		t.Fatalf("an account turning it off: %v %v", a.JianmuUserAgent, err)
	}
}

// compaction=native has a provider whose wire carries it be asked Codex's
// own compaction (#1099), compaction=jianmu goes back to jianmu's summary,
// and a provider without a Responses API is refused it.
func TestProviderCompactionPair(t *testing.T) {
	p := provider.Provider{ID: "relay", Name: "Relay", Responses: "https://relay.example/v1", Key: "k"}
	if err := applyPairs(&p, []string{"compaction=native"}); err != nil || !p.NativeCompaction {
		t.Fatalf("native: %v %v", p.NativeCompaction, err)
	}
	if err := applyPairs(&p, []string{"compaction=jianmu"}); err != nil || p.NativeCompaction {
		t.Fatalf("jianmu: %v %v", p.NativeCompaction, err)
	}
	if err := applyPairs(&p, []string{"compaction=sometimes"}); err == nil || !strings.Contains(err.Error(), "native|jianmu") {
		t.Fatalf("a made-up value: %v", err)
	}
	c := provider.Provider{ID: "chat", Name: "Chat", Chat: "https://chat.example/v1", Key: "k"}
	if err := applyPairs(&c, []string{"compaction=native"}); err == nil {
		t.Fatalf("a chat-only provider took it")
	}
	cp := provider.Provider{ID: "copilot", Name: "GitHub Copilot", Responses: "https://api.githubcopilot.com", Account: &provider.Account{Agent: "copilot"}}
	if err := applyPairs(&cp, []string{"compaction=native"}); err != nil || !cp.NativeCompaction {
		t.Fatalf("copilot: %v %v", cp.NativeCompaction, err)
	}
}
