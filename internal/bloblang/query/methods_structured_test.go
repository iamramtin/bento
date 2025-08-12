package query

import (
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/warpstreamlabs/bento/internal/message"
	"github.com/warpstreamlabs/bento/internal/value"
)

func TestMethodImmutability(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		target any
		args   []any
		exp    any
	}{
		{
			name:   "merge arrays",
			method: "merge",
			target: []any{"foo", "bar"},
			args: []any{
				[]any{"baz", "buz"},
			},
			exp: []any{"foo", "bar", "baz", "buz"},
		},
		{
			name:   "merge into an array",
			method: "merge",
			target: []any{"foo", "bar"},
			args: []any{
				map[string]any{"baz": "buz"},
			},
			exp: []any{"foo", "bar", map[string]any{"baz": "buz"}},
		},
		{
			name:   "merge objects",
			method: "merge",
			target: map[string]any{"foo": "bar"},
			args: []any{
				map[string]any{"baz": "buz"},
			},
			exp: map[string]any{
				"foo": "bar",
				"baz": "buz",
			},
		},
		{
			name:   "merge collision",
			method: "merge",
			target: map[string]any{"foo": "bar", "baz": "buz"},
			args: []any{
				map[string]any{"foo": "qux"},
			},
			exp: map[string]any{
				"foo": []any{"bar", "qux"},
				"baz": "buz",
			},
		},

		{
			name:   "assign arrays",
			method: "assign",
			target: []any{"foo", "bar"},
			args: []any{
				[]any{"baz", "buz"},
			},
			exp: []any{"foo", "bar", "baz", "buz"},
		},
		{
			name:   "assign into an array",
			method: "assign",
			target: []any{"foo", "bar"},
			args: []any{
				map[string]any{"baz": "buz"},
			},
			exp: []any{"foo", "bar", map[string]any{"baz": "buz"}},
		},
		{
			name:   "assign objects",
			method: "assign",
			target: map[string]any{"foo": "bar"},
			args: []any{
				map[string]any{"baz": "buz"},
			},
			exp: map[string]any{
				"foo": "bar",
				"baz": "buz",
			},
		},
		{
			name:   "assign collision",
			method: "assign",
			target: map[string]any{"foo": "bar", "baz": "buz"},
			args: []any{
				map[string]any{"foo": "qux"},
			},
			exp: map[string]any{
				"foo": "qux",
				"baz": "buz",
			},
		},

		{
			name:   "contains object positive",
			method: "contains",
			target: []any{
				map[string]any{"foo": "bar"},
			},
			args: []any{
				map[string]any{"foo": "bar"},
			},
			exp: true,
		},
		{
			name:   "contains object negative",
			method: "contains",
			target: []any{
				map[string]any{"foo": "bar"},
			},
			args: []any{
				map[string]any{"baz": "buz"},
			},
			exp: false,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			targetClone := value.IClone(test.target)
			argsClone := value.IClone(test.args).([]any)

			fn, err := InitMethodHelper(test.method, NewLiteralFunction("", targetClone), argsClone...)
			require.NoError(t, err)

			res, err := fn.Exec(FunctionContext{
				Maps:     map[string]Function{},
				Index:    0,
				MsgBatch: nil,
			})
			require.NoError(t, err)

			assert.Equal(t, test.exp, res)
			assert.Equal(t, test.target, targetClone)
			assert.Equal(t, test.args, argsClone)
		})
	}
}

