package store

import (
	"errors"
	"time"
)

var (
	// ErrConfigNotFound is returned if the specified config is not found in the
	// parameter store.
	ErrConfigNotFound = errors.New("config not found")
)

// ParameterName represents full name of the configuration parameter
type ParameterName struct {
	ParameterPath string
	Name          string
}

// Value contans actual value of parameter
type Value struct {
	Value *string
	Meta  Metadata
}

// RawValue represents value without any metadata
type RawValue struct {
	Value string
	Key   string
}

type Metadata struct {
	Key              string
	Description      string
	Secure           bool
	Version          int
	LastModifiedDate time.Time
	LastModifiedUser string
}

type Store interface {
	Put(name ParameterName, value Value) error
	Get(name ParameterName, version int) (Value, error)
	List(prefix string, includeValues bool) ([]Value, error)
	ListRaw(prefix string) ([]RawValue, error)
	Delete(name ParameterName) error
}
