package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"
)

const (
	keepalive = 25 * time.Second
	writeWait = 5 * time.Second
)

type Stream struct {
	mu    sync.Mutex
	w     io.Writer
	rc    *http.ResponseController
	scope string
}

type Streams struct {
	mu    sync.Mutex
	subs  map[string]map[*Stream]struct{}
	scope string
}

func (s *Stream) write(payload string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.rc.SetWriteDeadline(time.Now().Add(writeWait))
	if _, err := io.WriteString(s.w, payload); err != nil {
		return err
	}
	return s.rc.Flush()
}

func (s *Stream) event(name, key, payload string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "event: %s\n", name)
	for line := range strings.SplitSeq(payload, "\n") {
		fmt.Fprintf(&b, "data: %s %s\n", key, line)
	}
	b.WriteString("\n")
	return s.write(b.String())
}

func (s *Streams) add(topic string, st *Stream) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.subs[topic] == nil {
		s.subs[topic] = map[*Stream]struct{}{}
	}
	s.subs[topic][st] = struct{}{}
}

func (s *Streams) remove(topic string, st *Stream) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subs[topic], st)
	if len(s.subs[topic]) == 0 {
		delete(s.subs, topic)
	}
}

func (m *Mount) NewStreams() *Streams {
	return &Streams{subs: map[string]map[*Stream]struct{}{}, scope: m.got.Name}
}

func scoped(scope string, v any) ([]byte, error) {
	return json.Marshal(map[string]any{scope: v})
}

func openStream(w http.ResponseWriter, scope string) *Stream {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	s := &Stream{w: w, rc: http.NewResponseController(w), scope: scope}
	_ = s.rc.Flush()
	return s
}

func (s *Stream) Signals(v any) error {
	b, err := scoped(s.scope, v)
	if err != nil {
		return err
	}
	return s.event("datastar-patch-signals", "signals", string(b))
}

func (s *Stream) Elements(ctx context.Context, c templ.Component) error {
	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		return err
	}
	return s.event("datastar-patch-elements", "elements", strings.TrimRight(buf.String(), "\n"))
}

func (s *Streams) Publish(topic string, send func(*Stream) error) {
	s.mu.Lock()
	open := slices.Collect(maps.Keys(s.subs[topic]))
	s.mu.Unlock()
	var wg sync.WaitGroup
	for _, st := range open {
		wg.Go(func() { _ = send(st) })
	}
	wg.Wait()
}

func (s *Streams) Subscribe(w http.ResponseWriter, r *http.Request, topic string, send func(*Stream) error) {
	st := openStream(w, s.scope)
	s.add(topic, st)
	defer s.remove(topic, st)
	if err := send(st); err != nil {
		return
	}
	tick := time.NewTicker(keepalive)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			if err := st.write(":\n\n"); err != nil {
				return
			}
		}
	}
}

func Done(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Datastar-Request") != "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	back := "/"
	if u, err := url.Parse(r.Header.Get("Referer")); err == nil && strings.HasPrefix(u.Path, "/") && (u.Host == "" || u.Host == r.Host) {
		back = u.Path
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}
