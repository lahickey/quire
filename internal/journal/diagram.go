package journal

import "strings"

var SVG = Format{Ext: "svg", Type: "image/svg+xml"}

func DiagramName(src, fontID string) string {
	return "mermaid/" + Hash([]byte(fontID+"\x00"+strings.TrimSpace(src)))
}
