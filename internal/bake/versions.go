package bake

var converters = []converter{
	{"markdown", 5, "55e45317d36063d1"},
	{"diagram", 1, "ce269429ee55a051"},
	{"card", 2, "ea5f672beafcf564"},
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
