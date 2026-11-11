package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/DamonZS/Jianmu/internal/catalog"
	"github.com/DamonZS/Jianmu/internal/filememo"
)

// A Remote jianmu's list is followed (KaCli on Discord): the jianmu
// serving the gateway here waits at the other jianmu's list for it to
// change (ListWaitHeader), and fetches it again the moment it has, through
// the same RefetchTaking as KeepListsFresh, so the agents' configs written
// from it (catalog.Touched, then their Sync) follow a route, group or
// provider changed there within seconds rather than on the next
// RemoteJianmuEvery. The schedule stays, for a jianmu from before that
// doesn't wait (it answers its list at once, with no ETag), which is then
// asked to wait again only after followOldAgain. A Remote jianmu whose
// user set its list to be fetched only when asked ("off") is not
// followed either.

// ListWaitHeader asks a jianmu's model list to be answered once it is
// other than the one If-None-Match names, waiting up to the seconds it
// says; at the end of the wait, 304 says it is the same.
const ListWaitHeader = "X-Magpie-List-Wait"

// listWaitAsk is how many seconds a follower asks the other jianmu to
// wait, which holds it under a minute (the gateway's listWaitMost).
const listWaitAsk = 50

// follow paces: the shortest time between two asks of one jianmu, the
// first wait after a failure (doubled to followFailMost), and how long
// a jianmu that doesn't wait is left to the schedule alone.
var (
	followGap      = time.Second
	followFailWait = 5 * time.Second
	followFailMost = 5 * time.Minute
	followOldAgain = time.Hour
	// followLook is how often the providers followed are looked at again
	// with no write of jianmu's to say they changed
	followLook = time.Minute
	// followQuick: a 304 sooner than this that doesn't say it waited
	// (ListWaitHeader) is from a jianmu that didn't wait
	followQuick = 5 * time.Second
)

// errListNotWaited is a jianmu that answered the list at once, with no
// ETag: one from before ListWaitHeader. Or one whose 304 came at once and
// doesn't say it waited: a jianmu that tags every list, reached through a
// proxy that dropped ListWaitHeader, which asked again would be asked
// each followGap.
var errListNotWaited = errors.New("its jianmu doesn't wait for its list to change; fetched on its schedule")

// KeepRemoteListsFollowed follows the list of each Remote jianmu the user
// has, while ctx lasts: run by the jianmu serving the gateway, beside
// KeepListsFresh.
func KeepRemoteListsFollowed(ctx context.Context) {
	type follower struct {
		stop context.CancelFunc
		done chan struct{}
	}
	going := map[string]follower{}
	defer func() {
		for _, f := range going {
			f.stop()
			<-f.done
		}
	}()
	for {
		written := filememo.Written()
		want := map[string]Provider{}
		for _, p := range followedJianmus() {
			want[followKey(p)] = p
		}
		for k, f := range going {
			if _, ok := want[k]; !ok {
				f.stop()
				<-f.done
				delete(going, k)
			}
		}
		for k, p := range want {
			if _, ok := going[k]; ok {
				continue
			}
			fctx, stop := context.WithCancel(ctx)
			f := follower{stop, make(chan struct{})}
			going[k] = f
			go func() {
				defer close(f.done)
				followList(fctx, p.ID)
			}()
		}
		look := time.NewTimer(followLook)
		select {
		case <-ctx.Done():
			look.Stop()
			return
		case <-written:
		case <-look.C:
		}
		look.Stop()
	}
}

// followedJianmus are the Remote jianmus whose lists are followed: on,
// listing, not this jianmu itself, with a schedule (their user's, or
// RemoteJianmuEvery when none is set).
func followedJianmus() []Provider {
	own, _ := ownOf(load())
	var out []Provider
	for _, p := range own {
		if p.IsRemoteJianmu() && !p.Off && p.lists() && p.Anthropic != "" && !p.SelfLeftOut() && p.RefreshInterval() > 0 {
			out = append(out, p)
		}
	}
	return out
}

