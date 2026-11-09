package bake

type converter struct {
	dir         string
	version     int
	fingerprint string
}

var converters = []converter{
	{"markdown", 1, "f018b0ab0bfe949d"},
	{"diagram", 1, "f16b8725718c3899"},
	{"card", 1, "8688d94f191d84ac"},
	{"picture", 1, "e859a6924958a894"},
}

func version(dir string) int {
	for _, c := range converters {
		if c.dir == dir {
			return c.version
		}
	}
	panic("quire: no converter " + dir)
}
