package internal

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskField(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{in: "", out: ""},
		{in: "asd", out: "***"},
		{in: "foo-ba", out: "***"},
		{in: "foo-bar", out: "fo***ar"},
		{in: "super-secret-long-string", out: "su***ng"},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := maskField(test.in)
			assert.Equal(t, test.out, out)
		})
	}
}
