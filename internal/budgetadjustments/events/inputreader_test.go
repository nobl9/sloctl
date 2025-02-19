package events

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitYAMLDocuments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "Basic YAML Split",
			input: `---
name: doc1
value: 1
---
name: doc2
value: 2
---
name: doc3
value: 3
		`,
			expected: []string{
				"name: doc1\nvalue: 1",
				"name: doc2\nvalue: 2",
				"name: doc3\nvalue: 3",
			},
		},
		{
			name: "Basic YAML Split with additional separators",
			input: `---
---
---
name: doc1
value: 1
---
---
name: doc2
value: 2
---
name: doc3
value: 3
---
---`,
			expected: []string{
				"name: doc1\nvalue: 1",
				"name: doc2\nvalue: 2",
				"name: doc3\nvalue: 3",
			},
		},
		{
			name: "YAML with Lists",
			input: `---
list:
  - item1
  - item2
---
list:
  - item3
  - item4
`,
			expected: []string{
				"list:\n  - item1\n  - item2",
				"list:\n  - item3\n  - item4",
			},
		},
		{
			name: "YAML with Nested Structures",
			input: `---
parent:
  child: value1
---
parent:
  child: value2
`,
			expected: []string{
				"parent:\n  child: value1",
				"parent:\n  child: value2",
			},
		},
		{
			name: "invalid YAML",
			input: `---
foo bar
baz: bob
`,
			expected: []string{"foo bar\nbaz: bob"},
		},
		{
			name:     "just YAML",
			input:    "YAML",
			expected: []string{"YAML"},
		},
		{
			name: "YAML with doc separators found in content",
			input: `---
parent:
  child: "foo---bar"
---
parent:
  child: value2
---
----nt:
  child: value3
----
---
 ---
 --- 
`,
			expected: []string{
				"parent:\n  child: \"foo---bar\"",
				"parent:\n  child: value2",
				"----nt:\n  child: value3\n----",
				"---\n ---",
			},
		},
		{
			name: "YAML with correct event format",
			input: `
- eventStart: 2024-12-04T06:37:00Z
  eventEnd: 2024-12-04T06:59:00Z
  slos:
    - project: test-project
      name: sample-slo-1-578b974d-8e27-43cf-85a3-7751a774f13d
  update:
    eventStart: 2024-12-04T06:37:00Z
    eventEnd: 2024-12-04T06:59:00Z
`,
			expected: []string{
				`- eventStart: 2024-12-04T06:37:00Z
  eventEnd: 2024-12-04T06:59:00Z
  slos:
    - project: test-project
      name: sample-slo-1-578b974d-8e27-43cf-85a3-7751a774f13d
  update:
    eventStart: 2024-12-04T06:37:00Z
    eventEnd: 2024-12-04T06:59:00Z`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitYAMLDocs([]byte(tt.input))

			// Trim whitespace from results for better comparison
			for i := range result {
				result[i] = strings.TrimSpace(result[i])
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Test %s failed. Expected %v, but got %v", tt.name, tt.expected, result)
			}
		})
	}
}
