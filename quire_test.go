package quire

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

const q = "github.com/lahickey/quire"

func TestServingLinksNoBake(t *testing.T) {
	heavy := []string{q + "/bake", q + "/pack/base/convert", "github.com/yuin/goldmark", q + "/libs/lexis"}
	for what, args := range map[string][]string{
		"the serving parts":             {"./site", "./server", "./libs/streams", "./feature/..."},
		"quire built with quire_nobake": {"-tags", "quire_nobake", "."},
	} {
		out, err := exec.Command("go", append([]string{"list", "-deps"}, args...)...).Output()
		if err != nil {
			t.Fatal(err)
		}
		for _, dep := range strings.Fields(string(out)) {
			if slices.ContainsFunc(heavy, func(pkg string) bool { return strings.HasPrefix(dep, pkg) }) {
				t.Errorf("%s links %s", what, dep)
			}
		}
	}
}

// The tree, lowest first: a package imports nothing above its own tier.
func tier(pkg string) int {
	for i, prefixes := range [][]string{
		{"/model", "/service/"},
		{"/bundle"},
		{"/site"},
		{"/pack/", "/feature/"},
		{"/server"},
		{""},
	} {
		for _, p := range prefixes {
			if pkg == q+p || p != "" && strings.HasPrefix(pkg, q+p) && (strings.HasSuffix(p, "/") || strings.HasPrefix(pkg, q+p+"/")) {
				return i
			}
		}
	}
	return -1
}

func TestLayers(t *testing.T) {
	out, err := exec.Command("go", "list", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", "./...").Output()
	if err != nil {
		t.Fatal(err)
	}
	may := map[string]func(string) bool{
		q + "/model":        func(p string) bool { return p == q+"/model/radix" },
		q + "/bundle":       func(p string) bool { return p == q+"/model" || p == q+"/internal/journal" },
		q + "/pack/base/ui": func(p string) bool { return p == q+"/model" },
	}
	for line := range strings.Lines(string(out)) {
		f := strings.Fields(line)
		pkg := f[0]
		for _, imp := range f[1:] {
			if !strings.HasPrefix(imp, q) {
				continue
			}
			if from, to := tier(pkg), tier(imp); from >= 0 && to > from {
				t.Errorf("%s imports %s, which is above it", pkg, imp)
			}
			if (strings.HasPrefix(pkg, q+"/pack/") || strings.HasPrefix(pkg, q+"/feature/") || strings.HasPrefix(pkg, q+"/service/")) && strings.HasPrefix(imp, q+"/internal/") {
				t.Errorf("%s imports %s, which one from outside could not", pkg, imp)
			}
			if strings.HasPrefix(pkg, q+"/libs/") && !strings.HasPrefix(imp, q+"/libs/") {
				t.Errorf("%s imports %s", pkg, imp)
			}
			if rule, ok := may[pkg]; ok && !rule(imp) {
				t.Errorf("%s imports %s", pkg, imp)
			}
		}
	}
}
