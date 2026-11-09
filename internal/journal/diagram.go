package journal

import "strings"

func DiagramName(src, fontID string) string {
	return "mermaid/" + Hash([]byte(fontID+"\x00"+strings.TrimSpace(src)))
}

var SVG = Format{Ext: "svg", Type: "image/svg+xml"}
