package splitwriter_test

import (
	"github.com/gogolibs/splitwriter"
	"github.com/stretchr/testify/require"
	"os/exec"
	"testing"
)

func TestExample(t *testing.T) {
	cmd := exec.Command("echo", "-e", "one\ntwo\nthree")
	var result []string
	cmd.Stdout = splitwriter.NewWriterFunc(func(token []byte) error {
		result = append(result, string(token))
		return nil
	}).Split(splitwriter.ScanLines)
	err := cmd.Run()
	require.NoError(t, err)
	require.Equal(t, []string{"one", "two", "three"}, result)
}
