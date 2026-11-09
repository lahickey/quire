package picture

import (
	"runtime/debug"
	"strings"
	"sync"
)

var buildInfo = sync.OnceValues(debug.ReadBuildInfo)

func moduleVersion(path string) string {
	bi, ok := buildInfo()
	if !ok {
		return ""
	}
	if bi.Main.Path == path {
		return bi.Main.Version
	}
	for _, d := range bi.Deps {
		if d.Path != path {
			continue
		}
		if d.Replace != nil {
			return strings.TrimPrefix(d.Replace.Path+"@"+d.Replace.Version, "@")
		}
		return d.Version
	}
	return ""
}
