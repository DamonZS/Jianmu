package gateway

import (
	"context"
	"net/http"
	"strings"

	"github.com/DamonZS/Jianmu/internal/provider"
)

// jianmuUA is the User-Agent jianmu's requests go to a vendor with.
func jianmuUA() string { return "jianmu/" + Version }

// userAgentFor is the User-Agent a request to p goes with, ua being the
// one the request came to jianmu with: the agent's own, as it sent it,
// where p passes it on (provider.PassesUserAgent: a key+URL provider not
// set to go as jianmu, #1851), and jianmu's where it doesn't, where the
// agent sent none, and on jianmu's own
// requests — its classifier's, vision's, search's, the image tool's
// (jianmusOwn) — which no agent sent. A User-Agent the user set among the
// provider's headers is put on after, when the request is signed, and
// stands over either.
func userAgentFor(p provider.Provider, ua string) string {
	if ua != "" && p.PassesUserAgent() && !jianmusOwn(ua) {
		return ua
	}
	return jianmuUA()
}

// jianmusOwn is a User-Agent jianmu's own requests carry: another jianmu's
// (jianmu/…), the classifier's, vision's and search's (jianmu-router/1,
// jianmu-vision/1, jianmu-search/1), and the image tool's, which names the
// agent it draws for before its own (claude-code/2.1.0 jianmu-image/1)
// from what the agent told it, not from a User-Agent the agent sent.
func jianmusOwn(ua string) bool {
	return strings.HasPrefix(ua, "jianmu/") || strings.HasPrefix(ua, "jianmu-") || strings.HasSuffix(ua, " "+DrawAgent)
}

// callerUA is the User-Agent the request ctx is for came with (withCaller),
// or the agent's another jianmu passed it on for; "" when there is none.
func callerUA(ctx context.Context) string {
	c, _ := ctx.Value(callerCtx{}).(caller)
	return c.ua
}

// stampUA puts on req, a request to p that isn't jianmu's own forwarded
// one (an embedding, a drawing, a video), the User-Agent forwardOnce's
// carry: jianmu's, or the agent's where p passes it on. A signed-in
// account's is left to its signing, as before, and another jianmu's is
// set where it is asked.
func stampUA(ctx context.Context, p provider.Provider, h http.Header) {
	if p.Account != nil || p.IsRemoteJianmu() {
		return
	}
	h.Set("User-Agent", userAgentFor(p, callerUA(ctx)))
}

// agentUA is the User-Agent of the agent a request with headers in is
// for: the one it came with, or, from another jianmu (jianmu/…), the
// agent's that jianmu named (ClientUAHeader). jianmu's own requests made
// here (jianmu-router/1, …) keep theirs.
func agentUA(ctx context.Context, in http.Header) string {
	ua := in.Get("User-Agent")
	if c, ok := ctx.Value(callerCtx{}).(caller); ok && c.relayed && strings.HasPrefix(ua, "jianmu/") {
		return c.ua
	}
	return ua
}

// passOnUA names, on a request to a remote jianmu, the agent's User-Agent
// ua, for that jianmu to send on as this one would; jianmu's own is
// never named, and a vendor is never sent the header (withCaller takes
// it off what comes in).
func passOnUA(req *http.Request, ua string) {
	req.Header.Del(ClientUAHeader)
	if ua != "" && !jianmusOwn(ua) {
		req.Header.Set(ClientUAHeader, ua)
	}
}
