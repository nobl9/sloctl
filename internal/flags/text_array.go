package flags

import (
	"encoding"
	"fmt"
	"reflect"
)

type textEncoder interface {
	fmt.Stringer
	encoding.TextUnmarshaler
}

type TextArray[T textEncoder] []T

func (a TextArray[T]) String() string {
	if len(a) == 0 {
		return ""
	}
	s := make([]string, 0, len(a))
	for _, v := range a {
		s = append(s, v.String())
	}
	str, _ := writeAsCSV(s)
	return "[" + str + "]"
}

func (a *TextArray[T]) Set(s string) error {
	values, err := readAsCSV(s)
	if err != nil {
		return err
	}
	if a == nil {
		a = new(TextArray[T])
		*a = make([]T, 0, len(values))
	}
	for _, v := range values {
		var vt T
		decoded := reflect.New(reflect.TypeOf(vt).Elem()).Interface().(T)
		if err := decoded.UnmarshalText([]byte(v)); err != nil {
			return err
		}
		*a = append(*a, decoded)
	}
	return nil
}

func (a TextArray[T]) Type() string {
	return "textArray"
}
