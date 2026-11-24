// Package store is somewhere to keep small things by key: a cart, a preference, a draft.
// What needs to ask questions of its data brings a database of its own.
package store

import (
	"bytes"
	"container/list"
	"context"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/lahickey/quire/model"
)

// Store keeps bytes by key. What the bytes are is its user's business. List is in key order.
type Store interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Put(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) iter.Seq2[string, []byte]
}

// memoryMax is how much a Memory keeps, in bytes of keys and values.
const memoryMax = 8 << 20

// Memory is a store that a restart empties. It keeps only so much: when it is full, what was written longest ago is dropped.
type Memory struct {
	mu    sync.RWMutex
	kept  map[string][]byte
	order *list.List
	at    map[string]*list.Element
	size  int
	max   int
}

type under struct {
	in     Store
	prefix string
}

func NewMemory() *Memory {
	return &Memory{kept: map[string][]byte{}, order: list.New(), at: map[string]*list.Element{}, max: memoryMax}
}

func (m *Memory) drop(key string) {
	if e, ok := m.at[key]; ok {
		m.size -= len(key) + len(m.kept[key])
		m.order.Remove(e)
		delete(m.at, key)
		delete(m.kept, key)
	}
}

func (m *Memory) Start(context.Context, model.Env) error { return nil }

func (m *Memory) Stop(context.Context) error { return nil }

func (m *Memory) Get(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.kept[key]
	return bytes.Clone(v), ok, nil
}

func (m *Memory) Put(_ context.Context, key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	need := len(key) + len(value)
	if need > m.max {
		return fmt.Errorf("store: %s is %d bytes, and the whole store keeps %d", key, need, m.max)
	}
	m.drop(key)
	for m.size+need > m.max {
		m.drop(m.order.Front().Value.(string))
	}
	m.kept[key], m.at[key] = bytes.Clone(value), m.order.PushBack(key)
	m.size += need
	return nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drop(key)
	return nil
}

func (m *Memory) List(_ context.Context, prefix string) iter.Seq2[string, []byte] {
	m.mu.RLock()
	found := map[string][]byte{}
	for k, v := range m.kept {
		if strings.HasPrefix(k, prefix) {
			found[k] = bytes.Clone(v)
		}
	}
	m.mu.RUnlock()
	return func(yield func(string, []byte) bool) {
		for _, k := range slices.Sorted(maps.Keys(found)) {
			if !yield(k, found[k]) {
				return
			}
		}
	}
}

// Under is a store's keys beneath a name, so two users of one store cannot read or write each other's. A feature puts its own name here.
func Under(s Store, name string) Store { return under{s, name + "/"} }

func (u under) Get(ctx context.Context, key string) ([]byte, bool, error) {
	return u.in.Get(ctx, u.prefix+key)
}

func (u under) Put(ctx context.Context, key string, value []byte) error {
	return u.in.Put(ctx, u.prefix+key, value)
}

func (u under) Delete(ctx context.Context, key string) error { return u.in.Delete(ctx, u.prefix+key) }

func (u under) List(ctx context.Context, prefix string) iter.Seq2[string, []byte] {
	return func(yield func(string, []byte) bool) {
		for k, v := range u.in.List(ctx, u.prefix+prefix) {
			if !yield(strings.TrimPrefix(k, u.prefix), v) {
				return
			}
		}
	}
}
