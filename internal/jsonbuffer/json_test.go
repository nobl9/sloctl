package jsonbuffer

import "testing"

func TestIsJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "valid json object",
			input:    []byte(`{"key": "value"}`),
			expected: true,
		},
		{
			name:     "valid json object with leading whitespace",
			input:    []byte(`  {"key": "value"}`),
			expected: true,
		},
		{
			name:     "valid json object with tabs",
			input:    []byte(`	{"key": "value"}`),
			expected: true,
		},
		{
			name:     "valid json array of objects",
			input:    []byte(`[{"key": "value"}]`),
			expected: true,
		},
		{
			name:     "valid json array of objects with whitespace",
			input:    []byte(`  [  {"key": "value"}]`),
			expected: true,
		},
		{
			name:     "valid json array of objects with mixed whitespace",
			input:    []byte(`	 [ 	{"key": "value"}]`),
			expected: true,
		},
		{
			name:     "simple json array",
			input:    []byte(`["a", "b", "c"]`),
			expected: false,
		},
		{
			name:     "simple json array with whitespace",
			input:    []byte(`  ["a", "b", "c"]`),
			expected: false,
		},
		{
			name:     "empty json object",
			input:    []byte(`{}`),
			expected: true,
		},
		{
			name:     "empty json array",
			input:    []byte(`[]`),
			expected: false,
		},
		{
			name:     "plain text",
			input:    []byte(`this is not json`),
			expected: false,
		},
		{
			name:     "empty buffer",
			input:    []byte(``),
			expected: false,
		},
		{
			name:     "only whitespace",
			input:    []byte(`   `),
			expected: false,
		},
		{
			name:     "json string",
			input:    []byte(`"hello world"`),
			expected: false,
		},
		{
			name:     "json number",
			input:    []byte(`42`),
			expected: false,
		},
		{
			name:     "json boolean",
			input:    []byte(`true`),
			expected: false,
		},
		{
			name:     "json null",
			input:    []byte(`null`),
			expected: false,
		},
		{
			name:     "malformed json starting with brace",
			input:    []byte(`{invalid json`),
			expected: true,
		},
		{
			name:     "nested json object",
			input:    []byte(`{"outer": {"inner": "value"}}`),
			expected: true,
		},
		{
			name:     "complex json object",
			input:    []byte(`{"name": "test", "items": [{"id": 1}, {"id": 2}]}`),
			expected: true,
		},
		{
			name:     "array starting with object",
			input:    []byte(`[{"first": true}, "second", 3]`),
			expected: true,
		},
		{
			name:     "newlines and spaces",
			input:    []byte("  \n  \t  {\n  \"key\": \"value\"\n}"),
			expected: true,
		},
		{
			name:     "newlines before array with object",
			input:    []byte("  \n  [\n  {\"key\": \"value\"}\n]"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsJSON(tt.input)
			if result != tt.expected {
				t.Errorf("IsJSON(%q) = %v, expected %v", string(tt.input), result, tt.expected)
			}
		})
	}
}
