package gateway

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/DamonZS/Jianmu/internal/catalog"
	"github.com/DamonZS/Jianmu/internal/provider"
)

// A Remote jianmu searches the web for the models it serves as it would
// for its own agents: by the model's provider, when that searches by
// itself, else with its own searcher. Its list says which of the two for
// each model (web_search: "native" or "jianmu"), and a web search offered
// to one of them goes to it as it was offered rather than being run here
// with this jianmu's searcher (Player on Discord: remoteJianmu/glm-5.3's
// searches went to another model). Only a model it searches by itself,
// natively, is one this jianmu may search with for its other models: a
// search jianmu runs for another model is marked (SearchingHeader), and
// the other jianmu runs no searcher of its own for it, so two jianmus
// that each name the other never ask each other in turn.

// SearchingHeader marks a request a jianmu made to search for another
// model, to a remote jianmu, which then searches only by the model itself.
const SearchingHeader = "X-Magpie-Searching"

// Web search as a Remote jianmu's list says it.
const (
	searchNative = "native"
	searchJianmu = "jianmu"
)

// remoteSearch is how a Remote jianmu says it searches the web for model,
// "" for another provider or a model it doesn't.
func remoteSearch(p provider.Provider, model string) string {
	if !p.IsRemoteJianmu() {
		return ""
	}
	if i := slices.IndexFunc(p.Available(), func(m catalog.Model) bool { return m.ID == model }); i >= 0 {
		return p.Available()[i].WebSearch
	}
	return ""
}

// remoteSearchesAny is whether a Remote jianmu searches natively for any
// of its models, which makes it one this jianmu may search with.
func remoteSearchesAny(p provider.Provider) bool {
	return p.IsRemoteJianmu() && slices.ContainsFunc(p.Available(), func(m catalog.Model) bool { return m.WebSearch == searchNative })
}

// searchableModel is a model of p that jianmu may search with: a Gemini
// on a Google sign-in, one a Remote jianmu searches natively for, one of
// searchModels or a plugin marks that searches by itself (partSearcher),
// any other
// provider's.
func searchableModel(p provider.Provider, model string) bool {
	switch {
	case googleAccount(p), geminiKeySearches(p):
		return searchModel(model)
	case p.IsRemoteJianmu():
		return remoteSearch(p, model) == searchNative
	case partSearcher(p):
		return searchesAlone(p, model)
	}
	return true
}

// webSearchOf is what this jianmu's list tells another jianmu of how it
// searches for e (remoteSearch): jianmu is whether it has a searcher.
func webSearchOf(e provider.Entry, jianmu bool) string {
	p := e.Provider
	if e.Group == "" {
		if p.IsRemoteJianmu() && remoteSearch(p, e.Model) == searchNative || codeAssistSearches(p, provider.CodeAssist, e.Model) {
			return searchNative
		}
		for _, proto := range p.Speaks() {
			if searchesItself(p, proto) || modelSearches(p, proto, e.Model) {
				return searchNative
			}
		}
	}
	if jianmu || searchesOwn(p) {
		return searchJianmu
	}
	return ""
}

// markSearching marks a request to a remote jianmu made for a search.
func markSearching(ctx context.Context, req *http.Request) {
	if searching(ctx) {
		req.Header.Set(SearchingHeader, "1")
	}
}

// searchingFrom is whether a request is a search another jianmu made.
func searchingFrom(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("User-Agent"), "jianmu/") && r.Header.Get(SearchingHeader) != ""
}
