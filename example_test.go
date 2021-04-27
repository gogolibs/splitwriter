package splitwriter_test

import (
	"github.com/gogolibs/splitwriter"
	"github.com/stretchr/testify/require"
	"testing"
)

type testWriter struct {
	result []string
}

func (w *testWriter) Write(data []byte) (int, error) {
	w.result = append(w.result, string(data))
	return len(data), nil
}

func TestExample(t *testing.T) {
	underlyingWriter := &testWriter{result: []string{}}
	splitWriter := splitwriter.New(underlyingWriter)
	splitWriter.Split(splitwriter.ScanLines)
	data := []byte("one\ntwo\nthree\nincomplete line")
	bytesWritten, err := splitWriter.Write(data)
	require.NoError(t, err)
	require.Equal(t, len(data), bytesWritten)
	require.Equal(t, []string{
		"one",
		"two",
		"three",
	}, underlyingWriter.result)
	require.Equal(t, len([]byte("incomplete line")), splitWriter.BufferLen())
}
