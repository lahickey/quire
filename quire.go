// Package quire turns a folder of markdown into a website served from one binary. Docs: https://quire.hiddenbridge.io
package quire

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/lahickey/quire/internal/serve"
	"github.com/lahickey/quire/model"
)

const (
	imageMemory      = 12 << 20
	imageMemoryLimit = 64 << 10
)

var shipped fs.FS

func (s *settings) serve(bundle fs.FS, b Blueprint) (*Server, error) {
	store := serve.NewStore(imageMemory, imageMemoryLimit, s.dev, s.fonts.ID)
	site := s.site(store)
	if bundle != nil {
		if err := store.Load(bundle); err != nil {
			return nil, fmt.Errorf("bundle: %w; run the site's bake", err)
		}
		if err := site.addSiteCard(); err != nil {
			return nil, err
		}
	}
	if err := site.build(b); err != nil {
		return nil, err
	}
	return newServer(site, store)
}

// New serves a blueprint with no bundle behind it, which is what a test of a blueprint or a plugin needs.
func New(id model.Identity, b Blueprint, opts ...Option) (*Server, error) {
	return configure(id, opts...).serve(nil, b)
}

func Ship(files fs.FS, dir string) {
	b, err := fs.Sub(files, dir)
	if err != nil {
		panic("quire: ship: " + err.Error())
	}
	shipped = b
}

func Run(id model.Identity, b Blueprint, extra ...Option) {
	dev := flag.Bool("dev", false, "live reload")
	drafts := flag.Bool("drafts", false, "include draft documents")
	addr := flag.String("addr", ":8090", "listen address")
	flag.Parse()
	baking := flag.Arg(0) == "bake"
	if flag.NArg() > 1 || (flag.NArg() == 1 && !baking) {
		log.Fatalf("usage: %s [flags] [bake]", os.Args[0])
	}
	s := configure(id, append([]Option{WithDev(*dev), WithDrafts(*drafts)}, extra...)...)
	if baking || (shipped == nil && canBake) {
		if err := bakeSite(s, b); err != nil {
			log.Fatal(err)
		}
		if baking {
			return
		}
	}
	bundle := shipped
	if bundle == nil {
		bundle = os.DirFS(s.bakeSpec.WithDefaults().Bundle())
	}
	v, err := s.serve(bundle, b)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("listening on", *addr)
	srv := &http.Server{Addr: *addr, Handler: v, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 2 * time.Minute}
	err = srv.ListenAndServe()
	if op, ok := errors.AsType[*net.OpError](err); ok && errors.Is(op.Err, syscall.EADDRINUSE) {
		log.Fatalf("listen %s: already in use", *addr)
	}
	log.Fatal(err)
}
