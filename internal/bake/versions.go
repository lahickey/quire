package bake

var converters = []converter{
	{"markdown", 3, "e56b37141e283c99"},
	{"diagram", 1, "9664dafd90b39955"},
	{"card", 1, "c274426c61cf9040"},
	{"picture", 1, "af903e52aafd453d"},
}

type converter struct {
	dir         string
	version     int
	fingerprint string
}

func version(dir string) int {
	for _, c := range converters {
		if c.dir == dir {
			return c.version
		}
	}
	panic("quire: no converter " + dir)
}
