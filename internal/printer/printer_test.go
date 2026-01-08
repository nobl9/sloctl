package printer

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobl9/sloctl/internal/csv"
)

type testStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestPrinter_Print(t *testing.T) {
	tests := map[string]struct {
		config   Config
		v        any
		expected string
		err      error
	}{
		"default YAML": {
			config: Config{},
			v:      testStruct{Name: "foo", Age: 30},
			expected: `name: foo
age: 30
`,
		},
		"YAML": {
			config: Config{OutputFormat: YAMLFormat},
			v:      testStruct{Name: "foo", Age: 30},
			expected: `name: foo
age: 30
`,
		},
		"JSON": {
			config: Config{OutputFormat: JSONFormat},
			v:      testStruct{Name: "foo", Age: 30},
			expected: `{
  "name": "foo",
  "age": 30
}
`,
		},
		"JSON string scalar": {
			config:   Config{OutputFormat: JSONFormat},
			v:        "hello world",
			expected: "hello world\n",
		},
		"JSON integer scalar": {
			config:   Config{OutputFormat: JSONFormat},
			v:        int(42),
			expected: "42\n",
		},
		"JSON float scalar with no decimal": {
			config:   Config{OutputFormat: JSONFormat},
			v:        float64(42),
			expected: "42\n",
		},
		"JSON decimal scalar": {
			config:   Config{OutputFormat: JSONFormat},
			v:        float64(42.56),
			expected: "42.56\n",
		},
		"JSON decimal scalar (big decimal)": {
			config:   Config{OutputFormat: JSONFormat},
			v:        float64(42.560918230975),
			expected: "42.560918230975\n",
		},
		"JSON nil scalar": {
			config:   Config{OutputFormat: JSONFormat},
			v:        nil,
			expected: "",
		},
		"JSON bool true scalar": {
			config:   Config{OutputFormat: JSONFormat},
			v:        true,
			expected: "true\n",
		},
		"JSON bool false scalar": {
			config:   Config{OutputFormat: JSONFormat},
			v:        false,
			expected: "false\n",
		},
		"CSV": {
			config: Config{OutputFormat: CSVFormat},
			v:      testStruct{Name: "foo", Age: 30},
			expected: `age,name
30,"foo"
`,
		},
		"CSV list": {
			config: Config{OutputFormat: CSVFormat},
			v:      []testStruct{{Name: "foo", Age: 30}, {Name: "bar", Age: 40}},
			expected: `age,name
30,"foo"
40,"bar"
`,
		},
		"CSV - custom record separator": {
			config:   Config{OutputFormat: CSVFormat, CSVRecordSeparator: "|"},
			v:        []testStruct{{Name: "foo", Age: 30}, {Name: "bar", Age: 40}},
			expected: `age,name|30,"foo"|40,"bar"|`,
		},
		"CSV - custom field separator": {
			config: Config{OutputFormat: CSVFormat, CSVFieldSeparator: "|"},
			v:      testStruct{Name: "foo", Age: 30},
			expected: `age|name
30|"foo"
`,
		},
		"invalid format": {
			config: Config{OutputFormat: "foo", CSVFieldSeparator: "|"},
			err:    errors.New(`unknown output format "foo"`),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			buf := bytes.Buffer{}
			tc.config.Output = &buf

			o := NewPrinter(tc.config)
			err := o.Print(tc.v)

			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, buf.String())
			}
		})
	}
}

func TestPrinter_MustRegisterFlags(t *testing.T) {
	defaultConfig := Config{
		Output:             os.Stdout,
		OutputFormat:       YAMLFormat,
		CSVFieldSeparator:  csv.DefaultFieldSeparator,
		CSVRecordSeparator: csv.DefaultRecordSeparator,
	}

	tests := map[string]struct {
		args        []string
		getExpected func() Config
		err         error
	}{
		"invalid format": {
			args: []string{"-o", "foo"},
			err:  errors.New(`invalid argument "foo" for "-o, --output" flag: invalid value for Format: foo`),
		},
		"full output flag (JSON)": {
			args: []string{"--output", "json"},
			getExpected: func() Config {
				conf := defaultConfig
				conf.OutputFormat = JSONFormat
				conf.SupportedFromats = ObjectsSupportedFormats
				return conf
			},
		},
		"short output flag (CSV)": {
			args: []string{"-o", "yaml"},
			getExpected: func() Config {
				conf := defaultConfig
				conf.OutputFormat = YAMLFormat
				conf.SupportedFromats = ObjectsSupportedFormats
				return conf
			},
		},
		"csv flags": {
			args: []string{
				"-o", "csv",
				"--record-separator", "|",
				"--field-separator", "~",
			},
			getExpected: func() Config {
				conf := defaultConfig
				conf.OutputFormat = CSVFormat
				conf.CSVRecordSeparator = "|"
				conf.CSVFieldSeparator = "~"
				conf.SupportedFromats = ObjectsSupportedFormats
				return conf
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			o := NewPrinter(Config{})
			cmd := &cobra.Command{}
			o.MustRegisterFlags(cmd)
			err := cmd.ParseFlags(tc.args)

			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.getExpected(), o.config)
			}
		})
	}
}
