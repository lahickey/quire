package assets

import (
	"bytes"
	"regexp"
	"strconv"
	"testing"

	"github.com/lahickey/quire/model"
)

func TestOwnSheetsLeaveTheScaleToScale(t *testing.T) {
	sc := model.DefaultScale()
	for _, name := range ownSheets {
		body, err := files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(body, []byte("@media (max-width:")) || bytes.Contains(body, []byte("@media (width <")) {
			t.Errorf("%s writes a width media query; Scale emits the only one", name)
		}
		for _, banned := range []string{
			"--qr-col:", "--qr-measure:", "--qr-measure-wide:", "--qr-measure-full:", "--qr-text-root:",
			"--qr-gutter:", "--qr-thumb:",
			"--qr-nav-columns:", "--qr-nav-links-column:", "--qr-nav-end-column:", "--qr-nav-justify:",
		} {
			if bytes.Contains(body, []byte(banned)) {
				t.Errorf("%s redeclares %s, which the scale owns", name, banned)
			}
		}
		for _, m := range regexp.MustCompile(`(\d+)px`).FindAllSubmatch(body, -1) {
			n, _ := strconv.Atoi(string(m[1]))
			switch n {
			case sc.Root, sc.Col, sc.Measure(), sc.Wide(), sc.Full(), sc.Gutter(), sc.Thumb():
				t.Errorf("%s restates %dpx, which the scale owns", name, n)
			}
		}
	}
}
