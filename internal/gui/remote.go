package gui

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DamonZS/Jianmu/internal/edit"
	"github.com/DamonZS/Jianmu/internal/settings"
)

// The app opens another jianmu's page in a window of its own: a jianmu on
// a server (`jianmu daemon install --web`, `jianmu web`, or an app with
// Settings › Network and sharing › Browser access on), whose routing and
// usage the user reads from the desktop (xuanwo on X). The window is that
// jianmu's own web page, signed in with its own key as a browser is
// (webGuard there): nothing is served here, and the page in the window
// talks to that jianmu alone. The address and key live in
// remote-jianmu.json beside settings.json, this computer's alone, never
// in settings.json: a sync or a backup never carries the key elsewhere.

// remoteFile is where the other jianmu's address and key are kept, 0600.
func remoteFile() string { return filepath.Join(settings.Dir(), "remote-jianmu.json") }

// remoteConf is remote-jianmu.json: URL is the page's address without the
// key, Key the key its link carries.
type remoteConf struct {
	URL string `json:"url,omitempty"`
	Key string `json:"key,omitempty"`
}

// loadRemote reads remote-jianmu.json; none is an empty conf.
func loadRemote() (remoteConf, error) {
	var c remoteConf
	b, err := os.ReadFile(remoteFile())
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return remoteConf{}, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return remoteConf{}, fmt.Errorf("%s: %w", remoteFile(), err)
	}
	return c, nil
}

// saveRemote writes remote-jianmu.json readable by its owner alone: the
// key opens every setting and key the other jianmu has.
func saveRemote(c remoteConf) error {
	p := remoteFile()
	if c == (remoteConf{}) {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	if f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600); err == nil {
		f.Close()
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	if err := edit.WriteAtomic(p, append(b, '\n')); err != nil {
		return err
	}
	return os.Chmod(p, 0o600)
}

// remoteAddr reads what was typed as the other jianmu's address: the link
// it printed (with ?k=, which is taken out as the key) or the bare
// address, https:// when no scheme is named. Plain http is taken only
// where the key doesn't cross the internet in the clear: an IP address
// (this computer, the LAN, a Tailscale address), localhost, a .local name
// or a one-word name; a name on the internet needs https, which is also
// what a Mac's webview asks of one.
func remoteAddr(raw string) (addr, key string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", errors.New("type the other jianmu's address")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return "", "", fmt.Errorf("%q isn't an address such as https://jianmu.example.com", raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", fmt.Errorf("the address is http:// or https://, not %s://", u.Scheme)
	}
	if u.User != nil {
		return "", "", errors.New("the address can't carry a user name or password: the key signs in")
	}
	if u.Scheme == "http" && !plainHTTPOK(u.Hostname()) {
		return "", "", fmt.Errorf("%s is a name on the internet: open it over https://, or the key crosses the network unencrypted", u.Hostname())
	}
	q := u.Query()
	key = q.Get("k")
	q.Del("k")
	u.RawQuery = q.Encode()
	u.Fragment, u.RawFragment = "", ""
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String(), key, nil
}

// plainHTTPOK: a page on host may be reached over plain http.
func plainHTTPOK(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	return net.ParseIP(h) != nil || h == "localhost" || strings.HasSuffix(h, ".local") || !strings.Contains(h, ".")
}

// remoteLink is the address with the key, which the page trades for its
// cookie and takes out of the address bar (webGuard).
func remoteLink(c remoteConf) string {
	u, err := url.Parse(c.URL)
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("k", c.Key)
	u.RawQuery = q.Encode()
	return u.String()
}

