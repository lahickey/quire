package bake

var converters = []converter{
	{"markdown", 4, "3b2b8ed1dcbb8224"},
	{"diagram", 1, "ce269429ee55a051"},
	{"card", 1, "47807f6ccb03cebd"},
	{"picture", 1, "dd3fe6e9843d54fb"},
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
