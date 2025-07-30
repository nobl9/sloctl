package jq

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpressionRunner_EvaluateAndPrint(t *testing.T) {
	ctx := context.Background()

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
			printer := &mockPrinter{}
			runner := NewExpressionRunner(Config{
				Printer:    printer,
				Expression: tc.expr,
			})

			err := runner.EvaluateAndPrint(ctx, tc.input)

			if tc.err != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.err)
			} else {
				require.NoError(t, err)
				if tc.expected == nil {
					assert.Nil(t, printer.printed)
				} else {
					assert.Equal(t, tc.expected, printer.printed)
				}
			}
		})
	}
}

func TestExpressionRunner_EvaluateAndPrint_PrinterError(t *testing.T) {
	ctx := context.Background()

	printer := &mockPrinter{err: fmt.Errorf("printer error")}
	runner := NewExpressionRunner(Config{
		Printer:    printer,
		Expression: ".",
	})

	err := runner.EvaluateAndPrint(ctx, map[string]any{"test": "value"})

	require.Error(t, err)
	assert.Equal(t, "printer error", err.Error())
}

func TestExpressionRunner_EvaluateAndPrint_ParseErrorFormatting(t *testing.T) {
	ctx := context.Background()

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
			printer := &mockPrinter{}
			runner := NewExpressionRunner(Config{
				Printer:    printer,
				Expression: tc.expr,
			})

			err := runner.EvaluateAndPrint(ctx, map[string]any{"test": "value"})

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
	ctx := context.Background()

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

	printer := &mockPrinter{}
	runner := NewExpressionRunner(Config{
		Printer:    printer,
		Expression: ".name",
	})

	err := runner.EvaluateAndPrint(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, []any{"test"}, printer.printed)
}

type mockPrinter struct {
	printed []any
	err     error
}

func (m *mockPrinter) Print(v any) error {
	if m.err != nil {
		return m.err
	}
	m.printed = append(m.printed, v)
	return nil
}
