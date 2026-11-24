// Package base is Quire's default look: what a document's fences and pictures become, how they look, and the card a page shares.
package base

import (
	"encoding/json/v2"
	"fmt"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/feature/content"
	"github.com/lahickey/quire/model"
	"github.com/lahickey/quire/pack/base/assets"
	"github.com/lahickey/quire/pack/base/fonts"
	"github.com/lahickey/quire/pack/base/ui"
	"github.com/lahickey/quire/site"
)

// Pack is base as a feature with no name: it is the site's look. It comes before a site's own, since what it sets has to be in place before content is read.
type Pack struct{}

func shareCard(s *site.Site) error {
	if !s.Baked() {
		return nil
	}
	body, err := json.Marshal(s.ShareCard())
	if err != nil {
		return err
	}
	file, err := s.Converted("card", body)
	if err != nil {
		return fmt.Errorf("share card: %w", err)
	}
	s.SetShareImage(file)
	return nil
}

func (Pack) Name() string { return "" }

func (Pack) Declare(s *site.Site) error {
	s.SetWidget("mermaid", mermaid(s))
	s.SetWidget("cards", cards)
	s.SetWidget("palette", palette)
	s.SetPicturePlacer(picture)
	s.SetVideoPlacer(video)
	c := s.Chrome()
	s.SetFonts(fonts.Web())
	s.AddBaseCSS("scale", assets.ScaleCSS(c.Scale))
	s.AddBaseCSS("palette", assets.PaletteCSS(s.Palette(), c.Palettes))
	for _, name := range assets.Sheets {
		sheet, err := assets.Files.ReadFile(name)
		if err != nil {
			return err
		}
		s.AddBaseCSS(name, sheet)
	}
	s.AddBaseCSS("breakpoint", assets.BreakpointCSS(c.Scale))
	s.AddScripts(assets.Files, assets.Scripts...)
	s.UseDatastar()
	s.SetFrame(ui.Frame)
	s.SetDefaultHome(ui.Home())
	s.SetNotFound(ui.NotFound())
	return shareCard(s)
}

// Content is a whole site from the content folder, in the default look: index.md is the home page, a file beside it is a page, and a folder is a collection.
func Content() site.Feature {
	return content.Pages{Page: ui.Page, Listing: ui.Listing, Document: ui.Article}
}

// Pages gives a collection the default look's two pages: a listing at its address and an article for each document.
func Pages(s *site.Site, c *model.Collection) {
	s.AddCollectionPages(c, ui.Listing(c), func(d model.Document) templ.Component { return ui.Article(c, d) })
}
