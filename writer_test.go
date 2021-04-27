package splitwriter_test

import (
	"github.com/gogolibs/splitwriter"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

type testCase struct {
	name   string
	input  []string
	result []string
}

var splitCases = []struct {
	name      string
	split     splitwriter.SplitFunc
	testCases []testCase
}{
	{
		name:  "scan bytes",
		split: splitwriter.ScanBytes,
		testCases: []testCase{
			{
				input:  []string{"abc"},
				result: []string{"a", "b", "c"},
			},
		},
	},
	{
		name:  "scan lines",
		split: splitwriter.ScanLines,
		testCases: []testCase{
			{
				name:   "one write",
				input:  []string{"hello\nworld\n"},
				result: []string{"hello", "world"},
			},
			{
				name:   "two writes",
				input:  []string{"one\ntwo", "\nthree\nfour\n"},
				result: []string{"one", "two", "three", "four"},
			},
		},
	},
}

func TestCases(t *testing.T) {
	for _, splitCase := range splitCases {
		t.Run(splitCase.name, func(t *testing.T) {
			for _, testCase := range splitCase.testCases {
				t.Run(testCase.name, func(t *testing.T) {
					tw := &testWriter{[]string{}}
					w := splitwriter.New(tw)
					w.Split(splitCase.split)
					for _, inputPart := range testCase.input {
						inputData := []byte(inputPart)
						bytesWritten, err := w.Write(inputData)
						require.NoError(t, err)
						require.Equal(t, len(inputData), bytesWritten)
					}
					require.Equal(t, testCase.result, tw.result)
				})
			}
		})
	}
}

type mockWriter struct {
	mock.Mock
}

func (w *mockWriter) Write(data []byte) (int, error) {
	args := w.Called(data)
	return args.Int(0), args.Error(1)
}

func TestLongWrite(t *testing.T) {
	m := new(mockWriter)
	w := splitwriter.New(m)
	m.On("Write", []byte("one")).Return(3, nil)
	m.On("Write", []byte("two")).Return(42, nil)
	bytesWritten, err := w.Write([]byte("one\ntwo\n"))
	require.Error(t, err)
	require.Equal(t, "long write: have 42 want 3 bytes", err.Error())
	require.Equal(t, 4, bytesWritten)
}

func TestShortWrite(t *testing.T) {
	m := new(mockWriter)
	w := splitwriter.New(m)
	m.On("Write", []byte("one")).Return(3, nil)
	m.On("Write", []byte("two")).Return(1, nil)
	bytesWritten, err := w.Write([]byte("one\ntwo\n"))
	require.Error(t, err)
	require.Equal(t, "short write: have 1 want 3 bytes", err.Error())
	require.Equal(t, 4, bytesWritten)
}

func TestPartialWrite(t *testing.T) {
	m := new(mockWriter)
	w := splitwriter.New(m)
	m.On("Write", []byte("one")).Return(3, errors.New("write err"))
	bytesWritten, err := w.Write([]byte("one"))
	require.NoError(t, err)
	require.Equal(t, 3, bytesWritten)
	require.Equal(t, 3, w.BufferLen())
	bytesWritten, err = w.Write([]byte("\n"))
	require.Error(t, err)
	require.Equal(t, "failed to write: write err", err.Error())
	require.Equal(t, 0, bytesWritten)
}
