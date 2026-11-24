// Package quire is the helpers over Quire's parts. Run is a whole site in one call: it bakes, builds and serves. RunSites is several. Docs: https://quire.hiddenbridge.io
package quire

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lahickey/quire/bundle"
	"github.com/lahickey/quire/feature/reload"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base"
	"github.com/lahickey/quire/server"
	"github.com/lahickey/quire/site"
)

const closeWithin = 10 * time.Second

var shipped fs.FS

// Ship hands the binary the bundles a bake embedded in it. The ship.go a bake writes calls it.
func Ship(bundles fs.FS) { shipped = bundles }

// Bundle opens what a bake left for spec: from the binary if bundles were shipped in it, and from disk otherwise.
func Bundle(spec model.BakeSpec) (*bundle.Bundle, error) {
	dir := spec.WithDefaults().Bundle()
	if shipped == nil {
		return bundle.Open(os.DirFS(dir))
	}
	files, err := fs.Sub(shipped, filepath.ToSlash(dir))
	if err != nil {
		return nil, err
	}
	return bundle.Open(files)
}

// Site is one of several that a program runs in one server: who it is, what it declares, and where it is found and served.
type Site struct {
	Identity model.Identity
	Feature  site.Feature
	Options  []site.Option
	// Root is the path the site is served under. The first of several has none, and is the box's own.
	Root string
	// Content is the site's content folder: content if it names none.
	Content string
}

// spec is where the site's content is and where its bake goes: the first into .quire, and another into a folder of its own beneath it.
func (st Site) spec() model.BakeSpec {
	spec := model.BakeSpec{Content: st.Content}
	if st.Root != "" {
		spec.Dir = filepath.Join(model.QuireDir, "sites", st.Root)
	}
	return spec
}

// under is a place for the site at one path beneath a root, with what Quire serves for itself gathered under the root.
func under(root, at string) model.Place {
	if root+at == "" {
		return model.Place{}
	}
	return model.Place{Root: root + at, Engine: root + "/_quire" + at}
}

// serve starts the site's services and answers requests until the process is told to stop, then stops the services.
func serve(addr string, v *server.Server) error {
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()
	if err := v.Start(ctx); err != nil {
		return err
	}
	srv := &http.Server{
		Addr: addr, Handler: v, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 2 * time.Minute,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	failed := make(chan error, 1)
	go func() { failed <- srv.ListenAndServe() }()
	log.Println("listening on", addr)
	var err error
	select {
	case err = <-failed:
		if op, ok := errors.AsType[*net.OpError](err); ok && errors.Is(op.Err, syscall.EADDRINUSE) {
			err = fmt.Errorf("listen %s: already in use", addr)
		}
	case <-ctx.Done():
	}
	closing, cancel := context.WithTimeout(context.Background(), closeWithin)
	defer cancel()
	return errors.Join(err, srv.Shutdown(closing), v.Stop(closing))
}

// run bakes each site, builds it over the bundle that leaves, and serves them all until the process is interrupted or told to end.
func run(sites ...Site) {
	dev := flag.Bool("dev", false, "live reload")
	drafts := flag.Bool("drafts", false, "include draft documents")
	addr := flag.String("addr", ":8090", "listen address")
	root := flag.String("root", "", "the path everything is served under, as behind a proxy that hands it one")
	flag.Parse()
	baking := flag.Arg(0) == "bake"
	if flag.NArg() > 1 || (flag.NArg() == 1 && !baking) {
		log.Fatalf("usage: %s [flags] [bake]", os.Args[0])
	}
	var built []*site.Site
	for _, st := range sites {
		opts := append([]site.Option{site.WithDev(*dev), site.WithDrafts(*drafts), site.WithPlace(under(*root, st.Root))}, st.Options...)
		if *dev {
			opts = append(opts, site.WithFeatures(reload.New()))
		}
		d := site.Describe(st.Identity, st.Feature, opts...)
		if baking || (shipped == nil && canBake) {
			if err := Bake(d, st.spec()); err != nil {
				log.Fatal(err)
			}
		}
		if baking {
			continue
		}
		baked, err := Bundle(st.spec())
		if err != nil {
			log.Fatalf("bundle: %v; run the site's bake", err)
		}
		s, err := d.Build(baked)
		if err != nil {
			log.Fatal(err)
		}
		built = append(built, s)
	}
	if baking {
		return
	}
	handler, err := server.New(built...)
	if err != nil {
		log.Fatal(err)
	}
	if err := serve(*addr, handler); err != nil {
		log.Fatal(err)
	}
}

// Run bakes ./content into ./.quire with base's converters, builds the site over that bundle with base beneath it, and serves it.
// A binary handed a bundle by Ship, or built with quire_nobake, serves without baking.
func Run(id model.Identity, f site.Feature, extra ...site.Option) {
	RunSites(Site{Identity: id, Feature: f, Options: extra})
}

// RunBare is Run with no pack beneath the site: Quire's machinery and nothing of its look.
func RunBare(id model.Identity, f site.Feature, extra ...site.Option) {
	run(Site{Identity: id, Feature: f, Options: extra})
}

// RunSites is Run for several sites in one server, each with base beneath it and a content folder and a bundle of its own.
// The first is the box's own, at the top, and each of the others is served under its root.
func RunSites(sites ...Site) {
	for i, st := range sites {
		sites[i].Options = append([]site.Option{site.WithFeatures(base.Pack{})}, st.Options...)
	}
	run(sites...)
}