// remoteProbe asks the other jianmu whether it takes the key, as the
// window's first load would, so a wrong address or key is said in Settings
// rather than as a window with an error page. It reads nothing behind the
// key: webGuard answers a link with the key with a redirect and a cookie,
// and one without it with 401.
var remoteProbe = func(c remoteConf) error {
	cl := &http.Client{
		Timeout:       8 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequest(http.MethodGet, remoteLink(c), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/html")
	resp, err := cl.Do(req)
	if err != nil {
		var cert *tls.CertificateVerificationError
		if errors.As(err, &cert) {
			return fmt.Errorf("%s's certificate isn't trusted by this computer: %v", req.URL.Host, cert.Err)
		}
		return fmt.Errorf("%s doesn't answer: %v", req.URL.Host, errors.Unwrap(err))
	}
	resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusSeeOther && remoteSignedIn(resp):
		return nil
	case resp.StatusCode == http.StatusUnauthorized:
		return errors.New("that jianmu turned the key away: use the key in the link it printed (JIANMU_WEB_KEY, or the k= in it)")
	case resp.StatusCode == http.StatusForbidden:
		return errors.New("that jianmu doesn't let this computer in: add this computer's address to its allowed addresses")
	}
	return fmt.Errorf("%s answered %s, not as a jianmu page: check the address and port (jianmu web's is 3430)", req.URL.Host, resp.Status)
}

// remoteSignedIn: the answer sets jianmu web's cookie.
func remoteSignedIn(resp *http.Response) bool {
	for _, c := range resp.Cookies() {
		if strings.HasPrefix(c.Name, "jianmu_web_") {
			return true
		}
	}
	return false
}

// remoteOpener is a host that can open the other jianmu's window: the
// app's, not a browser's.
type remoteOpener interface {
	OpenRemote(link, title string)
}

// remoteJSON is GET /api/remote: the address kept, and whether a key is.
// The key itself never goes back to the page.
type remoteJSON struct {
	URL    string `json:"url"`
	HasKey bool   `json:"hasKey"`
}

// remoteRoutes: GET /api/remote says what is kept; POST takes {"url",
// "key"}, checks that jianmu takes the key, keeps both and opens its
// window (a key left out keeps the one kept for the same address);
// POST /api/remote/forget forgets them. The app's own windows' alone.
func remoteRoutes(mux *http.ServeMux, w Windows) {
	mux.HandleFunc("GET /api/remote", func(rw http.ResponseWriter, r *http.Request) {
		c, err := loadRemote()
		if err != nil {
			fail(rw, err)
			return
		}
		writeJSON(rw, remoteJSON{URL: c.URL, HasKey: c.Key != ""})
	})
	mux.HandleFunc("POST /api/remote", func(rw http.ResponseWriter, r *http.Request) {
		var in struct {
			URL string `json:"url"`
			Key string `json:"key"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(rw, r.Body, 8192)).Decode(&in); err != nil {
			fail(rw, err)
			return
		}
		addr, key, err := remoteAddr(in.URL)
		if err != nil {
			fail(rw, err)
			return
		}
		if k := strings.TrimSpace(in.Key); k != "" {
			key = k
		}
		if key == "" {
			if old, err := loadRemote(); err == nil && old.URL == addr {
				key = old.Key
			}
		}
		if key == "" {
			fail(rw, errors.New("type that jianmu's key: the k= in the link it printed, or its JIANMU_WEB_KEY"))
			return
		}
		if err := CheckWebKey(key); err != nil {
			fail(rw, errors.New(strings.Replace(err.Error(), "JIANMU_WEB_KEY", "the key", 1)))
			return
		}
		o, ok := w.(remoteOpener)
		if !ok {
			fail(rw, errors.New("only the app opens another jianmu in a window"))
			return
		}
		c := remoteConf{URL: addr, Key: key}
		if err := remoteProbe(c); err != nil {
			fail(rw, err)
			return
		}
		if err := saveRemote(c); err != nil {
			fail(rw, err)
			return
		}
		u, _ := url.Parse(addr)
		o.OpenRemote(remoteLink(c), "jianmu · "+u.Host)
		writeJSON(rw, remoteJSON{URL: c.URL, HasKey: true})
	})
	mux.HandleFunc("POST /api/remote/forget", func(rw http.ResponseWriter, r *http.Request) {
		if err := saveRemote(remoteConf{}); err != nil {
			fail(rw, err)
			return
		}
		writeJSON(rw, remoteJSON{})
	})
}
