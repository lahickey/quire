// Package base is Quire's default kind of site: what a document's fences and pictures become, how they look, and the card a page shares.
package base

import (
	"encoding/json/v2"
	"fmt"

	"github.com/a-h/templ"
	"github.com/lahickey/quire"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base/assets"
	"github.com/lahickey/quire/pack/base/fonts"
	"github.com/lahickey/quire/pack/base/ui"
)

// Pack is base as a blueprint. It comes before a site's own, since what it sets has to be in place before content is read.
type Pack struct{}

func shareCard(site *quire.Site) error {
	if !site.Baked() {
		return nil
	}
	body, err := json.Marshal(site.ShareCard())
	if err != nil {
		return err
	}
	file, err := site.Converted("card", body, site.FontSetID())
	if err != nil {
		return fmt.Errorf("share card: %w", err)
	}
	site.SetShareImage(file)
	return nil
}

func (Pack) Declare(site *quire.Site) error {
	site.SetWidget("mermaid", mermaid(site))
	site.SetWidget("cards", cards)
	site.SetWidget("palette", palette)
	site.SetPicturePlacer(picture)
	site.SetVideoPlacer(video)
	c := site.Chrome()
	site.SetFonts(fonts.Web())
	site.AddBaseCSS("scale", assets.ScaleCSS(c.Scale))
	site.AddBaseCSS("palette", assets.PaletteCSS(site.Palette(), c.Palettes))
	for _, name := range assets.Sheets {
		sheet, err := assets.Files.ReadFile(name)
		if err != nil {
			return err
		}
		site.AddBaseCSS(name, sheet)
	}
	site.AddBaseCSS("breakpoint", assets.BreakpointCSS(c.Scale))
	site.AddScripts(assets.Files, assets.Scripts...)
	site.SetDatastar(true)
	site.SetFrame(ui.Frame)
	site.SetDefaultHome(ui.Home())
	site.SetNotFound(ui.NotFound())
	return shareCard(site)
}

// Pages gives a collection the default look's two pages: a listing at its address and an article for each document.
// A collection with nothing in it, as when every document is a draft, gets neither.
func Pages[T model.Metadata](site *quire.Site, c *model.Collection[T]) {
	if c.Len() == 0 {
		return
	}
	site.AddPage(c.Page(), ui.Listing(c))
	site.AddDocumentPages(c, func(d model.Document[T]) (model.PageSpec, templ.Component) {
		return model.DocumentPage(c, d), ui.Article(c, d)
	})
}
