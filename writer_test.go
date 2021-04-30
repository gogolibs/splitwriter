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
			{
				name:   "two buffered writes",
				input:  []string{"hello ", "world", "\n"},
				result: []string{"hello world"},
			},
		},
	},
}

func TestCases(t *testing.T) {
	for _, splitCase := range splitCases {
		t.Run(splitCase.name, func(t *testing.T) {
			for _, testCase := range splitCase.testCases {
				t.Run(testCase.name, func(t *testing.T) {
					result := []string{}
					w := splitwriter.NewWriterFunc(func(token []byte) error {
						result = append(result, string(token))
						return nil
					}).Split(splitCase.split)
					for _, inputPart := range testCase.input {
						inputData := []byte(inputPart)
						bytesWritten, err := w.Write(inputData)
						require.NoError(t, err)
						require.Equal(t, len(inputData), bytesWritten)
					}
					require.Equal(t, testCase.result, result)
				})
			}
		})
	}
}

type mockHandler struct {
	mock.Mock
}

func (h *mockHandler) Handle(token []byte) error {
	args := h.Called(token)
	return args.Error(0)
}

func TestHandlerErrBuffered(t *testing.T) {
	h := new(mockHandler)
	w := splitwriter.NewWriter(h)
	h.On("Handle", []byte("one")).Return(errors.New("handle err")).Once()
	h.On("Handle", []byte("one")).Return(nil).Once()
	bytesWritten, err := w.Write([]byte("one"))
	require.NoError(t, err)
	require.Equal(t, 3, bytesWritten)
	require.Equal(t, 3, w.BufferLen())
	bytesWritten, err = w.Write([]byte("\n"))
	require.Error(t, err)
	require.Equal(t, `failed to handle token "one": handle err`, err.Error())
	require.Equal(t, 0, bytesWritten)
	bytesWritten, err = w.Write([]byte("\n"))
	require.NoError(t, err)
	require.Equal(t, 1, bytesWritten)
}

func TestHandlerErrEmptyBuffer(t *testing.T) {
	h := new(mockHandler)
	w := splitwriter.NewWriter(h)
	h.On("Handle", []byte("one")).Return(errors.New("handle err"))
	bytesWritten, err := w.Write([]byte("one\n"))
	require.Error(t, err)
	require.Equal(t, 0, bytesWritten)
	require.Equal(t, `failed to handle token "one": handle err`, err.Error())
}

func TestRecoverFromErrEmptyBuffer(t *testing.T) {
	h := new(mockHandler)
	w := splitwriter.NewWriter(h)
	h.On("Handle", []byte("one")).Return(errors.New("handle err"))
	h.On("Handle", []byte("two")).Return(nil)
	bytesWritten, err := w.Write([]byte("one\n"))
	require.Error(t, err)
	require.Equal(t, 0, bytesWritten)
	require.Equal(t, `failed to handle token "one": handle err`, err.Error())
	bytesWritten, err = w.Write([]byte("two\n"))
	require.NoError(t, err)
	require.Equal(t, 4, bytesWritten)
}

func TestRecoverFromErrBuffered(t *testing.T) {
	h := new(mockHandler)
	w := splitwriter.NewWriter(h)
	h.On("Handle", []byte("buf one")).Return(errors.New("handle err"))
	h.On("Handle", []byte("buf two")).Return(nil)
	bytesWritten, err := w.Write([]byte("buf "))
	require.NoError(t, err)
	require.Equal(t, 4, bytesWritten)
	bytesWritten, err = w.Write([]byte("one\n"))
	require.Error(t, err)
	require.Equal(t, 0, bytesWritten)
	require.Equal(t, `failed to handle token "buf one": handle err`, err.Error())
	bytesWritten, err = w.Write([]byte("two\n"))
	require.NoError(t, err)
	require.Equal(t, 4, bytesWritten)
}

func TestSplitErrEmptyBuffer(t *testing.T) {
	h := new(mockHandler)
	h.On("Handle").Panic("Handle must not be called in this test")
	w := splitwriter.NewWriter(h).Split(func(data []byte) (advance int, token []byte, err error) {
		return 0, nil, errors.New("split err")
	})
	bytesWritten, err := w.Write([]byte("hello"))
	require.Error(t, err)
	require.Equal(t, 0, bytesWritten)
	require.Equal(t, "failed to split: split err", err.Error())
}

func TestSplitErrBuffered(t *testing.T) {
	h := new(mockHandler)
	h.On("Handle").Panic("Handle must not be called in this test")
	w := splitwriter.NewWriter(h).Split(func(data []byte) (advance int, token []byte, err error) {
		if string(data) == "one" {
			return 0, nil, nil
		}
		return 0, nil, errors.New("split err")
	})
	bytesWritten, err := w.Write([]byte("one"))
	require.NoError(t, err)
	require.Equal(t, 3, bytesWritten)
	bytesWritten, err = w.Write([]byte("two"))
	require.Error(t, err)
	require.Equal(t, 0, bytesWritten)
	require.Equal(t, "failed to split: split err", err.Error())
}

func TestCallSplitAfterWrite(t *testing.T) {
	h := new(mockHandler)
	h.On("Handle").Panic("Handle must not be called in this test")
	w := splitwriter.NewWriter(h)
	bytesWritten, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, bytesWritten)
	defer func() {
		r := recover()
		require.Equal(t, "Split called after Write", r)
	}()
	w.Split(splitwriter.ScanRunes)
}
