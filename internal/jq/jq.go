package jq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/itchyny/gojq"
)

type printerInterface interface {
	Print(v any) error
}

func NewExpressionRunner(config Config) *ExpressionRunner {
	return &ExpressionRunner{config: config}
}

type ExpressionRunner struct {
	config Config
}

type Config struct {
	Printer    printerInterface
	Expression string
}

func (e *ExpressionRunner) ShouldRun() bool {
	return e.config.Expression != ""
}

func (e *ExpressionRunner) EvaluateAndPrint(ctx context.Context, v any) error {
	query, err := gojq.Parse(e.config.Expression)
	if err != nil {
		var parseErr *gojq.ParseError
		if errors.As(err, &parseErr) {
			str, line, column := getLineColumn(e.config.Expression, parseErr.Offset-len(parseErr.Token))
			return fmt.Errorf(
				"failed to parse jq expression (line %d, column %d)\n    %s\n    %*c  %w",
				line, column, str, column, '^', err,
			)
		}
		return err
	}

	code, err := gojq.Compile(query, gojq.WithEnvironLoader(os.Environ))
	if err != nil {
		return err
	}

	anyValue, err := toAny(v)
	if err != nil {
		return err
	}

	iter := code.RunWithContext(ctx, anyValue)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		switch v := v.(type) {
		case error:
			var haltErr *gojq.HaltError
			if errors.As(v, &haltErr) && haltErr.Value() == nil {
				break
			}
			return v
		default:
			if err = e.config.Printer.Print(v); err != nil {
				return err
			}
		}
	}
	return nil
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
