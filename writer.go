package splitwriter

import (
	"bufio"
	"github.com/pkg/errors"
	"io"
)

// SplitFunc is the signature of the split function used to tokenize the
// input. It is a wrapper around bufio.SplitFunc and the only difference
// is that it does not receive atEOF flag.
type SplitFunc func(data []byte) (advance int, token []byte, err error)

// New returns a new Writer.
// The split function defaults to ScanLines.
func New(writer io.Writer) *Writer {
	return &Writer{
		writer:      writer,
		split:       ScanLines,
		writeCalled: false,
		leftover:    make([]byte, 0),
	}
}

// Writer is used to tokenize data supplied into Write method
// before passing it to the underlying writer. Buffers data as necessary.
// Can be viewed as an alternative of bufio.Scanner for writers.
type Writer struct {
	writer      io.Writer // The writer provided by the client.
	split       SplitFunc // The function to split the tokens.
	writeCalled bool      // Write has been called; buffer is in use.
	leftover    []byte    // Missing or incomplete token data from previous writes.
}

func zeroIfLessThanZero(bytesWritten int) int {
	if bytesWritten < 0 {
		return 0
	}
	return bytesWritten
}

// Write tokenizes data and passes it to an underlying writer by calling its Write method with data holding
// every token, one at a time. As a consequence, every Write call may result in 0..n calls of the writer.Write
// depending on how many tokens has been scanned.
func (w *Writer) Write(data []byte) (int, error) {
	w.writeCalled = true
	bytesWritten := -len(w.leftover)
	dataRemainder := make([]byte, len(w.leftover)+len(data))
	for index, b := range w.leftover {
		dataRemainder[index] = b
	}
	for index, b := range data {
		dataRemainder[len(w.leftover)+index] = b
	}
	for {
		if len(dataRemainder) == 0 {
			break
		}
		advance, token, err := w.split(dataRemainder)
		if err != nil {
			return zeroIfLessThanZero(bytesWritten), errors.Wrap(err, "failed to split")
		}
		// If advance is zero it means no token has been found.
		// Need to wait for more writes to supply the token.
		if advance == 0 {
			break
		}
		tokenBytesWritten, err := w.writer.Write(token)
		if err != nil {
			return zeroIfLessThanZero(bytesWritten), errors.Wrap(err, "failed to write")
		}
		if tokenBytesWritten != len(token) {
			var description string
			if tokenBytesWritten > len(token) {
				description = "long"
			} else {
				description = "short"
			}
			return zeroIfLessThanZero(bytesWritten), errors.Errorf(
				"%s write: have %d want %d bytes",
				description,
				tokenBytesWritten,
				len(token),
			)
		}
		dataRemainder = dataRemainder[advance:]
		bytesWritten += advance
	}
	w.leftover = make([]byte, len(dataRemainder))
	for index, b := range dataRemainder {
		w.leftover[index] = b
	}
	bytesWritten += len(dataRemainder)
	return bytesWritten, nil
}

// Split sets the split function for the Writer.
// The default split function is ScanLines.
//
// Split panics if it is called after writing has started.
func (w *Writer) Split(split SplitFunc) {
	if w.writeCalled {
		panic("Split called after Write")
	}
	w.split = split
}

// BufferLen returns the length of the buffered data (missing or incomplete token).
func (w *Writer) BufferLen() int {
	return len(w.leftover)
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
