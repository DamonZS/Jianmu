package agent

// Prime Agent (PrimeIntellect-ai/prime-agent, the prime-agent command) is
// a Rust agent that keeps Pi's settings.json and models.json in an agent
// folder of its own: $PRIME_AGENT_CODING_AGENT_DIR, "~" in it standing for
// home, else ~/.prime/agent (pa-types platform/dirs.rs, agent_dir). It
// reads both files with Pi's fields and types (pa-core models/custom.rs,
// settings/types.rs), so jianmu wires it as it does Pi:
// defaultProvider/defaultModel/defaultThinkingLevel in settings.json and
// jianmu as the provider "jianmu" in models.json.
//
// Its requests carry no User-Agent (its HTTP client sets none), so the key
// jianmu gives it names it (jianmu-prime-agent): that is how the gateway
// counts its requests as Prime Agent's.
//
// Its global settings.json alone is written, as Pi's is: a project's
// .prime/agent/settings.json is the user's, and so are allowedModels (its
// daemon's allow-list, which a model outside fails loudly) and
// subagentDefaultModel.

import (
	"github.com/DamonZS/Jianmu/internal/appdir"
)

func primeAgent(home string) *Agent { return primeAgentIn(here(home)) }

// primeAgentIn is Prime Agent at a place: this machine's home, where its
// variable moves it, or a WSL distro's, whose variables jianmu can't read.
func primeAgentIn(at place) *Agent {
	return piLikeKeyed(at, "prime-agent", "Prime Agent", primeAgentDir(at),
		func() string { return agentKeyAt("prime-agent", at.gw()) })
}

// primeAgentDir is Prime Agent's agent folder at a place. A relative
// PRIME_AGENT_CODING_AGENT_DIR is its working directory's, which jianmu
// can't know, so it is not taken.
func primeAgentDir(at place) string {
	if at.spell == nil {
		if d := homeDir(at.home, appdir.Getenv("PRIME_AGENT_CODING_AGENT_DIR")); d != "" {
			return d
		}
	}
	return at.join(".prime", "agent")
}
