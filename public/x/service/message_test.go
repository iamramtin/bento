package service

import (
	"errors"
	"testing"

	"github.com/Jeffail/benthos/v3/lib/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageCopyAirGap(t *testing.T) {
	p := message.NewPart([]byte("hello world"))
	p.Metadata().Set("foo", "bar")
	g1 := newMessageFromPart(p)
	g2 := g1.Copy()

	b := p.Get()
	v := p.Metadata().Get("foo")
	assert.Equal(t, "hello world", string(b))
	assert.Equal(t, "bar", v)

	b, err := g1.AsBytes()
	v, _ = g1.MetaGet("foo")
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(b))
	assert.Equal(t, "bar", v)

	b, err = g2.AsBytes()
	v, _ = g2.MetaGet("foo")
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(b))
	assert.Equal(t, "bar", v)

	g2.SetBytes([]byte("and now this"))
	g2.MetaSet("foo", "baz")

	b = p.Get()
	v = p.Metadata().Get("foo")
	assert.Equal(t, "hello world", string(b))
	assert.Equal(t, "bar", v)

	b, err = g1.AsBytes()
	v, _ = g1.MetaGet("foo")
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(b))
	assert.Equal(t, "bar", v)

	b, err = g2.AsBytes()
	v, _ = g2.MetaGet("foo")
	require.NoError(t, err)
	assert.Equal(t, "and now this", string(b))
	assert.Equal(t, "baz", v)

	g1.SetBytes([]byte("but not this"))
	g1.MetaSet("foo", "buz")

	b = p.Get()
	v = p.Metadata().Get("foo")
	assert.Equal(t, "hello world", string(b))
	assert.Equal(t, "bar", v)

	b, err = g1.AsBytes()
	v, _ = g1.MetaGet("foo")
	require.NoError(t, err)
	assert.Equal(t, "but not this", string(b))
	assert.Equal(t, "buz", v)

	b, err = g2.AsBytes()
	v, _ = g2.MetaGet("foo")
	require.NoError(t, err)
	assert.Equal(t, "and now this", string(b))
	assert.Equal(t, "baz", v)
}

func TestMessageQuery(t *testing.T) {
	p := message.NewPart([]byte(`{"foo":"bar"}`))
	p.Metadata().Set("foo", "bar")
	p.Metadata().Set("bar", "baz")
	g1 := newMessageFromPart(p)

	b, err := g1.AsBytes()
	assert.NoError(t, err)
	assert.Equal(t, `{"foo":"bar"}`, string(b))

	s, err := g1.AsStructured()
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, s)

	m, ok := g1.MetaGet("foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", m)

	seen := map[string]string{}
	err = g1.MetaWalk(func(k, v string) error {
		seen[k] = v
		return errors.New("stop")
	})
	assert.EqualError(t, err, "stop")
	assert.Len(t, seen, 1)

	seen = map[string]string{}
	err = g1.MetaWalk(func(k, v string) error {
		seen[k] = v
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"foo": "bar",
		"bar": "baz",
	}, seen)
}

func TestMessageMutate(t *testing.T) {
	p := message.NewPart([]byte(`not a json doc`))
	p.Metadata().Set("foo", "bar")
	p.Metadata().Set("bar", "baz")
	g1 := newMessageFromPart(p)

	_, err := g1.AsStructured()
	assert.Error(t, err)

	g1.SetStructured(map[string]interface{}{
		"foo": "bar",
	})
	assert.Equal(t, "not a json doc", string(p.Get()))

	s, err := g1.AsStructured()
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"foo": "bar",
	}, s)

	g1.SetBytes([]byte("foo bar baz"))
	assert.Equal(t, "not a json doc", string(p.Get()))

	_, err = g1.AsStructured()
	assert.Error(t, err)

	b, err := g1.AsBytes()
	assert.NoError(t, err)
	assert.Equal(t, "foo bar baz", string(b))

	g1.MetaDelete("foo")

	seen := map[string]string{}
	err = g1.MetaWalk(func(k, v string) error {
		seen[k] = v
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"bar": "baz"}, seen)

	g1.MetaSet("foo", "new bar")

	seen = map[string]string{}
	err = g1.MetaWalk(func(k, v string) error {
		seen[k] = v
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"foo": "new bar", "bar": "baz"}, seen)
}

func TestNewMessageMutate(t *testing.T) {
	g0 := NewMessage([]byte(`not a json doc`))
	g0.MetaSet("foo", "bar")
	g0.MetaSet("bar", "baz")

	g1 := g0.Copy()

	_, err := g1.AsStructured()
	assert.Error(t, err)

	g1.SetStructured(map[string]interface{}{
		"foo": "bar",
	})
	g0Bytes, err := g0.AsBytes()
	require.NoError(t, err)
	assert.Equal(t, "not a json doc", string(g0Bytes))

	s, err := g1.AsStructured()
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"foo": "bar",
	}, s)

	g1.SetBytes([]byte("foo bar baz"))
	g0Bytes, err = g0.AsBytes()
	require.NoError(t, err)
	assert.Equal(t, "not a json doc", string(g0Bytes))

	_, err = g1.AsStructured()
	assert.Error(t, err)

	b, err := g1.AsBytes()
	assert.NoError(t, err)
	assert.Equal(t, "foo bar baz", string(b))

	g1.MetaDelete("foo")

	seen := map[string]string{}
	err = g1.MetaWalk(func(k, v string) error {
		seen[k] = v
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"bar": "baz"}, seen)

	g1.MetaSet("foo", "new bar")

	seen = map[string]string{}
	err = g1.MetaWalk(func(k, v string) error {
		seen[k] = v
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"foo": "new bar", "bar": "baz"}, seen)
}
