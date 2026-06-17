package stdin

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// ReadArgs reads STDIN and appends it to the list of arguments.
// It's useful when we want to support piping.
func ReadArgs(cmd *cobra.Command, args []string) ([]string, error) {
	data, err := readData(cmd.InOrStdin())
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	names := make([]string, 0, len(args))
	names = append(names, args...)
	names = append(names, strings.Fields(string(data))...)
	return names, nil
}

// readData returns data from input unless input is an interactive terminal.
func readData(input io.Reader) ([]byte, error) {
	file, ok := input.(*os.File)
	if !ok {
		return io.ReadAll(input)
	}

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat stdin: %w", err)
	}
	if stat.Mode()&os.ModeCharDevice != 0 {
		return nil, nil
	}

	return io.ReadAll(file)
}
