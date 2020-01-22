package environ

import (
	"fmt"
	"strings"

	"github.com/zbiljic/sicc/store"
)

// Environ is a slice of strings representing the environment,
// in the form "key=value".
type Environ []string

// Unset an environment variable by key.
func (e *Environ) Unset(key string) {
	for i := range *e {
		if strings.HasPrefix((*e)[i], key+"=") {
			(*e)[i] = (*e)[len(*e)-1]
			*e = (*e)[:len(*e)-1]

			break
		}
	}
}

// IsSet returns whether or not a key is currently set in the environ.
func (e *Environ) IsSet(key string) bool {
	for i := range *e {
		if strings.HasPrefix((*e)[i], key+"=") {
			return true
		}
	}

	return false
}

// Set adds an environment variable, replacing any existing ones of the same key.
func (e *Environ) Set(key, val string) {
	e.Unset(key)
	*e = append(*e, key+"="+val)
}

// Map squashes the list-like environ, taking the latter value when there are
// collisions, like a shell would. Invalid items (e.g., missing `=`) are dropped.
func (e *Environ) Map() map[string]string {
	ret := map[string]string{}

	//nolint:gomnd
	for _, kv := range []string(*e) {
		s := strings.SplitN(kv, "=", 2)

		if len(s) != 2 {
			// drop invalid kv pairs
			// this could happen in theory
			continue
		}

		ret[s[0]] = s[1]
	}

	return ret
}

// fromMap returns an Environ based on m. Order is arbitrary.
func fromMap(m map[string]string) Environ {
	e := make([]string, 0, len(m))

	for k, v := range m {
		e = append(e, k+"="+v)
	}

	return Environ(e)
}

// transforms a config key to an environment variable name, i.e. uppercase,
// substitute `-` -> `_`.
func configKeyToEnvVarName(k string) string {
	return normalizeEnvVarName(k)
}

func normalizeEnvVarName(k string) string {
	envVarName := strings.ToUpper(k)
	envVarName = strings.ReplaceAll(envVarName, "/", "_")
	envVarName = strings.ReplaceAll(envVarName, "-", "_")
	envVarName = strings.ReplaceAll(envVarName, ".", "_")
	envVarName = strings.TrimPrefix(envVarName, "_")

	return envVarName
}

// Load loads environment variables into 'e' from 's' given a prefix path.
// Collisions will be populated with any keys that get overwritten.
func (e *Environ) Load(s store.Store, prefixPath string, collisions *[]string) error {
	return e.load(s, prefixPath, collisions)
}

func (e *Environ) load(s store.Store, prefixPath string, collisions *[]string) error {
	rawValues, err := s.ListRaw(prefixPath)
	if err != nil {
		return fmt.Errorf("failed to list store contents (%s): %w", prefixPath, err)
	}

	for _, rawValue := range rawValues {
		key := strings.TrimPrefix(rawValue.Key, prefixPath)
		envVarKey := configKeyToEnvVarName(key)

		if e.IsSet(envVarKey) {
			*collisions = append(*collisions, envVarKey)
		}

		e.Set(envVarKey, rawValue.Value)
	}

	return nil
}

// LoadStrict loads all prefix paths from 's' in strict mode: env vars in 'e'
// with value equal to 'valueExpected' are the only ones substituted.
// If there are any env vars in 's' that are also in 'e', but don't have their
// value set to 'valueExpected' it returns an error.
func (e *Environ) LoadStrict(s store.Store, valueExpected string, pristine bool, prefixPaths ...string) error {
	return e.loadStrict(s, valueExpected, pristine, prefixPaths...)
}

func (e *Environ) loadStrict(s store.Store, valueExpected string, pristine bool, prefixPaths ...string) error {
	for _, prefixPath := range prefixPaths {
		rawValues, err := s.ListRaw(prefixPath)
		if err != nil {
			return fmt.Errorf("failed to list store contents (%s): %w", prefixPath, err)
		}

		err = e.loadStrictOne(rawValues, valueExpected, pristine)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Environ) loadStrictOne(rawValues []store.RawValue, valueExpected string, pristine bool) error {
	parentMap := e.Map()
	parentExpects := map[string]struct{}{}

	for k, v := range parentMap {
		if v == valueExpected {
			if k != normalizeEnvVarName(k) {
				return ExpectedKeyUnnormalizedError{Key: k, ValueExpected: valueExpected}
			}

			parentExpects[k] = struct{}{}
		}
	}

	envVarKeysAdded := map[string]struct{}{}

	for _, rawValue := range rawValues {
		envVarKey := configKeyToEnvVarName(rawValue.Key)

		parentVal, parentOk := parentMap[envVarKey]
		// skip injecting configurations that are not present in the parent
		if !parentOk {
			continue
		}

		delete(parentExpects, envVarKey)

		if parentVal != valueExpected {
			return StoreUnexpectedValueError{Key: envVarKey, ValueExpected: valueExpected, ValueActual: parentVal}
		}

		envVarKeysAdded[envVarKey] = struct{}{}

		e.Set(envVarKey, rawValue.Value)
	}

	for k := range parentExpects {
		return StoreMissingKeyError{Key: k, ValueExpected: valueExpected}
	}

	if pristine {
		// unset all envvars that were in the parent env but not in store
		for k := range parentMap {
			if _, ok := envVarKeysAdded[k]; !ok {
				e.Unset(k)
			}
		}
	}

	return nil
}

type StoreUnexpectedValueError struct {
	// store-style key
	Key           string
	ValueExpected string
	ValueActual   string
}

func (e StoreUnexpectedValueError) Error() string {
	return fmt.Sprintf("parent env has %s, but was expecting value `%s`, not `%s`", e.Key, e.ValueExpected, e.ValueActual)
}

type StoreMissingKeyError struct {
	// env-style key
	Key           string
	ValueExpected string
}

func (e StoreMissingKeyError) Error() string {
	return fmt.Sprintf("parent env was expecting %s=%s, but was not in store", e.Key, e.ValueExpected)
}

type ExpectedKeyUnnormalizedError struct {
	Key           string
	ValueExpected string
}

func (e ExpectedKeyUnnormalizedError) Error() string {
	//nolint:lll
	return fmt.Sprintf("parent env has key `%s` with expected value `%s`, but key is not normalized like `%s`, so would never get substituted",
		e.Key, e.ValueExpected, normalizeEnvVarName(e.Key))
}