func TestMethodCut(t *testing.T) {
	type easyMethod struct {
		name string
		args []any
	}

	type easyMsg struct {
		content string
		meta    map[string]any
	}

	literalFn := func(val any) Function {
		fn := NewLiteralFunction("", val)
		return fn
	}

	jsonFn := func(json string) Function {
		t.Helper()
		gObj, err := gabs.ParseJSON([]byte(json))
		require.NoError(t, err)
		fn := NewLiteralFunction("", gObj.Data())
		return fn
	}

	methods := func(fn Function, methods ...easyMethod) Function {
		t.Helper()
		for _, m := range methods {
			var err error
			fn, err = InitMethodHelper(m.name, fn, m.args...)
			require.NoError(t, err)
		}
		return fn
	}

	method := func(name string, args ...any) easyMethod {
		return easyMethod{name: name, args: args}
	}

	tests := map[string]struct {
		input    Function
		value    *any
		output   any
		err      string
		messages []easyMsg
		index    int
	}{
		// String tests with static delimiter
		"cut string static delimiter found": {
			input: methods(
				literalFn("Bento 🍱 is a fast stream processor"),
				method("cut", "🍱"),
			),
			output: []any{"Bento 🍱", " is a fast stream processor"},
		},
		"cut string static delimiter at start": {
			input: methods(
				literalFn("🍱Bento is fast"),
				method("cut", "🍱"),
			),
			output: []any{"🍱", "Bento is fast"},
		},
		"cut string static delimiter at end": {
			input: methods(
				literalFn("Bento is fast🍱"),
				method("cut", "🍱"),
			),
			output: []any{"Bento is fast🍱", ""},
		},
		"cut string static delimiter not found": {
			input: methods(
				literalFn("Bento is fast"),
				method("cut", "🍱"),
			),
			output: []any{"Bento is fast", ""},
		},
		"cut string static delimiter empty": {
			input: methods(
				literalFn("Bento is fast"),
				method("cut", ""),
			),
			output: []any{"Bento is fast", ""},
		},
		"cut string empty input": {
			input: methods(
				literalFn(""),
				method("cut", "x"),
			),
			output: []any{"", ""},
		},
		"cut string multi-char delimiter": {
			input: methods(
				literalFn("foo::bar::baz"),
				method("cut", "::"),
			),
			output: []any{"foo::", "bar::baz"},
		},

		// Array tests with static delimiter
		"cut array static delimiter found": {
			input: methods(
				jsonFn(`["foo", "bar", "baz", "qux"]`),
				method("cut", "bar"),
			),
			output: []any{[]any{"foo", "bar"}, []any{"baz", "qux"}},
		},
		"cut array static delimiter at start": {
			input: methods(
				jsonFn(`["bar", "foo", "baz"]`),
				method("cut", "bar"),
			),
			output: []any{[]any{"bar"}, []any{"foo", "baz"}},
		},
		"cut array static delimiter at end": {
			input: methods(
				jsonFn(`["foo", "baz", "bar"]`),
				method("cut", "bar"),
			),
			output: []any{[]any{"foo", "baz", "bar"}, []any{}},
		},
		"cut array static delimiter not found": {
			input: methods(
				jsonFn(`["foo", "baz", "qux"]`),
				method("cut", "bar"),
			),
			output: []any{[]any{"foo", "baz", "qux"}, []any{}},
		},
		"cut array empty input": {
			input: methods(
				jsonFn(`[]`),
				method("cut", "bar"),
			),
			output: []any{[]any{}, []any{}},
		},
		"cut array mixed types": {
			input: methods(
				jsonFn(`["foo", 42, "bar", true]`),
				method("cut", 42),
			),
			output: []any{[]any{"foo", 42.0}, []any{"bar", true}},
		},

		// Error cases
		"cut invalid input type with static delimiter": {
			input: methods(
				literalFn(42),
				method("cut", "test"),
			),
			err: `expected array value, got number`,
		},
		"cut object input type with static delimiter": {
			input: methods(
				jsonFn(`{"foo": "bar"}`),
				method("cut", "test"),
			),
			err: `expected array value, got object`,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var err error
			var res any

			if len(test.messages) == 0 {
				test.messages = []easyMsg{
					{content: `{}`},
				}
			}

			msgBatch := message.QuickBatch(nil)
			part := message.NewPart(nil)
			if test.messages[test.index].content != "" {
				part.SetBytes([]byte(test.messages[test.index].content))
			}
			if test.messages[test.index].meta != nil {
				for k, v := range test.messages[test.index].meta {
					part.MetaSetMut(k, v)
				}
			}
			if test.value != nil {
				part = message.NewPart([]byte(`{}`))
			}
			msgBatch = append(msgBatch, part)

			for range 10 {
				res, err = test.input.Exec(FunctionContext{
					Maps:     map[string]Function{},
					Index:    test.index,
					MsgBatch: msgBatch,
				})
				if test.err != "" {
					require.Error(t, err)
					assert.Contains(t, err.Error(), test.err)
				} else {
					require.NoError(t, err)
				}
			}

			if test.err == "" {
				assert.Equal(t, test.output, res)
			}

			// Ensure nothing changed
			assert.Equal(t, test.messages[test.index].content, string(part.AsBytes()))
			for k, v := range test.messages[test.index].meta {
				actual, _ := part.MetaGetMut(k)
				assert.Equal(t, v, actual)
			}
		})
	}
}
