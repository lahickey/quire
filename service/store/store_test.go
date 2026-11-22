package store

import (
	"maps"
	"slices"
	"testing"
)

func TestMemoryAndUnder(t *testing.T) {
	ctx, all := t.Context(), NewMemory()
	basket, search := Under(all, "basket"), Under(all, "search")
	for key, st := range map[string]Store{"b": basket, "a": basket, "x": search} {
		if err := st.Put(ctx, key, []byte(key)); err != nil {
			t.Fatal(err)
		}
	}
	if got := slices.Collect(maps.Keys(maps.Collect(basket.List(ctx, "")))); len(got) != 2 {
		t.Errorf("the basket sees %v, want only its own two", got)
	}
	var order []string
	for k := range all.List(ctx, "") {
		order = append(order, k)
	}
	if want := []string{"basket/a", "basket/b", "search/x"}; !slices.Equal(order, want) {
		t.Errorf("listed %v, want %v", order, want)
	}
	if _, ok, _ := search.Get(ctx, "a"); ok {
		t.Error("search read the basket's key")
	}
	v, _, _ := basket.Get(ctx, "a")
	v[0] = 'z'
	if again, _, _ := basket.Get(ctx, "a"); string(again) != "a" {
		t.Error("changing what Get returned changed what is kept")
	}
	if err := basket.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := all.Get(ctx, "basket/a"); ok {
		t.Error("a deleted key is still kept")
	}
}
