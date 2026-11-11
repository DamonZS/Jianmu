package gui

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

const cf524 = `<!DOCTYPE html>
<!--[if lt IE 7]> <html class="no-js ie6 oldie" lang="en-US"> <![endif]-->
<head><title>jianmu.example.com | 524: A timeout occurred</title></head>
<body><h1>A timeout occurred</h1></body></html>`

// An error with a proxy's page in it reaches the page as the page's title,
// said by a key the GUI translates, and not as markup.
func TestFailSaysAProxyPageByItsTitle(t *testing.T) {
	w := httptest.NewRecorder()
	fail(w, errors.New("remote jianmu answered 524: "+cf524))
	var got map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	what := "remote jianmu answered 524: jianmu.example.com | 524: A timeout occurred"
	if got["what"] != what || got["said"] != "A proxy in front of jianmu answered, not jianmu ({what})" {
		t.Fatalf("got %v", got)
	}
	if strings.Contains(got["error"], "<") {
		t.Fatalf("markup in the error: %q", got["error"])
	}
	// no page: the text stays as it was
	w = httptest.NewRecorder()
	fail(w, errors.New("a < b and <b>bold</b>"))
	got = nil
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["error"] != "a < b and <b>bold</b>" || got["said"] != "" {
		t.Fatalf("plain text changed: %v", got)
	}
}
