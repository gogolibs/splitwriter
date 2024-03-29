# splitwriter #

[![GoDoc](https://godoc.org/github.com/gogolibs/splitwriter?status.svg)](https://pkg.go.dev/github.com/gogolibs/splitwriter)
[![Go Report Card](https://goreportcard.com/badge/github.com/gogolibs/splitwriter?style=flat)](https://goreportcard.com/report/github.com/gogolibs/splitwriter)
[![CI](https://github.com/gogolibs/splitwriter/actions/workflows/test-and-coverage.yml/badge.svg)](https://github.com/gogolibs/splitwriter/actions/workflows/test-and-coverage.yml)
[![codecov](https://codecov.io/gh/gogolibs/splitwriter/branch/main/graph/badge.svg?token=Nbd92Hkjl6)](https://codecov.io/gh/gogolibs/splitwriter)

**splitwriter** provides an `io.Writer` implementation that
tokenizes the data written to it using `bufio.SplitFunc`
and allows to handle tokens by providing an implementation of `splitwriter.Handler`
or just a simple `splitwriter.HandlerFunc`. A practical example:

```go
package splitwriter_test

import (
	"github.com/gogolibs/splitwriter"
	"github.com/stretchr/testify/assert"
	"io"
	"strings"
	"testing"
)

func TestExample(t *testing.T) {
	reader := strings.NewReader("one\ntwo\nthree\n")
	var result []string
	writer := splitwriter.New(splitwriter.HandlerFunc(func(token []byte) error {
		result = append(result, string(token))
		return nil
	})).Split(splitwriter.ScanLines)
	_, err := io.Copy(writer, reader)
	assert.NoError(t, err)
	assert.Equal(t, []string{"one", "two", "three"}, result)
}
```