package events

import (
	"io"
	"os"
)

func readFile(path string) (data []byte, err error) {
	if path == "" || path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path) // #nosec G304
}
