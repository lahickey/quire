package lru

import (
	"container/list"
	"sync"
)

type Cache[V any] struct {
	mu    sync.Mutex
	max   int
	used  int
	cost  func(V) int
	order *list.List
	items map[string]*list.Element
}

type item[V any] struct {
	key   string
	value V
}

func New[V any](max int, cost func(V) int) *Cache[V] {
	return &Cache[V]{max: max, cost: cost, order: list.New(), items: map[string]*list.Element{}}
}

func (c *Cache[V]) drop(el *list.Element) {
	it := el.Value.(*item[V])
	c.order.Remove(el)
	delete(c.items, it.key)
	c.used -= c.cost(it.value)
}

func (c *Cache[V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

func (c *Cache[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	c.order.MoveToFront(el)
	return el.Value.(*item[V]).value, true
}

func (c *Cache[V]) Put(key string, v V) {
	cost := c.cost(v)
	c.mu.Lock()
	defer c.mu.Unlock()
	if cost > c.max {
		return
	}
	if el, ok := c.items[key]; ok {
		c.used -= c.cost(el.Value.(*item[V]).value)
		el.Value = &item[V]{key, v}
		c.order.MoveToFront(el)
		c.used += cost
	} else {
		c.items[key] = c.order.PushFront(&item[V]{key, v})
		c.used += cost
	}
	for c.used > c.max {
		cold := c.order.Back()
		if cold == nil {
			return
		}
		c.drop(cold)
	}
}
