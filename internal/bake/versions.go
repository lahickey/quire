package bake

var converters = []converter{
	{"markdown", 7, "a4c339ad297df0ac"},
	{"diagram", 1, "5d0af2ddf6605ec0"},
	{"card", 2, "e80be1fe6f878132"},
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
