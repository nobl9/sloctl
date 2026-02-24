package yamlenc

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncoder(t *testing.T) {
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	assert.NotNil(t, encoder)
}

func TestEncoder_BasicTypes(t *testing.T) {
	tests := map[string]struct {
		input    any
		expected string
	}{
		"simple string": {
			input:    "hello",
			expected: "hello\n",
		},
		"integer": {
			input:    42,
			expected: "42\n",
		},
		"float": {
			input:    3.14,
			expected: "3.14\n",
		},
		"boolean true": {
			input:    true,
			expected: "true\n",
		},
		"boolean false": {
			input:    false,
			expected: "false\n",
		},
		"nil": {
			input:    nil,
			expected: "null\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			encoder := NewEncoder(&buf)

			err := encoder.Encode(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestEncoder_StructEncoding(t *testing.T) {
	type TestStruct struct {
		Name string `yaml:"name"`
		Age  int    `yaml:"age"`
	}

	tests := map[string]struct {
		input    any
		validate func(t *testing.T, output string)
	}{
		"simple struct": {
			input: TestStruct{Name: "Alice", Age: 30},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "name: Alice")
				assert.Contains(t, output, "age: 30")
			},
		},
		"map": {
			input: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "key1: value1")
				assert.Contains(t, output, "key2: 42")
			},
		},
		"slice": {
			input: []string{"a", "b", "c"},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "- a")
				assert.Contains(t, output, "- b")
				assert.Contains(t, output, "- c")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			encoder := NewEncoder(&buf)

			err := encoder.Encode(tc.input)
			require.NoError(t, err)
			tc.validate(t, buf.String())
		})
	}
}

func TestEncoder_IndentSequence(t *testing.T) {
	input := map[string]any{
		"items": []string{"one", "two", "three"},
	}

	var buf bytes.Buffer
	encoder := NewEncoder(&buf)

	err := encoder.Encode(input)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "items:")
	assert.Contains(t, output, "  - one")
	assert.Contains(t, output, "  - two")
	assert.Contains(t, output, "  - three")
}

func TestEncoder_MultilineString(t *testing.T) {
	input := map[string]any{
		"description": "This is a\nmultiline\nstring",
	}

	var buf bytes.Buffer
	encoder := NewEncoder(&buf)

	err := encoder.Encode(input)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "description: |-")
	assert.Contains(t, output, "This is a")
	assert.Contains(t, output, "multiline")
	assert.Contains(t, output, "string")
}

func TestYamlNumberMarshaler(t *testing.T) {
	tests := map[string]struct {
		input    json.Number
		expected string
	}{
		"integer": {
			input:    json.Number("42"),
			expected: "42",
		},
		"float": {
			input:    json.Number("3.14"),
			expected: "3.14",
		},
		"large number": {
			input:    json.Number("1234567890"),
			expected: "1234567890",
		},
		"negative number": {
			input:    json.Number("-42"),
			expected: "-42",
		},
		"zero": {
			input:    json.Number("0"),
			expected: "0",
		},
		"scientific notation": {
			input:    json.Number("1.23e10"),
			expected: "1.23e10",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := yamlNumberMarshaler(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(result))
		})
	}
}

func TestEncoder_WithJSONNumber(t *testing.T) {
	tests := map[string]struct {
		input    map[string]any
		validate func(t *testing.T, output string)
	}{
		"integer as json.Number": {
			input: map[string]any{
				"count": json.Number("42"),
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "count: 42")
				assert.NotContains(t, output, "42.0")
			},
		},
		"float as json.Number": {
			input: map[string]any{
				"value": json.Number("3.14"),
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "value: 3.14")
			},
		},
		"mixed types with json.Number": {
			input: map[string]any{
				"name":  "test",
				"count": json.Number("100"),
				"rate":  json.Number("99.5"),
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "name: test")
				assert.Contains(t, output, "count: 100")
				assert.Contains(t, output, "rate: 99.5")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			encoder := NewEncoder(&buf)

			err := encoder.Encode(tc.input)
			require.NoError(t, err)
			tc.validate(t, buf.String())
		})
	}
}

func TestEncoder_NestedStructures(t *testing.T) {
	input := map[string]any{
		"metadata": map[string]any{
			"name": "test-object",
			"labels": map[string]string{
				"env":  "prod",
				"team": "platform",
			},
		},
		"spec": map[string]any{
			"replicas": json.Number("3"),
			"ports": []map[string]any{
				{"name": "http", "port": json.Number("80")},
				{"name": "https", "port": json.Number("443")},
			},
		},
	}

	var buf bytes.Buffer
	encoder := NewEncoder(&buf)

	err := encoder.Encode(input)
	require.NoError(t, err)

	output := buf.String()

	// Verify structure
	var decoded map[string]any
	err = yaml.Unmarshal([]byte(output), &decoded)
	require.NoError(t, err)

	// Verify content
	assert.Equal(t, "test-object", decoded["metadata"].(map[string]any)["name"])
	// When YAML is unmarshaled, numbers become native types (int, uint, float)
	// so we verify the numeric value rather than the string representation
	assert.EqualValues(t, 3, decoded["spec"].(map[string]any)["replicas"])
}

func TestEncoder_EmptyValues(t *testing.T) {
	tests := map[string]struct {
		input    any
		expected string
	}{
		"empty map": {
			input:    map[string]any{},
			expected: "{}\n",
		},
		"empty slice": {
			input:    []string{},
			expected: "[]\n",
		},
		"empty string": {
			input:    "",
			expected: "\"\"\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			encoder := NewEncoder(&buf)

			err := encoder.Encode(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, buf.String())
		})
	}
}
