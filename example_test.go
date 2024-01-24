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
