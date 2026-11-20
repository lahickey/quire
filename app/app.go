// Package app is the helpers over Quire's parts. Run is a whole site in one call: it bakes, builds and serves.
package app

import (
	"errors"
	"flag"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/lahickey/quire"
	"github.com/lahickey/quire/bundle"
	"github.com/lahickey/quire/model"
)

var shipped fs.FS

// Ship hands Run a bundle embedded in the binary. The ship.go a bake writes calls it.
func Ship(files fs.FS, dir string) {
	b, err := fs.Sub(files, dir)
	if err != nil {
		panic("quire: ship: " + err.Error())
	}
	shipped = b
}

// Run bakes ./content into ./.quire with base's converters, builds the site over that bundle and serves it.
// A binary handed a bundle by Ship, or built with quire_nobake, serves without baking.
func Run(id model.Identity, b quire.Blueprint, extra ...quire.Option) {
	dev := flag.Bool("dev", false, "live reload")
	drafts := flag.Bool("drafts", false, "include draft documents")
	addr := flag.String("addr", ":8090", "listen address")
	flag.Parse()
	baking := flag.Arg(0) == "bake"
	if flag.NArg() > 1 || (flag.NArg() == 1 && !baking) {
		log.Fatalf("usage: %s [flags] [bake]", os.Args[0])
	}
	opts := append([]quire.Option{quire.WithDev(*dev), quire.WithDrafts(*drafts)}, extra...)
	if baking || (shipped == nil && canBake) {
		if err := bakeSite(id, b, opts); err != nil {
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
	site, err := quire.Build(id, baked, b, opts...)
	if err != nil {
		log.Fatal(err)
	}
	server, err := quire.NewServer(site)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("listening on", *addr)
	srv := &http.Server{Addr: *addr, Handler: server, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 2 * time.Minute}
	err = srv.ListenAndServe()
	if op, ok := errors.AsType[*net.OpError](err); ok && errors.Is(op.Err, syscall.EADDRINUSE) {
		log.Fatalf("listen %s: already in use", *addr)
	}
	log.Fatal(err)
}
