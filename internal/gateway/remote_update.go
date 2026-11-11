package gateway

import (
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/DamonZS/Jianmu/internal/access"
	"github.com/DamonZS/Jianmu/internal/provider"
)

var selfUpdater atomic.Pointer[func(want string) provider.RemoteUpdate]

// SetUpdater has provider.RemoteUpdatePath call f with the version the
// asking jianmu runs; the gui's updater and jianmu serve set it. Without
// one the answer is "unsupported".
func SetUpdater(f func(want string) provider.RemoteUpdate) {
	if f == nil {
		selfUpdater.Store(nil)
		return
	}
	selfUpdater.Store(&f)
}

// remoteUpdate updates this jianmu when another jianmu using it as a
// Remote jianmu has updated (provider.UpdateRemotes). It is the owner's
// to ask, as a Codex reset is: from another machine only while jianmu is
// shared, always with an enabled gateway key of full access (none
// read-only, none held to a budget, models or accounts), and never from
// a web page, which a key put in a page could otherwise do.
func (s *Server) remoteUpdate(w http.ResponseWriter, r *http.Request) {
	if !local(r) && !sharedWith(r) {
		writeError(w, provider.Chat, http.StatusForbidden, "jianmu is updated from another machine only when it is shared on the local network (Settings → Network and sharing → Share on local network) and the request carries its API key")
		return
	}
	if fromBrowser(r) {
		writeError(w, provider.Chat, http.StatusForbidden, "jianmu doesn't update itself when a web page asks")
		return
	}
	who := access.Caller(r.Context())
	if who.KeyID == "" || who.ReadOnly {
		writeError(w, provider.Chat, http.StatusUnauthorized, "updating jianmu takes an enabled jianmu gateway key with full access (Gateway → Gateway keys), sent as Authorization: Bearer <key> or x-api-key, from this computer too")
		return
	}
	if refuseHeld(w, r, "this gateway key is held to a budget, models or accounts, so it doesn't update jianmu: use a key with full access") {
		return
	}
	want := strings.TrimSpace(r.URL.Query().Get("version"))
	out := provider.RemoteUpdate{State: "unsupported", Why: "source"}
	if f := selfUpdater.Load(); f != nil {
		out = (*f)(want)
	}
	out.Version = Version
	log.Printf("update to follow %q asked by gateway key %q from %s: %s %s", want, who.KeyName, r.RemoteAddr, out.State, out.Why)
	writeJSON(w, http.StatusOK, out)
}
