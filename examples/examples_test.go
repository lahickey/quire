package examples

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os/exec"
	"path/filepath"
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
		{"GET", "/missing", 404, "never written"},
	},
	"4-plugin": {
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
		{"SSE", "/_basket/events", 200, "datastar-patch-signals"},
		{"GET", "/_quire/style.css", 200, "basket-narrow.css"},
		{"GET", "/missing", 404, ""},
	},
}

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

func (c check) run(t *testing.T, client *http.Client, base string) {
	t.Helper()
	method, accept, wait := c.method, "text/html", 3*time.Second
	if method == "SSE" {
		method, accept, wait = http.MethodGet, "text/event-stream", time.Second
	}
	ctx, cancel := context.WithTimeout(t.Context(), wait)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, base+c.path, nil)
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
	}
}

func serve(t *testing.T, bin, cwd string, cs []check) {
	t.Helper()
	addr := freePort(t)
	var log bytes.Buffer
	cmd := exec.Command(bin, "--addr", addr)
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
		if r, err := http.Get(base + "/_quire/health"); err == nil {
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
		c.run(t, client, base)
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
			dir := filepath.Join(t.TempDir(), name)
			if out, err := exec.Command(quire, "new", "--from", name, "--replace", root, dir).CombinedOutput(); err != nil {
				t.Fatalf("quire new: %v\n%s", err, out)
			}
			if !t.Run("bakes", func(t *testing.T) { serve(t, build(t, dir), dir, cs) }) {
				return
			}
			t.Run("ships", func(t *testing.T) {
				serve(t, build(t, dir, "-tags", "quire_nobake", "-overlay", ".quire/ship.json"), t.TempDir(), cs)
			})
		})
	}
}
