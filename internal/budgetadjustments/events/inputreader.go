package events

import (
	"io"
	"os"
	"path/filepath"
)

func readFile(path string) (data []byte, err error) {
	if path == "" || path == "-" {
		return io.ReadAll(os.Stdin)
	}
	path = filepath.Clean(path)
	return os.ReadFile(path) // #nosec G304
}
