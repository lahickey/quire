package main

import (
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
)

func mount(e *quire.Engine) error {
	_, err := e.AddCollection(model.Basic{ID: "docs", Title: "Documentation"})
	return err
}

func main() {
	quire.Run(model.Site{
		Name:        "Quire",
		BaseURL:     "https://quire.hiddenbridge.io",
		Description: "A Go engine that turns a folder of markdown into a finished website, served from a single binary.",
	}, mount)
}
