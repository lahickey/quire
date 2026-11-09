package model

import (
	"net/url"
	"strings"

	"github.com/a-h/templ"
)

type Theme int

const (
	ThemeLight Theme = iota
	ThemeDark
)

type Icon struct{ Rel, Type, Href string }

type Site struct {
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

type NavItem struct {
	Label string
	Href  string
}

type LiveReload struct {
	On     bool
	BootID string
	Init   string
}

type Chrome struct {
	Site

	Theme       Theme
	Header      func(Chrome, PageSpec) templ.Component
	Footer      func(Chrome, PageSpec) templ.Component
	ScriptURL   string
	ModuleURLs  []string
	NavEnd      []templ.Component
	StyleURLs   []string
	FontPreload []string
	LiveReload  LiveReload
	OGImageURL  string
	Domain      string
	Nav         []NavItem
	Sections    []NavItem
	Icons       []Icon
}

type Shell = func(PageSpec) templ.Component

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

func (s Site) AbsURL(path string) string { return absURL(s.BaseURL, path) }

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
		return p.ShareImage
	}
	return c.OGImageURL
}

func (s Site) Host() string { return hostOf(s.BaseURL) }

func (s Site) WithDefaults() Site {
	s.Dates = s.Dates.orDefault()
	s.BaseURL = strings.TrimRight(s.BaseURL, "/")
	return s
}
