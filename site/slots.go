package site

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"
	"github.com/lahickey/quire/internal/journal"
	"github.com/lahickey/quire/model"
)

func (s *Site) barePicture(pic model.Picture, sl model.PictureSlot) templ.Component {
	return bareImg(s.chrome.Scale.Attrs(pic, model.SizePage, false), sl.Alt)
}

func (s *Site) bareVideo(src string, poster model.Picture, _ model.PictureSlot) templ.Component {
	return bareVid(src, s.chrome.Scale.URL(poster, model.SizePage), poster)
}

func (s *Site) rendered(c templ.Component) (string, error) {
	var b strings.Builder
	err := c.Render(model.WithChrome(context.Background(), &s.chrome), &b)
	return b.String(), err
}

func (s *Site) fillPicture(sl model.PictureSlot) (string, error) {
	if journal.VideoType(sl.Name) == "" {
		pic, err := s.Picture(sl.Name)
		if err != nil {
			return "", err
		}
		return s.rendered(s.placePicture.v(pic, sl))
	}
	if !sl.Figure {
		return "", fmt.Errorf("%s: a video goes in a paragraph of its own", sl.Name)
	}
	file, err := s.from.File(sl.Name)
	if err != nil {
		return "", err
	}
	poster, err := s.Picture(strings.TrimSuffix(sl.Name, path.Ext(sl.Name)) + ".png")
	if err != nil {
		return "", fmt.Errorf("%s needs a still of the same name as its poster: %w", sl.Name, err)
	}
	return s.rendered(s.placeVideo.v(model.BundleURL(file), poster, sl))
}

func (s *Site) fill(sl model.Slot) (string, error) {
	f := sl.Fence
	if f.Name == "" {
		return s.fillPicture(sl.Picture)
	}
	w := s.widgets[0][f.Name]
	if w == nil {
		w = s.widgets[1][f.Name]
	}
	switch {
	case w == nil:
		return f.HTML, nil
	case f.Body == "":
		return "", fmt.Errorf("widget %q has the name of a language, so the bake highlighted its fence as code: rename the widget and its fences", f.Name)
	}
	out, err := w(f.Info, f.Body)
	if err != nil {
		return "", fmt.Errorf("%s: %w", f.Name, err)
	}
	return out, nil
}

func (s *Site) declaring(what string) bool {
	mustBeOpen(s.built, what)
	if s.read {
		s.fail("%s: set after content was read, so documents already loaded lack it; set it first", what)
	}
	return !s.read
}

// Picture is a picture of the site's content, by its master's name under the content root.
func (s *Site) Picture(name string) (model.Picture, error) {
	name = path.Clean(filepath.ToSlash(name))
	if pic, ok := s.pictures[name]; ok {
		return pic, nil
	}
	pic, err := s.from.Picture(name)
	if err != nil {
		return pic, err
	}
	base := path.Base(name)
	pic.Name = strings.TrimSuffix(base, path.Ext(base))
	s.pictures[name] = pic
	return pic, nil
}

// Converted is the output of the named converter for a body and its arguments, as the bundle file to link to.
func (s *Site) Converted(converter string, body []byte, args ...string) (string, error) {
	return s.from.Converted(converter, body, args...)
}

// Body reads a bundle file, to put its contents in a page.
func (s *Site) Body(file string) ([]byte, error) { return s.from.Body(file) }

// Baked says the site has a source behind it. One built over nothing has no converter's output to ask for.
func (s *Site) Baked() bool { return s.baked }

// FontSetID names the font set the site draws in, for a converter that measures text.
func (s *Site) FontSetID() string { return s.fonts.ID }

// SetWidget fills every fence of that name, in any document of the site. The site's own wins over a pack's of the same name.
func (s *Site) SetWidget(name string, w model.Widget) {
	if s.owner != "" && name != s.owner && !strings.HasPrefix(name, s.owner+"-") {
		s.fail("widget %s is outside the %s- namespace", name, s.owner)
	}
	if s.declaring("widget " + name) {
		s.widgets[min(s.depth, 1)][name] = w
	}
}

// SetPicturePlacer draws a picture in a document's body. Without one it is a bare img.
func (s *Site) SetPicturePlacer(place func(model.Picture, model.PictureSlot) templ.Component) {
	if s.declaring("the picture placer") {
		s.placePicture.set(place, s.own("the picture placer"))
	}
}

// SetVideoPlacer draws a video in a document's body, from its address and its poster. Without one it is a bare video.
func (s *Site) SetVideoPlacer(place func(src string, poster model.Picture, sl model.PictureSlot) templ.Component) {
	if s.declaring("the video placer") {
		s.placeVideo.set(place, s.own("the video placer"))
	}
}