// followKey tells a follower to start again when what it asks with is
// another: its address, key, headers or proxy.
func followKey(p Provider) string {
	return fmt.Sprint(p.ID, "\x00", p.Anthropic, "\x00", p.Key, "\x00", p.Proxy, "\x00", p.Headers)
}

// followed counts, for the tests, the lists a follower fetched again.
var followed = struct {
	sync.Mutex
	n map[string]int
}{n: map[string]int{}}

// followList waits at the Remote jianmu id's list for its change, and
// fetches it again each time it changed, until ctx ends. The list it has
// first is fetched once too: it may have changed since it was fetched.
func followList(ctx context.Context, id string) {
	tag := ""
	failWait := followFailWait
	pause := func(d time.Duration) bool {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return false
		case <-t.C:
			return true
		}
	}
	for ctx.Err() == nil {
		asked := time.Now()
		p := followedOne(id)
		if p == nil {
			return
		}
		now, err := waitRemoteList(ctx, *p, tag)
		switch {
		case ctx.Err() != nil:
			return
		case errors.Is(err, errListNotWaited):
			log.Printf("%s: %v", id, err)
			tag = ""
			if !pause(followOldAgain) {
				return
			}
			continue
		case err != nil:
			if !pause(failWait) {
				return
			}
			failWait = min(failWait*2, followFailMost)
			continue
		}
		failWait = followFailWait
		if now != tag {
			fctx, cancel := context.WithTimeout(ctx, time.Minute)
			ms, dropped, taken, err := p.RefetchTaking(fctx)
			cancel()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				// the tag isn't kept, so the next answer fetches again
				log.Printf("%s: its list changed there, and its fetch failed: %s", id, err)
				if !pause(failWait) {
					return
				}
				continue
			}
			if tag != "" {
				log.Printf("%s: its list changed there, fetched again: %d models%s%s", id, len(ms), listNote(" taken by rule: ", taken), listNote(" no longer listed: ", dropped))
			}
			followed.Lock()
			followed.n[id]++
			followed.Unlock()
			tag = now
		}
		if wait := followGap - time.Since(asked); wait > 0 && !pause(wait) {
			return
		}
	}
}

// followedOne is the Remote jianmu id as it is saved now, if it is still
// followed.
func followedOne(id string) *Provider {
	for _, p := range followedJianmus() {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

// waitRemoteList asks p's list to be answered once it is other than the
// one tag names, and answers the tag of the list it has now. It is asked
// as the list is fetched (listHeaders, signRemote), so the list the other
// jianmu compares is the one a fetch is answered.
func waitRemoteList(ctx context.Context, p Provider, tag string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, (listWaitAsk+20)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.Anthropic+"/v1/models", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "jianmu")
	for k, v := range p.listHeaders() {
		catalog.PutUserHeader(req.Header, k, v)
	}
	p.signRemote(req, p.Key)
	req.Header.Set(ListWaitHeader, strconv.Itoa(listWaitAsk))
	if tag != "" {
		req.Header.Set("If-None-Match", tag)
	}
	asked := time.Now()
	res, err := p.Do(http.DefaultClient, req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, io.LimitReader(res.Body, 64<<20))
	switch {
	case res.StatusCode == http.StatusNotModified:
		// a jianmu from before this one says no wait it held, and holds
		// each wait to its end; a 304 at once is a gateway stopping, or a
		// list that wasn't waited for
		if res.Header.Get(ListWaitHeader) == "" && time.Since(asked) < followQuick {
			return "", errListNotWaited
		}
		return tag, nil
	case res.StatusCode != http.StatusOK:
		return "", fmt.Errorf("%s answered %s", remoteName(p), res.Status)
	}
	now := res.Header.Get("ETag")
	if now == "" {
		return "", errListNotWaited
	}
	return now, nil
}
