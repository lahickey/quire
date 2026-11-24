package model

import (
	"cmp"
	"context"
	"log/slog"
	"net/url"
	"slices"
	"strings"

	"github.com/a-h/templ"
)

type Theme int

const (
	ThemeLight Theme = iota
	ThemeDark
)

var BuiltWithQuire = Link{Label: "Built with Quire", URL: "https://quire.hiddenbridge.io"}

type Icon struct{ Rel, Type, Href string }

type Identity struct {
	Name        string
	Author      string
	Description string
	BaseURL     string
	Dates       Dates
}

type ArticleSEO struct {
	Published string
	Modified  string
	Draft     bool
}

type Link struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
	Badge string `yaml:"-"`
}

// ShareCard is what the picture a page shares says.
type ShareCard struct {
	Kicker  string
	Title   string
	Lines   []string
	Domain  string
	Palette Palette
}

type NavItem struct {
	Label string
	Href  string
}

// EnginePath is what Quire serves for itself: assets, the bundle's files, the health check.
const (
	EnginePath = "/_quire/"
	engineTop  = "/_quire"
)

// Place is where a site is put in the address space it is served from. A site writes its own addresses as if it were at /,
// and they are placed on the way in and on the way out. Root goes in front of its pages, its features and its groups.
// Engine is where what the site has under EnginePath goes. Left empty that is /_quire, the top, which is right for a site alone.
// Each of several sites on a box takes a part of it, /_quire/shop, and a site in a host's address space takes one under its root.
type Place struct{ Root, Engine string }

// Env is what is fixed for the life of a running site. A service is told it once, when it starts.
type Env struct {
	Identity
	Place
	Dev bool
	Log *slog.Logger
}

type Chrome struct {
	Identity
	Place

	Brand       string
	Theme       Theme
	Palettes    []Palette
	FooterText  string
	FooterLinks []Link
	ScriptURL   string
	ModuleURLs  []string
	NavEnd      []templ.Component
	StyleURLs   []string
	FontPreload []string
	OGImageURL  string
	Nav         []NavItem
	Icons       []Icon
	Scale       Scale
}

type chromeKey struct{}

type PageSpec struct {
	Title       string
	Description string
	Path        string
	NoIndex     bool
	ShareImage  string
	Article     *ArticleSEO
}

func hostOf(base string) string {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://")
	}
	return u.Host
}

func absURL(base, path string) string {
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "//") {
		return path
	}
	if base == "" {
		base = "http://localhost"
	}
	if path == "" || path == "/" {
		return base + "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func (i Icon) Attrs() templ.Attributes {
	a := templ.Attributes{"rel": i.Rel, "href": i.Href}
	if i.Type != "" {
		a["type"] = i.Type
	}
	return a
}

func (t Theme) String() string {
	if t == ThemeDark {
		return "dark"
	}
	return "light"
}

func (c Chrome) Sections() []NavItem {
	return slices.DeleteFunc(slices.Clone(c.Nav), func(n NavItem) bool { return n.Href == "/" })
}

func (s Identity) AbsURL(path string) string { return absURL(s.BaseURL, path) }

func (p PageSpec) Canonical(c Chrome) string {
	if p.Path == "" || p.NoIndex || (p.Article != nil && p.Article.Draft) {
		return ""
	}
	return c.AbsURL(p.Path)
}

func (p PageSpec) MetaDescription(c Chrome) string {
	if p.Description != "" {
		return p.Description
	}
	return c.Description
}

func (p PageSpec) OGImage(c Chrome) string {
	if p.ShareImage != "" {
		return c.AbsURL(p.ShareImage)
	}
	return c.OGImageURL
}

// WithChrome gives a render the site it belongs to. A site does this for everything it draws.
func WithChrome(ctx context.Context, c *Chrome) context.Context {
	return context.WithValue(ctx, chromeKey{}, c)
}

// ChromeOf is the site a render belongs to: its identity, navigation, scale and the rest. It is false for a render no site began.
func ChromeOf(ctx context.Context) (Chrome, bool) {
	c, ok := ctx.Value(chromeKey{}).(*Chrome)
	if !ok {
		return Chrome{}, false
	}
	return *c, true
}

func (s Identity) Host() string { return hostOf(s.BaseURL) }

// Out is a site's own address as it is served.
func (p Place) Out(path string) string {
	switch {
	case strings.HasPrefix(path, EnginePath):
		return cmp.Or(p.Engine, engineTop) + path[len(engineTop):]
	case path == "/" && p.Root != "":
		return p.Root
	}
	return p.Root + path
}

// In is a served address as the site wrote it. One outside the place is left as it is.
func (p Place) In(path string) string {
	if rest, ok := strings.CutPrefix(path, cmp.Or(p.Engine, engineTop)+"/"); ok {
		return EnginePath + rest
	}
	if rest, ok := strings.CutPrefix(path, p.Root); ok && (rest == "" || rest[0] == '/') {
		return cmp.Or(rest, "/")
	}
	return path
}

// AbsURL is a site's own address in full, where it is served.
func (c Chrome) AbsURL(path string) string {
	if strings.HasPrefix(path, "/") {
		path = c.Out(path)
	}
	return c.Identity.AbsURL(path)
}

func (s Identity) WithDefaults() Identity {
	s.Dates = s.Dates.orDefault()
	s.BaseURL = strings.TrimRight(s.BaseURL, "/")
	return s
}
