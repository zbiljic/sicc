package environ

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zbiljic/sicc/store"
)

func TestUnset(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		var env Environ

		env.Set("k", "v")

		env.Unset("k")
	})

	t.Run("nonExistent", func(t *testing.T) {
		var env Environ

		env.Unset("k")
	})
}

func TestIsSet(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		var env Environ

		env.Set("k", "v")

		exists := env.IsSet("k")

		assert.True(t, exists)
	})

	t.Run("nonExistent", func(t *testing.T) {
		var env Environ

		exists := env.IsSet("k")

		assert.False(t, exists)
	})
}

func TestSet(t *testing.T) {
	var env Environ

	env.Set("k", "v")
}

func TestMap(t *testing.T) {
	cases := []struct {
		name string
		in   Environ
		out  map[string]string
	}{
		{
			"basic",
			Environ([]string{
				"k=v",
			}),
			map[string]string{
				"k": "v",
			},
		},
		{
			"nested",
			Environ([]string{
				"/t/k=v",
			}),
			map[string]string{
				"/t/k": "v",
			},
		},
		{
			"dropping malformed",
			Environ([]string{
				"k=v",
				// should work
				"k2=",
			}),
			map[string]string{
				"k":  "v",
				"k2": "",
			},
		},
		{
			"squash",
			Environ([]string{
				"k=v1",
				"k=v2",
			}),
			map[string]string{
				"k": "v2",
			},
		},
	}

	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			m := testCase.in.Map()

			assert.EqualValues(t, m, testCase.out)
		})
	}
}

func TestFromMap(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]string
		out  Environ
	}{
		{
			"basic",
			map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			Environ([]string{
				"k1=v1",
				"k2=v2",
			}),
		},
		{
			"nested",
			map[string]string{
				"t/k1": "v1",
				"t/k2": "v2",
			},
			Environ([]string{
				"t/k1=v1",
				"t/k2=v2",
			}),
		},
	}

	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			e := fromMap(testCase.in)
			// maps order is non-deterministic
			sort.Strings(e)

			assert.EqualValues(t, e, testCase.out)
		})
	}
}

func TestConfigKeyToEnvVarName(t *testing.T) {
	cases := []struct {
		key               string
		expectedEnvVarKey string
	}{
		{
			key:               "/k",
			expectedEnvVarKey: "K",
		},
		{
			key:               "/t/k",
			expectedEnvVarKey: "T_K",
		},
		{
			key:               "/t/p/k",
			expectedEnvVarKey: "T_P_K",
		},
	}

	for _, testCase := range cases {
		envVarKey := configKeyToEnvVarName(testCase.key)

		assert.EqualValues(t, envVarKey, testCase.expectedEnvVarKey)
	}
}

//nolint:funlen
func TestLoad(t *testing.T) {
	t.Run("nullStore", func(t *testing.T) {
		s := store.NewNullStore()

		var env Environ

		err := env.Load(s, "", nil)

		assert.Error(t, err)
	})

	cases := []struct {
		name               string
		e                  Environ
		prefixPath         string
		configs            map[string]string
		expectedEnvMap     map[string]string
		expectedErr        error
		expectedCollisions []string
	}{
		{
			name: "basic",
			e: fromMap(map[string]string{
				"HOME": "/tmp",
			}),
			prefixPath: "/test",
			configs: map[string]string{
				"/test/db/username": "admin",
				"/test/db/password": "pass",
			},
			expectedEnvMap: map[string]string{
				"HOME":        "/tmp",
				"DB_USERNAME": "admin",
				"DB_PASSWORD": "pass",
			},
			expectedCollisions: make([]string, 0),
		},
		{
			name:       "collision",
			e:          fromMap(map[string]string{}),
			prefixPath: "/test",
			configs: map[string]string{
				"/test/svc/key": "value1",
				"/test/svc_key": "value2",
			},
			expectedEnvMap: map[string]string{
				"SVC_KEY": "value2",
			},
			expectedCollisions: []string{"SVC_KEY"},
		},
	}

	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			s := store.NewMemoryStoreFromMap(testCase.configs)

			collisions := make([]string, 0)

			err := testCase.e.Load(s, testCase.prefixPath, &collisions)
			if err != nil {
				assert.EqualValues(t, testCase.expectedErr, err)
			} else {
				assert.EqualValues(t, testCase.expectedEnvMap, testCase.e.Map())
			}

			if len(collisions) > 0 {
				assert.Len(t, collisions, len(testCase.expectedCollisions))
				assert.EqualValues(t, testCase.expectedCollisions, collisions)
			}
		})
	}
}

//nolint:funlen
func TestLoadStrict(t *testing.T) {
	t.Run("nullStore", func(t *testing.T) {
		s := store.NewNullStore()

		var env Environ

		err := env.LoadStrict(s, "", false, "")

		assert.Error(t, err)
	})

	cases := []struct {
		name           string
		e              Environ
		strictVal      string // default: "changeme"
		pristine       bool
		prefixPaths    []string
		configs        map[string]string
		expectedEnvMap map[string]string
		expectedErr    error
	}{
		{
			name: "basic",
			e: fromMap(map[string]string{
				"HOME":        "/tmp",
				"DB_USERNAME": "changeme",
				"DB_PASSWORD": "changeme",
			}),
			prefixPaths: []string{"/test"},
			configs: map[string]string{
				"/test/db/username": "admin",
				"/test/db/password": "pass",
			},
			expectedEnvMap: map[string]string{
				"HOME":        "/tmp",
				"DB_USERNAME": "admin",
				"DB_PASSWORD": "pass",
			},
		},
		{
			name: "with unfilled",
			e: fromMap(map[string]string{
				"HOME":        "/tmp",
				"DB_USERNAME": "changeme",
				"DB_PASSWORD": "changeme",
				"EXTRA":       "changeme",
			}),
			prefixPaths: []string{"/test"},
			configs: map[string]string{
				"/test/db/username": "admin",
				"/test/db/password": "pass",
			},
			expectedErr: StoreMissingKeyError{Key: "EXTRA", ValueExpected: "changeme"},
		},
		{
			name: "pristine",
			e: fromMap(map[string]string{
				"HOME":        "/tmp",
				"DB_USERNAME": "changeme",
				"DB_PASSWORD": "changeme",
			}),
			pristine:    true,
			prefixPaths: []string{"/test"},
			configs: map[string]string{
				"/test/db/username": "admin",
				"/test/db/password": "pass",
			},
			expectedEnvMap: map[string]string{
				"DB_USERNAME": "admin",
				"DB_PASSWORD": "pass",
			},
		},
		{
			name: "pristine with unnormalized key name",
			e: fromMap(map[string]string{
				"HOME":        "/tmp",
				"DB_username": "changeme",
				"DB_PASSWORD": "changeme",
			}),
			pristine:    true,
			prefixPaths: []string{"/test"},
			configs: map[string]string{
				"/test/db/username": "admin",
				"/test/db/password": "pass",
			},
			expectedErr: ExpectedKeyUnnormalizedError{Key: "DB_username", ValueExpected: "changeme"},
		},
	}

	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			s := store.NewMemoryStoreFromMap(testCase.configs)

			strictVal := testCase.strictVal
			if strictVal == "" {
				strictVal = "changeme"
			}

			err := testCase.e.LoadStrict(s, strictVal, testCase.pristine, testCase.prefixPaths...)
			if err != nil {
				assert.EqualValues(t, testCase.expectedErr, err)
			} else {
				assert.EqualValues(t, testCase.expectedEnvMap, testCase.e.Map())
			}
		})
	}
}
