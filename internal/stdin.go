package internal

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// readStdinArgs reads STDIN and appends it to the list of arguments.
// It's useful when we want to support piping.
func readStdinArgs(cmd *cobra.Command, args []string) ([]string, error) {
	input := cmd.InOrStdin()
	if file, ok := input.(*os.File); ok {
		stat, err := file.Stat()
		if err != nil {
			return nil, fmt.Errorf("stat stdin: %w", err)
		}
		if stat.Mode()&os.ModeCharDevice != 0 {
			return args, nil
		}
	}

	data, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	names := make([]string, 0, len(args))
	names = append(names, args...)
	names = append(names, strings.Fields(string(data))...)
	return names, nil
}
