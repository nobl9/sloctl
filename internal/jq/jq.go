package jq

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"strings"

	"github.com/itchyny/gojq"
)

func NewExpressionRunner(config Config) ExpressionRunner {
	return ExpressionRunner{config: config}
}

type ExpressionRunner struct {
	config Config
}

type Config struct {
	Expression string
}

func (e ExpressionRunner) ShouldRun() bool {
	return e.config.Expression != ""
}

// Evaluate parses, compiles and runs jq expressions.
// It returns an iterator which yields jq expression result and error (if any).
func (e ExpressionRunner) Evaluate(v any) (iter.Seq2[any, error], error) {
	query, err := gojq.Parse(e.config.Expression)
	if err != nil {
		var parseErr *gojq.ParseError
		if errors.As(err, &parseErr) {
			str, line, column := getLineColumn(e.config.Expression, parseErr.Offset-len(parseErr.Token))
			return nil, fmt.Errorf(
				"failed to parse jq expression (line %d, column %d)\n    %s\n    %*c  %w",
				line, column, str, column, '^', err,
			)
		}
		return nil, err
	}

	code, err := gojq.Compile(query, gojq.WithEnvironLoader(os.Environ))
	if err != nil {
		return nil, err
	}

	anyValue, err := toAny(v)
	if err != nil {
		return nil, err
	}

	return func(yield func(any, error) bool) {
		iter := code.Run(anyValue)
		for {
			v, ok := iter.Next()
			if !ok {
				return
			}
			switch v := v.(type) {
			case error:
				var haltErr *gojq.HaltError
				if errors.As(v, &haltErr) && haltErr.Value() == nil {
					break
				}
				if !yield(nil, v) {
					return
				}
			default:
				if !yield(v, nil) {
					return
				}
			}
		}
	}, nil
}

func getLineColumn(expr string, offset int) (str string, line, newOffset int) {
	for line := 1; ; line++ {
		index := strings.Index(expr, "\n")
		if index < 0 {
			return expr, line, offset + 1
		}
		if index >= offset {
			return expr[:index], line, offset + 1
		}
		expr = expr[index+1:]
		offset -= index + 1
	}
}

func toAny(v any) (any, error) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var anyValue any
	err = json.Unmarshal(jsonData, &anyValue)
	if err != nil {
		return nil, err
	}
	return anyValue, nil
}
