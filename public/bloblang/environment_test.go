package bloblang

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironment(t *testing.T) {
	env1, env2 := NewEnvironment(), NewEnvironment()

	require.NoError(t, env1.RegisterMethod("foo", func(_ ...interface{}) (Method, error) {
		return StringMethod(func(s string) (interface{}, error) {
			return "foo:" + s, nil
		}), nil
	}))

	require.NoError(t, env2.RegisterFunction("bar", func(_ ...interface{}) (Function, error) {
		return func() (interface{}, error) {
			return "bar", nil
		}, nil
	}))

	_, err := env1.Parse(`root = bar()`)
	assert.EqualError(t, err, "unrecognised function 'bar': bar()")

	exe, err := env1.Parse(`root = "bar".foo()`)
	require.NoError(t, err)

	v, err := exe.Query(nil)
	require.NoError(t, err)
	assert.Equal(t, "foo:bar", v)

	_, err = env2.Parse(`root = "bar".foo()`)
	assert.EqualError(t, err, "unrecognised method 'foo': foo()")

	exe, err = env2.Parse(`root = bar()`)
	require.NoError(t, err)

	v, err = exe.Query(nil)
	require.NoError(t, err)
	assert.Equal(t, "bar", v)
}

func TestEnvironmentV2(t *testing.T) {
	env1, env2 := NewEnvironment(), NewEnvironment()

	require.NoError(t, env1.RegisterMethodV2("foo", NewPluginSpec(), func(_ *ParsedParams) (Method, error) {
		return StringMethod(func(s string) (interface{}, error) {
			return "foo:" + s, nil
		}), nil
	}))

	require.NoError(t, env2.RegisterFunctionV2("bar", NewPluginSpec(), func(_ *ParsedParams) (Function, error) {
		return func() (interface{}, error) {
			return "bar", nil
		}, nil
	}))

	_, err := env1.Parse(`root = bar()`)
	assert.EqualError(t, err, "unrecognised function 'bar': bar()")

	exe, err := env1.Parse(`root = "bar".foo()`)
	require.NoError(t, err)

	v, err := exe.Query(nil)
	require.NoError(t, err)
	assert.Equal(t, "foo:bar", v)

	_, err = env2.Parse(`root = "bar".foo()`)
	assert.EqualError(t, err, "unrecognised method 'foo': foo()")

	exe, err = env2.Parse(`root = bar()`)
	require.NoError(t, err)

	v, err = exe.Query(nil)
	require.NoError(t, err)
	assert.Equal(t, "bar", v)
}

func TestEmptyEnvironment(t *testing.T) {
	env := NewEmptyEnvironment()

	env.RegisterMethod("foo", func(_ ...interface{}) (Method, error) {
		return StringMethod(func(s string) (interface{}, error) {
			return "foo:" + s, nil
		}), nil
	})

	_, err := env.Parse(`root = now()`)
	assert.EqualError(t, err, "unrecognised function 'now': now()")

	exe, err := env.Parse(`root = "hello world".foo()`)
	require.NoError(t, err)

	v, err := exe.Query(nil)
	require.NoError(t, err)
	assert.Equal(t, "foo:hello world", v)
}
