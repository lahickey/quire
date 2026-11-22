// Package quire is the helpers over Quire's parts. Run is a whole site in one call: it bakes, builds and serves. Docs: https://quire.hiddenbridge.io
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

// Ship hands Run a bundle embedded in the binary. The ship.go a bake writes calls it.
func Ship(files fs.FS, dir string) {
	b, err := fs.Sub(files, dir)
	if err != nil {
		panic("quire: ship: " + err.Error())
	}
	shipped = b
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

// Run bakes ./content into ./.quire with base's converters, builds the site over that bundle with base beneath it, and serves it.
// A binary handed a bundle by Ship, or built with quire_nobake, serves without baking.
func Run(id model.Identity, f site.Feature, extra ...site.Option) {
	RunBare(id, f, append([]site.Option{site.WithFeatures(base.Pack{})}, extra...)...)
}

// RunBare is Run with no pack beneath the site: Quire's machinery and nothing of its look.
// It serves until the process is interrupted or told to end.
func RunBare(id model.Identity, f site.Feature, extra ...site.Option) {
	dev := flag.Bool("dev", false, "live reload")
	drafts := flag.Bool("drafts", false, "include draft documents")
	addr := flag.String("addr", ":8090", "listen address")
	root := flag.String("root", "", "the path the site is served under, as behind a proxy that hands it one")
	flag.Parse()
	baking := flag.Arg(0) == "bake"
	if flag.NArg() > 1 || (flag.NArg() == 1 && !baking) {
		log.Fatalf("usage: %s [flags] [bake]", os.Args[0])
	}
	opts := append([]site.Option{site.WithDev(*dev), site.WithDrafts(*drafts), site.WithPlace(model.Place{Root: *root, Engine: *root})}, extra...)
	if *dev {
		opts = append(opts, site.WithFeatures(reload.New()))
	}
	d := site.Describe(id, f, opts...)
	if baking || (shipped == nil && canBake) {
		if err := bakeSite(d); err != nil {
			log.Fatal(err)
		}
		if baking {
			return
		}
	}
	files := shipped
	if files == nil {
		files = os.DirFS(model.BakeSpec{}.WithDefaults().Bundle())
	}
	baked, err := bundle.Open(files)
	if err != nil {
		log.Fatalf("bundle: %v; run the site's bake", err)
	}
	built, err := d.Build(baked)
	if err != nil {
		log.Fatal(err)
	}
	handler, err := server.New(built)
	if err != nil {
		log.Fatal(err)
	}
	if err := serve(*addr, handler); err != nil {
		log.Fatal(err)
	}
}
