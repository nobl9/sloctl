package flags

import (
	"strconv"

	"github.com/spf13/pflag"
)

var _ pflag.Value = &FloatArray{}

type FloatArray []float64

func (a FloatArray) String() string {
	if len(a) == 0 {
		return ""
	}
	s := make([]string, 0, len(a))
	for _, f := range a {
		s = append(s, strconv.FormatFloat(f, 'f', 2, 64))
	}
	str, _ := writeAsCSV(s)
	return "[" + str + "]"
}

func (a *FloatArray) Set(s string) error {
	values, err := readAsCSV(s)
	if err != nil {
		return err
	}
	if a == nil {
		a = new(FloatArray)
		*a = make(FloatArray, 0, len(values))
	}
	for _, v := range values {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		*a = append(*a, f)
	}
	return nil
}

func (a FloatArray) Type() string {
	return "float64Array"
}
