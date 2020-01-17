package store

import (
	"path"
	"sort"
	"strings"
	"sync"
)

type MemoryStore struct {
	m map[string]Value

	mu sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		m: make(map[string]Value),
	}
}

func NewMemoryStoreFromMap(m map[string]string) *MemoryStore {
	configs := map[string]Value{}

	for k, v := range m {
		k := path.Join("/", k)
		v := v
		configs[k] = Value{
			Value: &v,
		}
	}

	return &MemoryStore{m: configs}
}

func (s *MemoryStore) Put(name ParameterName, value Value) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := path.Join(name.ParameterPath, name.Name)

	s.m[key] = value

	return nil
}

func (s *MemoryStore) Get(name ParameterName, version int) (Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := path.Join(name.ParameterPath, name.Name)

	if val, ok := s.m[key]; ok {
		return val, nil
	}

	return Value{}, ErrConfigNotFound
}

func (s *MemoryStore) List(prefix string, includeValues bool) ([]Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	configs := map[string]Value{}

	prefixPath := path.Join("/", prefix)

	// sorted map range
	keys := make([]string, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		if strings.HasPrefix(k, prefixPath) {
			key := strings.TrimPrefix(k, prefixPath)
			configs[key] = Value{
				Value: nil,
			}

			if includeValues {
				v := s.m[k]

				value := configs[key]
				value.Value = v.Value
				configs[key] = value
			}
		}
	}

	values := []Value{}
	for _, v := range configs {
		values = append(values, v)
	}

	return values, nil
}

func (s *MemoryStore) ListRaw(prefix string) ([]RawValue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	values := map[string]RawValue{}

	prefixPath := path.Join("/", prefix)

	// sorted map range
	keys := make([]string, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		if strings.HasPrefix(k, prefixPath) {
			v := s.m[k]
			key := strings.TrimPrefix(k, prefixPath)
			values[key] = RawValue{
				Value: *v.Value,
				Key:   key,
			}
		}
	}

	rawValues := make([]RawValue, len(values))
	i := 0

	for _, rawValue := range values {
		rawValues[i] = rawValue
		i++
	}

	return rawValues, nil
}

func (s *MemoryStore) Delete(name ParameterName) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := path.Join(name.ParameterPath, name.Name)

	delete(s.m, key)

	return nil
}

// Check the interfaces are satisfied
var (
	_ Store = &MemoryStore{}
)
