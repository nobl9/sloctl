package events

import (
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
)

func readFile(path string) (data []byte, err error) {
	if path == "" || path == "-" {
		return io.ReadAll(os.Stdin)
	}
	path = filepath.Clean(path)
	return os.ReadFile(path) // #nosec G304
}

func getEventsStringsFromFile(path string) ([]string, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read input data")
	}
	return splitYAMLDocs(data), nil
}

func splitYAMLDocs(data []byte) []string {
	re := regexp.MustCompile("(?m)^---$\n?")
	split := re.Split(string(data), -1)
	docs := make([]string, 0, len(split))
	for _, docStr := range split {
		if len(docStr) < 1 {
			continue
		}
		docs = append(docs, docStr)
	}
	return docs
}
