package io

import "io"

// CountWriter is an io.Writer that keeps track of the number of bytes written to it.
type CountWriter struct {
	count  int64
	writer io.Writer
}

func NewCountWriter(w io.Writer) *CountWriter {
	return &CountWriter{
		writer: w,
	}
}

func (w *CountWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.count += int64(n)
	return n, err
}

// Count returns the total number of bytes written to the writer.
func (w *CountWriter) Count() int64 {
	return w.count
}
