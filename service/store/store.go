// Package store is somewhere to keep small things by key: a cart, a preference, a draft.
// What needs to ask questions of its data brings a database of its own.
package store

import (
	"bytes"
	"context"
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

// Memory is a store that a restart empties.
type Memory struct {
	mu   sync.RWMutex
	kept map[string][]byte
}

type under struct {
	in     Store
	prefix string
}

func NewMemory() *Memory { return &Memory{kept: map[string][]byte{}} }

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
	m.kept[key] = bytes.Clone(value)
	return nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.kept, key)
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
