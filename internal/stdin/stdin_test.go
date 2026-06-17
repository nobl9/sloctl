package stdin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadArgs(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("service-a service-b\nservice-c\n"))

	got, err := ReadArgs(cmd, []string{"service-existing"})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"service-existing",
		"service-a",
		"service-b",
		"service-c",
	}, got)
}

func Test_readData_ReadsNonFileReader(t *testing.T) {
	t.Parallel()

	got, err := readData(strings.NewReader("agent-one\nagent-two\n"))
	require.NoError(t, err)
	assert.Equal(t, "agent-one\nagent-two\n", string(got))
}

func Test_readData_ReadsRegularFile(t *testing.T) {
	t.Parallel()

	file := tempFile(t, "project-a\nproject-b\n")
	got, err := readData(file)
	require.NoError(t, err)
	assert.Equal(t, "project-a\nproject-b\n", string(got))
}

func Test_readData_ReadsClosedPipe(t *testing.T) {
	reader, writer := pipe(t)
	t.Cleanup(func() {
		_ = writer.Close()
		_ = reader.Close()
	})

	_, err := writer.WriteString("service-a\nservice-b\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	got, err := readData(reader)
	require.NoError(t, err)
	assert.Equal(t, "service-a\nservice-b\n", string(got))
}

func Test_readData_ReadsDelayedPipe(t *testing.T) {
	reader, writer := pipe(t)
	t.Cleanup(func() {
		_ = writer.Close()
		_ = reader.Close()
	})

	resultCh := make(chan readOutcome, 1)
	go func() {
		data, err := readData(reader)
		resultCh <- readOutcome{data: data, err: err}
	}()

	time.Sleep(100 * time.Millisecond)
	_, err := writer.WriteString("delayed-user\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	select {
	case result := <-resultCh:
		require.NoError(t, result.err)
		assert.Equal(t, "delayed-user\n", string(result.data))
	case <-time.After(time.Second):
		require.FailNow(t, "readData() did not return after pipe writer closed")
	}
}

func tempFile(t *testing.T, contents string) *os.File {
	t.Helper()

	path := filepath.Join(t.TempDir(), "stdin.txt")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	file, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = file.Close()
	})
	return file
}

func pipe(t *testing.T) (reader, writer *os.File) {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	return reader, writer
}

type readOutcome struct {
	err  error
	data []byte
}
