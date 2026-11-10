package bake

var converters = []converter{
	{"markdown", 3, "ce4c2a038f325008"},
	{"diagram", 1, "32ff824c256af489"},
	{"card", 1, "627683148db2ae9f"},
	{"picture", 1, "8a6626e549e73e11"},
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
