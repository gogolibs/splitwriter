package splitwriter_test

import (
	"github.com/gogolibs/splitwriter"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"strings"
	"testing"
)

func TestCases(t *testing.T) {
	type testCase struct {
		input    []string
		expected []string
	}
	testCases := map[string]struct {
		split     splitwriter.SplitFunc
		testCases map[string]testCase
	}{
		"scan bytes": {
			split: splitwriter.ScanBytes,
			testCases: map[string]testCase{
				"simple": {
					input:    []string{"abc"},
					expected: []string{"a", "b", "c"},
				},
			},
		},
		"scan lines": {
			split: splitwriter.ScanLines,
			testCases: map[string]testCase{
				"one write": {
					input:    []string{"hello\nworld\n"},
					expected: []string{"hello", "world"},
				},
				"two writes": {
					input:    []string{"one\ntwo", "\nthree\nfour\n"},
					expected: []string{"one", "two", "three", "four"},
				},
				"two buffered writes": {
					input:    []string{"hello ", "world", "\n"},
					expected: []string{"hello world"},
				},
			},
		},
	}
	for groupName, testCasesGroup := range testCases {
		t.Run(groupName, func(t *testing.T) {
			for name, testCase := range testCasesGroup.testCases {
				t.Run(name, func(t *testing.T) {
					reader := strings.NewReader(strings.Join(testCase.input, ""))
					var actual []string
					writer := splitwriter.New(splitwriter.HandlerFunc(func(token []byte) error {
						actual = append(actual, string(token))
						return nil
					})).Split(testCasesGroup.split)
					_, err := io.Copy(writer, reader)
					assert.NoError(t, err)
					assert.Equal(t, testCase.expected, actual)
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
	w := splitwriter.New(h)
	h.On("Handle", []byte("one")).Return(errors.New("handle err")).Once()
	h.On("Handle", []byte("one")).Return(nil).Once()
	bytesWritten, err := w.Write([]byte("one"))
	assert.NoError(t, err)
	assert.Equal(t, 3, bytesWritten)
	assert.Equal(t, 3, w.BufferLen())
	bytesWritten, err = w.Write([]byte("\n"))
	assert.Error(t, err)
	assert.Equal(t, `failed to handle token "one": handle err`, err.Error())
	assert.Equal(t, 0, bytesWritten)
	bytesWritten, err = w.Write([]byte("\n"))
	assert.NoError(t, err)
	assert.Equal(t, 1, bytesWritten)
}

func TestHandlerErrEmptyBuffer(t *testing.T) {
	h := new(mockHandler)
	w := splitwriter.New(h)
	h.On("Handle", []byte("one")).Return(errors.New("handle err"))
	bytesWritten, err := w.Write([]byte("one\n"))
	assert.Error(t, err)
	assert.Equal(t, 0, bytesWritten)
	assert.Equal(t, `failed to handle token "one": handle err`, err.Error())
}

func TestRecoverFromErrEmptyBuffer(t *testing.T) {
	h := new(mockHandler)
	w := splitwriter.New(h)
	h.On("Handle", []byte("one")).Return(errors.New("handle err"))
	h.On("Handle", []byte("two")).Return(nil)
	bytesWritten, err := w.Write([]byte("one\n"))
	assert.Error(t, err)
	assert.Equal(t, 0, bytesWritten)
	assert.Equal(t, `failed to handle token "one": handle err`, err.Error())
	bytesWritten, err = w.Write([]byte("two\n"))
	assert.NoError(t, err)
	assert.Equal(t, 4, bytesWritten)
}

func TestRecoverFromErrBuffered(t *testing.T) {
	h := new(mockHandler)
	w := splitwriter.New(h)
	h.On("Handle", []byte("buf one")).Return(errors.New("handle err"))
	h.On("Handle", []byte("buf two")).Return(nil)
	bytesWritten, err := w.Write([]byte("buf "))
	assert.NoError(t, err)
	assert.Equal(t, 4, bytesWritten)
	bytesWritten, err = w.Write([]byte("one\n"))
	assert.Error(t, err)
	assert.Equal(t, 0, bytesWritten)
	assert.Equal(t, `failed to handle token "buf one": handle err`, err.Error())
	bytesWritten, err = w.Write([]byte("two\n"))
	assert.NoError(t, err)
	assert.Equal(t, 4, bytesWritten)
}

func TestSplitErrEmptyBuffer(t *testing.T) {
	h := new(mockHandler)
	h.On("Handle").Panic("Handle must not be called in this test")
	w := splitwriter.New(h).Split(func(data []byte) (advance int, token []byte, err error) {
		return 0, nil, errors.New("split err")
	})
	bytesWritten, err := w.Write([]byte("hello"))
	assert.Error(t, err)
	assert.Equal(t, 0, bytesWritten)
	assert.Equal(t, "failed to split: split err", err.Error())
}

func TestSplitErrBuffered(t *testing.T) {
	h := new(mockHandler)
	h.On("Handle").Panic("Handle must not be called in this test")
	w := splitwriter.New(h).Split(func(data []byte) (advance int, token []byte, err error) {
		if string(data) == "one" {
			return 0, nil, nil
		}
		return 0, nil, errors.New("split err")
	})
	bytesWritten, err := w.Write([]byte("one"))
	assert.NoError(t, err)
	assert.Equal(t, 3, bytesWritten)
	bytesWritten, err = w.Write([]byte("two"))
	assert.Error(t, err)
	assert.Equal(t, 0, bytesWritten)
	assert.Equal(t, "failed to split: split err", err.Error())
}

func TestCallSplitAfterWrite(t *testing.T) {
	h := new(mockHandler)
	h.On("Handle").Panic("Handle must not be called in this test")
	w := splitwriter.New(h)
	bytesWritten, err := w.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, bytesWritten)
	defer func() {
		r := recover()
		assert.Equal(t, "Split called after Write", r)
	}()
	w.Split(splitwriter.ScanRunes)
}
