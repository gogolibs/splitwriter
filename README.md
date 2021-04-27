# splitwriter #

[![GoDoc](https://godoc.org/github.com/gogolibs/splitwriter?status.svg)](https://pkg.go.dev/github.com/gogolibs/splitwriter)
[![Go Report Card](https://goreportcard.com/badge/github.com/gogolibs/splitwriter?style=flat)](https://goreportcard.com/report/github.com/gogolibs/splitwriter)
[![CI](https://github.com/gogolibs/splitwriter/actions/workflows/test-and-coverage.yml/badge.svg)](https://github.com/gogolibs/splitwriter/actions/workflows/test-and-coverage.yml)
[![codecov](https://codecov.io/gh/gogolibs/splitwriter/branch/main/graph/badge.svg?token=Nbd92Hkjl6)](https://codecov.io/gh/gogolibs/splitwriter)

**splitwriter** is utility package that provides a wrapper around `io.Writer`
that tokenizes data supplied to `Write` using `bufio.SplitFunc`, 
buffers it as necessary and writes tokens via separate writes to the underlying writer.

```go
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
```