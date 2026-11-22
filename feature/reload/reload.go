// Package reload is live reload for a site in dev: a page that finds the server restarted loads itself again.
package reload

import (
	"crypto/rand"
	"net/http"

	"github.com/lahickey/quire/libs/streams"
	"github.com/lahickey/quire/site"
)

type Feature struct{ boot string }

func New() Feature { return Feature{rand.Text()} }

func (Feature) Name() string { return "reload" }

func (f Feature) Declare(s *site.Site) error {
	live := s.Streams()
	s.AddBodyEnd(watch(live, f.boot))
	s.AddRoute("GET", s.URL("boot"), func(w http.ResponseWriter, r *http.Request) {
		live.Subscribe(w, r, "boot", func(st *streams.Stream) error { return st.Signals(map[string]string{"boot": f.boot}) })
	})
	return nil
}
