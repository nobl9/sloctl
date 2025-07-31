package jq

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpressionRunner_EvaluateAndPrint(t *testing.T) {
	tests := map[string]struct {
		input    any
		expr     string
		expected []any
		err      string
	}{
		"simple identity": {
			input:    map[string]any{"name": "test", "value": 42},
			expr:     ".",
			expected: []any{map[string]any{"name": "test", "value": float64(42)}},
		},
		"select field": {
			input:    map[string]any{"name": "test", "value": 42},
			expr:     ".name",
			expected: []any{"test"},
		},
		"select number field": {
			input:    map[string]any{"name": "test", "value": 42},
			expr:     ".value",
			expected: []any{float64(42)},
		},
		"array input": {
			input:    []string{"a", "b", "c"},
			expr:     ".[]",
			expected: []any{"a", "b", "c"},
		},
		"array length": {
			input:    []string{"a", "b", "c"},
			expr:     "length",
			expected: []any{3},
		},
		"filter array": {
			input: []map[string]any{
				{"name": "foo", "active": true},
				{"name": "bar", "active": false},
				{"name": "baz", "active": true},
			},
			expr: ".[] | select(.active)",
			expected: []any{
				map[string]any{"name": "foo", "active": true},
				map[string]any{"name": "baz", "active": true},
			},
		},
		"map values": {
			input:    []int{1, 2, 3},
			expr:     "map(. * 2)",
			expected: []any{[]any{float64(2), float64(4), float64(6)}},
		},
		"invalid expression": {
			input: map[string]any{"name": "test"},
			expr:  ".invalid syntax",
			err:   "failed to parse jq expression",
		},
		"empty result": {
			input:    map[string]any{"name": "test"},
			expr:     "empty",
			expected: nil,
		},
		"null result": {
			input:    map[string]any{"name": "test"},
			expr:     "null",
			expected: []any{nil},
		},
		"boolean results": {
			input:    map[string]any{"active": true, "disabled": false},
			expr:     ".active, .disabled",
			expected: []any{true, false},
		},
		"nested object access": {
			input: map[string]any{
				"user": map[string]any{
					"profile": map[string]any{
						"name": "John",
						"age":  30,
					},
				},
			},
			expr:     ".user.profile.name",
			expected: []any{"John"},
		},
		"error handling - undefined field": {
			input:    map[string]any{"name": "test"},
			expr:     ".nonexistent.field",
			expected: []any{nil},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runner := NewExpressionRunner(Config{
				Expression: tc.expr,
			})

			iter, err := runner.EvaluateAndPrint(tc.input)

			if tc.err != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.err)
			} else {
				require.NoError(t, err)

				values, iterErr := collectResults(iter)
				require.NoError(t, iterErr)

				if tc.expected == nil {
					assert.Nil(t, values)
				} else {
					assert.Equal(t, tc.expected, values)
				}
			}
		})
	}
}

func TestExpressionRunner_EvaluateAndPrint_ParseErrorFormatting(t *testing.T) {
	tests := map[string]struct {
		expr        string
		expectedErr string
	}{
		"invalid syntax": {
			expr:        ".invalid syntax",
			expectedErr: "failed to parse jq expression (line 1, column 10)",
		},
		"multiline error": {
			expr:        ".\nselect(",
			expectedErr: "failed to parse jq expression (line 2, column 1)",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runner := NewExpressionRunner(Config{
				Expression: tc.expr,
			})

			_, err := runner.EvaluateAndPrint(map[string]any{"test": "value"})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

func TestExpressionRunner_ShouldRun(t *testing.T) {
	tests := map[string]struct {
		expr     string
		expected bool
	}{
		"empty expression": {
			expr:     "",
			expected: false,
		},
		"non-empty expression": {
			expr:     ".",
			expected: true,
		},
		"whitespace expression": {
			expr:     "   ",
			expected: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runner := NewExpressionRunner(Config{
				Expression: tc.expr,
			})

			result := runner.ShouldRun()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExpressionRunner_EvaluateAndPrint_ComplexDataTypes(t *testing.T) {
	// Test with struct that gets converted through JSON marshaling/unmarshaling
	type TestStruct struct {
		Name   string            `json:"name"`
		Values []int             `json:"values"`
		Meta   map[string]string `json:"meta"`
	}

	input := TestStruct{
		Name:   "test",
		Values: []int{1, 2, 3},
		Meta:   map[string]string{"key": "value"},
	}

	runner := NewExpressionRunner(Config{
		Expression: ".name",
	})

	iter, err := runner.EvaluateAndPrint(input)
	require.NoError(t, err)

	values, iterErr := collectResults(iter)
	require.NoError(t, iterErr)
	assert.Equal(t, []any{"test"}, values)
}

// collectResults collects values from an iter.Seq2[any, error] and returns them as a slice
func collectResults(iter iter.Seq2[any, error]) ([]any, error) {
	var values []any // nolint: prealloc
	for v, err := range iter {
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}
