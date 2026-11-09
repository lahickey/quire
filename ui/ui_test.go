package ui

import "testing"

func TestNavActive(t *testing.T) {
	for _, c := range []struct {
		path, href string
		want       bool
	}{
		{"/", "/", true}, {"/a", "/", false},
		{"/project", "/project", true}, {"/project/x", "/project", true}, {"/projects", "/project", false},
	} {
		if navActive(c.path, c.href) != c.want {
			t.Errorf("navActive(%q, %q) = %t", c.path, c.href, !c.want)
		}
	}
}
