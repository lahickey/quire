package lru

import "testing"

func TestEvictsByBytes(t *testing.T) {
	entry := func(n int) []byte { return make([]byte, n) }
	c := New(300, func(b []byte) int { return len(b) })
	c.Put("a", entry(100))
	c.Put("b", entry(100))
	c.Put("c", entry(100))
	if c.used != 300 {
		t.Fatalf("used %d, want 300", c.used)
	}
	c.Get("a")
	c.Put("d", entry(100))
	if c.used > 300 {
		t.Errorf("used %d, over budget", c.used)
	}
	for key, want := range map[string]bool{"a": true, "b": false, "c": true, "d": true} {
		if _, got := c.Get(key); got != want {
			t.Errorf("%q cached = %v, want %v", key, got, want)
		}
	}
	c.Put("huge", entry(400))
	if _, ok := c.Get("huge"); ok || c.used > 300 {
		t.Errorf("oversized entry admitted (used %d)", c.used)
	}
}
