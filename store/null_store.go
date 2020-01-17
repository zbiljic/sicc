package store

import "errors"

type NullStore struct{}

func NewNullStore() *NullStore {
	return &NullStore{}
}

func (s *NullStore) Put(name ParameterName, value Value) error {
	return errors.New("not implemented for Null store")
}

func (s *NullStore) Get(name ParameterName, version int) (Value, error) {
	return Value{}, errors.New("not implemented for Null store")
}

func (s *NullStore) List(prefix string, includeValues bool) ([]Value, error) {
	return []Value{}, errors.New("not implemented for Null store")
}

func (s *NullStore) ListRaw(prefix string) ([]RawValue, error) {
	return []RawValue{}, errors.New("not implemented for Null store")
}

func (s *NullStore) Delete(name ParameterName) error {
	return errors.New("not implemented for Null store")
}

// Check the interfaces are satisfied
var (
	_ Store = &NullStore{}
)
