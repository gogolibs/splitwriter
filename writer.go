package splitwriter

import (
	"bufio"
	"bytes"
	"github.com/pkg/errors"
)

// Handler is an interface that must be implemented by client handlers that wish to
// handle tokens passed by splitwriter.Writer.
type Handler interface {
	Handle(token []byte) error
}

// HandlerFunc is an alternative way to specify a handler for splitwriter.Writer.
type HandlerFunc func(token []byte) error

func (f HandlerFunc) Handle(token []byte) error {
	return f(token)
}

// SplitFunc is the signature of the split function used to tokenize the
// input. It is a wrapper around bufio.SplitFunc and the only difference
// is that it does not receive atEOF flag. This flag must be always set to false
// as an io.Writer has no way of determining that it is at the end of the input.
type SplitFunc func(data []byte) (advance int, token []byte, err error)

// New returns a new splitwriter.Writer that will pass any tokens that were encountered
// when writing to it to splitwriter.Handler.
func New(handler Handler) *Writer {
	return &Writer{
		handler:     handler,
		writeCalled: false,
		split:       ScanLines,
		buffer:      new(bytes.Buffer),
	}
}

// Writer implements io.Writer and hands over any tokens found via splitwriter.SplitFunc
// to the handler.
type Writer struct {
	handler     Handler
	writeCalled bool          // Write has been called; buffer is in use.
	split       SplitFunc     // A function to split the tokens.
	buffer      *bytes.Buffer // A buffer to hold incomplete tokens.
}

// Write handles data by splitting it into tokens and hading them over to handler.Handle.
func (w *Writer) Write(data []byte) (int, error) {
	w.writeCalled = true
	initialBufferLen := w.BufferLen()
	w.buffer.Write(data)
	dataRemainder := w.buffer.Bytes()
	bytesWritten := 0
	if initialBufferLen > 0 {
		advance, token, err := w.split(dataRemainder)
		if err != nil {
			w.buffer.Truncate(initialBufferLen)
			return 0, errors.Wrap(err, "failed to split")
		}
		if advance == 0 {
			return len(data), nil
		}
		err = w.handler.Handle(token)
		if err != nil {
			w.buffer.Truncate(initialBufferLen)
			return 0, errors.Wrapf(err, `failed to handle token "%s"`, string(token))
		}
		bytesWritten += advance - initialBufferLen
		dataRemainder = dataRemainder[advance:]
	}
	w.buffer.Reset()
	for {
		if len(dataRemainder) == 0 {
			break
		}
		advance, token, err := w.split(dataRemainder)
		if err != nil {
			return bytesWritten, errors.Wrap(err, "failed to split")
		}
		if advance == 0 {
			break
		}
		err = w.handler.Handle(token)
		if err != nil {
			return bytesWritten, errors.Wrapf(err, `failed to handle token "%s"`, string(token))
		}
		dataRemainder = dataRemainder[advance:]
		bytesWritten += advance
	}
	if len(dataRemainder) > 0 {
		w.buffer.Write(dataRemainder)
		bytesWritten += len(dataRemainder)
	}
	return bytesWritten, nil
}

// Split sets the split function for the Writer.
// The default split function is ScanLines.
//
// Split panics if it is called after writing has started.
func (w *Writer) Split(split SplitFunc) *Writer {
	if w.writeCalled {
		panic("Split called after Write")
	}
	w.split = split
	return w
}

// BufferLen returns the length of the buffered data (missing or incomplete token).
func (w *Writer) BufferLen() int {
	return w.buffer.Len()
}

// WrapBufioSplitFunc is used to wrap bufio.SplitFunc, excluding atEOF argument by setting
// it to be always equal to false unconditionally.
func WrapBufioSplitFunc(bufioSplitFunc bufio.SplitFunc) SplitFunc {
	return func(data []byte) (advance int, token []byte, err error) {
		return bufioSplitFunc(data, false)
	}
}

// ScanBytes is wrapped version of bufio.ScanBytes
var ScanBytes = WrapBufioSplitFunc(bufio.ScanBytes)

// ScanRunes is wrapped version of bufio.ScanRunes
var ScanRunes = WrapBufioSplitFunc(bufio.ScanRunes)

// ScanLines is wrapped version of bufio.ScanLines
var ScanLines = WrapBufioSplitFunc(bufio.ScanLines)

// ScanWords is wrapped version of bufio.ScanWords
var ScanWords = WrapBufioSplitFunc(bufio.ScanWords)
