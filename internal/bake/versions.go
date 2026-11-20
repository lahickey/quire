package bake

var converters = []converter{
	{"markdown", 7, "2bfc9ba553ec0d9d"},
	{"diagram", 1, "5d0af2ddf6605ec0"},
	{"card", 2, "2ae7257ddf9650cd"},
	{"picture", 1, "27bfcf08fea1cb02"},
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
