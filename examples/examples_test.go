package examples

import (
	"bytes"
	"cmp"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var checks = map[string][]check{
	"1-minimal": {
		{"GET", "/", 200, "Example"},
		{"GET", "/posts", 200, "Hello"},
		{"GET", "/posts/hello", 200, "Some words"},
		{"GET", "/missing", 404, ""},
	},
	"2-pictures": {
		{"GET", "/", 200, "Pictures"},
		{"GET", "/posts", 200, "ridge"},
		{"GET", "/posts/ridge", 200, "/_quire/img/"},
		{"GET", "/posts/ridge", 200, "<video"},
		{"GET", "/posts/plain", 200, "<svg"},
		{"GET", "/missing", 404, ""},
	},
	"3-themed": {
		{"GET", "/", 200, "page--wide"},
		{"GET", "/", 200, "Featured"},
		{"GET", "/", 200, `data-theme="dark"`},
		{"GET", "/", 200, "Written and built by"},
		{"GET", "/", 200, "Quiet Machine"},
		{"GET", "/writing", 200, "Tags that know"},
		{"GET", "/writing/unfinished", 404, ""},
		{"GET", "/writing/one-binary", 200, `class="swatch"`},
		{"GET", "/writing/one-binary", 200, "<svg"},
		{"GET", "/writing/one-binary", 200, "tag-lang"},
		{"GET", "/writing/one-binary", 200, "toc-item"},
		{"GET", "/writing/one-binary", 200, `class="palette"`},
		{"GET", "/writing/one-binary", 200, "back-link"},
		{"GET", "/projects/quiet-machine", 200, "/_quire/img/"},
		{"GET", "/favicon.svg", 200, "<svg"},
		{"GET", "/", 200, `type="image/svg+xml"`},
		{"GET", "/_quire/style.css", 200, "site-narrow.css"},
		{"GET", "/", 200, "palette-picker"},
		{"GET", "/_quire/style.css", 200, `[data-palette="harbour"][data-theme="light"]`},
		{"GET", "/missing", 404, "never written"},
	},
	"4-feature": {
		{"GET", "/", 200, "Paper Moon"},
		{"GET", "/", 200, "figure--bleed"},
		{"GET", "/pieces", 200, "Tidal Drift"},
		{"GET", "/pieces", 200, "page--full"},
		{"GET", "/pieces/paper-moon", 200, "page--wide"},
		{"GET", "/_quire/style.css", 200, "--qr-measure-full"},
		{"GET", "/pieces/paper-moon", 200, "Add to basket"},
		{"GET", "/pieces/paper-moon", 200, "/_quire/img/"},
		{"POST", "/_basket/add?item=paper-moon", 303, ""},
		{"POST", "/_basket/add?item=nothing", 400, ""},
		{"GET", "/basket", 200, `basket-qty">1`},
		{"POST", "/_basket/dec?item=paper-moon", 303, ""},
		{"GET", "/basket", 200, "Nothing in it"},
		{"GET", "/admin/basket", 404, ""},
		{"SSE", "/_basket/events", 200, "datastar-patch-signals"},
		{"GET", "/_quire/style.css", 200, "basket-narrow.css"},
		{"GET", "/missing", 404, ""},
	},
	"several": {
		{"GET", "/", 200, `<a href="/notes">notes</a>`},
		{"GET", "/", 200, `href="/_quire/style.css`},
		{"GET", "/notes", 200, `<a href="/notes/first">the first</a>`},
		{"GET", "/notes", 200, `href="/_quire/notes/style.css`},
		{"GET", "/notes/first", 200, `<a href="/notes">the notes</a>`},
		{"GET", "/_quire/notes/style.css", 200, "@layer"},
		{"GET", "/_quire/health", 200, "ok"},
		{"GET", "/first", 404, ""},
		{"GET", "/notes/nothing", 404, ""},
	},
	"mounted": {
		{"GET", "/", 200, "The host's own page"},
		{"GET", "/", 200, `href="/docs/_quire/style.css`},
		{"GET", "/docs", 200, `<a href="/docs/install">installing it</a>`},
		{"GET", "/docs/install", 200, `<a href="/docs">the docs</a>`},
		{"GET", "/docs/_quire/style.css", 200, "@layer"},
		{"GET", "/install", 404, ""},
		{"GET", "/_quire/style.css", 404, ""},
	},
	"6-naked": {
		{"GET", "/", 200, `/posts/hello">Hello</a></li>`},
		{"GET", "/posts/hello", 200, "Some words"},
		{"GET", "/posts/hello", 200, "lx-number"},
		{"GET", "/_quire/style.css", 404, ""},
		{"GET", "/missing", 404, "Not found"},
	},
	"5-docs": {
		{"GET", "/", 200, "home-hero"},
		{"GET", "/", 200, "code-copy"},
		{"GET", "/", 200, "Below the cards"},
		{"GET", "/", 200, `<div class="card-grid">`},
		{"GET", "/", 200, "link-icon"},
		{"GET", "/", 200, "Built with Quire"},
		{"GET", "/docs", 200, "docs-label"},
		{"GET", "/docs/writing", 200, `aria-current="page"`},
		{"GET", "/docs/writing", 200, "<th>Field</th>"},
		{"GET", "/search?q=sidebar", 200, "search-hit"},
		{"GET", "/_quire/style.css", 200, "docs-wide.css"},
		{"GET", "/missing", 404, ""},
	},
}

// address finds what a page fetches or links to: a link, a source or a form's action, a source set, and what a Datastar action fetches.
var address = regexp.MustCompile(`(?:href|src|action|poster)="([^"]*)"|srcset="([^"]*)"|&#39;(/[^&]*)&#39;`)

type check struct {
	method, path string
	status       int
	needle       string
}

func build(t *testing.T, dir string, args ...string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "site")
	cmd := exec.Command("go", append(append([]string{"build", "-o", bin}, args...), ".")...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %v: %v\n%s", args, err, out)
	}
	return bin
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

// escapes is every address in a response that is the site's own and outside the root it was placed at.
func escapes(body []byte, root string) []string {
	var out []string
	for _, m := range address.FindAllSubmatch(body, -1) {
		for u := range strings.SplitSeq(cmp.Or(string(m[1]), string(m[2]), string(m[3])), ", ") {
			if strings.HasPrefix(u, "/") && !strings.HasPrefix(u, "//") && u != root && !strings.HasPrefix(u, root+"/") {
				out = append(out, u)
			}
		}
	}
	return out
}

func (c check) run(t *testing.T, client *http.Client, base, root string) {
	t.Helper()
	method, accept, wait := c.method, "text/html", 3*time.Second
	if method == "SSE" {
		method, accept, wait = http.MethodGet, "text/event-stream", time.Second
	}
	ctx, cancel := context.WithTimeout(t.Context(), wait)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, base+strings.TrimSuffix(root+c.path, "/"), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", accept)
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("%s %s: %v", c.method, c.path, err)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != c.status {
		t.Errorf("%s %s returned %d, want %d", c.method, c.path, resp.StatusCode, c.status)
	} else if !bytes.Contains(body, []byte(c.needle)) {
		t.Errorf("%s %s lacks %s", c.method, c.path, c.needle)
	} else if out := escapes(body, root); root != "" && len(out) > 0 {
		t.Errorf("%s %s under %s has addresses outside it: %v", c.method, c.path, root, out)
	}
}

func serve(t *testing.T, bin, cwd, root string, cs []check) {
	t.Helper()
	addr := freePort(t)
	var log bytes.Buffer
	args := []string{"--addr", addr}
	if root != "" {
		args = append(args, "--root", root)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = cwd, &log, &log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	defer func() {
		_ = cmd.Process.Kill()
		<-done
		if t.Failed() {
			t.Logf("server log:\n%s", log.String())
		}
	}()
	base := "http://" + addr
	for deadline := time.Now().Add(3 * time.Minute); ; time.Sleep(100 * time.Millisecond) {
		if r, err := http.Get(base); err == nil {
			_ = r.Body.Close()
			break
		}
		select {
		case <-done:
			t.Fatal("server exited")
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("server never became ready")
		}
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, c := range cs {
		c.run(t, client, base, root)
	}
}

func TestExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("builds, bakes and serves every example")
	}
	mains, err := filepath.Glob("*/main.go")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	quire := build(t, "../cmd/quire")
	for _, m := range mains {
		name := filepath.Dir(m)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cs, ok := checks[name]
			if !ok {
				t.Fatal("no checks for this example")
			}
			if name[0] < '0' || name[0] > '9' {
				serve(t, build(t, name), name, "", cs)
				return
			}
			dir := filepath.Join(t.TempDir(), name)
			if out, err := exec.Command(quire, "new", "--from", name, "--replace", root, dir).CombinedOutput(); err != nil {
				t.Fatalf("quire new: %v\n%s", err, out)
			}
			bin := build(t, dir)
			if !t.Run("bakes", func(t *testing.T) { serve(t, bin, dir, "", cs) }) {
				return
			}
			t.Run("placed", func(t *testing.T) { serve(t, bin, dir, "/nested", cs) })
			t.Run("ships", func(t *testing.T) {
				serve(t, build(t, dir, "-tags", "quire_nobake", "-overlay", ".quire/ship.json"), t.TempDir(), "", cs)
			})
		})
	}
}
