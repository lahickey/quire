package main

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/lahickey/quire/internal/fonts"
	"github.com/lahickey/quire/internal/journal"
)

const (
	text       = "Inter:wght@100..900"
	mono       = "JetBrains Mono:wght@100..800"
	webDir     = "internal/fonts/web"
	measureDir = "internal/bake/diagram/fonts"
)

var subsets = []string{"latin", "latin-ext"}

const (
	modernUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36"
	oldUA    = "Mozilla/5.0 (Windows NT 6.1)"
)

var (
	faceBlock = regexp.MustCompile(`/\*\s*([\w-]+)\s*\*/\s*@font-face\s*\{([^}]*)\}`)
	faceProp  = regexp.MustCompile(`([\w-]+)\s*:\s*([^;]+);`)
	srcURL    = regexp.MustCompile(`url\(([^)]+)\)`)
)

func get(u, ua string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", u, res.Status)
	}
	return io.ReadAll(res.Body)
}

func googleCSS(spec, ua string) ([]byte, error) {
	return get("https://fonts.googleapis.com/css2?family="+url.QueryEscape(spec), ua)
}

func props(block string) map[string]string {
	out := map[string]string{}
	for _, m := range faceProp.FindAllStringSubmatch(block, -1) {
		out[m[1]] = strings.TrimSpace(m[2])
	}
	return out
}

func slug(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

func fetchTo(u, ua, dest string) error {
	b, err := get(u, ua)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, b, 0o644)
}

func fetchFamily(spec string, subsets []string, webDir, measureDir string) (fonts.Family, []byte, error) {
	name, _, _ := strings.Cut(spec, ":")
	f := fonts.Family{Name: name, Measure: map[int]string{}}
	css, err := googleCSS(spec, modernUA)
	if err != nil {
		return f, nil, err
	}
	for _, m := range faceBlock.FindAllStringSubmatch(string(css), -1) {
		subset, p := m[1], props(m[2])
		if !slices.Contains(subsets, subset) {
			continue
		}
		u := srcURL.FindStringSubmatch(p["src"])
		if u == nil {
			return f, nil, fmt.Errorf("%s %s: no src", name, subset)
		}
		face := fonts.Face{
			File:   slug(name) + "-" + subset + ".woff2",
			Subset: subset,
			Weight: p["font-weight"],
			Style:  p["font-style"],
			Range:  p["unicode-range"],
		}
		if err := fetchTo(u[1], modernUA, filepath.Join(webDir, face.File)); err != nil {
			return f, nil, err
		}
		f.Faces = append(f.Faces, face)
	}
	if len(f.Faces) == 0 {
		return f, nil, fmt.Errorf("%s: none of %v offered", name, subsets)
	}

	var measured []byte
	for _, w := range []int{400, 700} {
		css, err := googleCSS(name+":wght@"+strconv.Itoa(w), oldUA)
		if err != nil {
			return f, nil, err
		}
		u := srcURL.FindStringSubmatch(string(css))
		if u == nil {
			return f, nil, fmt.Errorf("%s %d: no TTF", name, w)
		}
		file := slug(name) + "-" + strconv.Itoa(w) + ".ttf"
		ttf, err := get(u[1], oldUA)
		if err != nil {
			return f, nil, err
		}
		if err := os.WriteFile(filepath.Join(measureDir, file), ttf, 0o644); err != nil {
			return f, nil, err
		}
		f.Measure[w] = file
		measured = append(measured, ttf...)
	}
	return f, measured, nil
}

func fetchLicence(name, dest string) error {
	for _, l := range []string{"ofl/OFL.txt", "apache/LICENSE.txt", "ufl/UFL.txt"} {
		dir, file, _ := strings.Cut(l, "/")
		u := "https://raw.githubusercontent.com/google/fonts/main/" + dir + "/" +
			strings.ReplaceAll(slug(name), "-", "") + "/" + file
		if b, err := get(u, modernUA); err == nil {
			return os.WriteFile(dest, b, 0o644)
		}
	}
	return fmt.Errorf("%s: no licence found on github.com/google/fonts", name)
}

func main() {
	log.SetFlags(0)
	for _, d := range []string{webDir, measureDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Fatal(err)
		}
	}

	var set fonts.FontSet
	var measured bytes.Buffer
	for _, f := range []struct {
		spec string
		into *fonts.Family
	}{{text, &set.Text}, {mono, &set.Mono}} {
		fam, ttfs, err := fetchFamily(f.spec, subsets, webDir, measureDir)
		if err != nil {
			log.Fatal(err)
		}
		*f.into = fam
		measured.Write(ttfs)

		lic := "LICENSE-" + slug(fam.Name) + ".txt"
		for _, d := range []string{webDir, measureDir} {
			if err := fetchLicence(fam.Name, filepath.Join(d, lic)); err != nil {
				log.Fatal(err)
			}
		}
		log.Printf("%-16s %d faces, measured at %v", fam.Name, len(fam.Faces), slices.Sorted(maps.Keys(fam.Measure)))
	}
	set.ID = journal.Hash(measured.Bytes())
	b, err := json.Marshal(set, json.Deterministic(true))
	if err != nil {
		log.Fatal(err)
	}

	for _, d := range []string{webDir, measureDir} {
		if err := os.WriteFile(filepath.Join(d, fonts.FontsName), append(b, '\n'), 0o644); err != nil {
			log.Fatal(err)
		}
	}
}
