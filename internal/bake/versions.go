package bake

var converters = []converter{
	{"markdown", 3, "8c403ecbfef34760"},
	{"diagram", 1, "9664dafd90b39955"},
	{"card", 1, "62f16c70e3628e9f"},
	{"picture", 1, "d07796946d6b9e39"},
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
