package api

import (
	"bytes"
	"fmt"

	"google.golang.org/genproto/googleapis/api/httpbody"
)

type httpBodyWriter struct {
	chunkSize   int
	contentType string
	buf         *bytes.Buffer
	stream      tradingDataServiceExportServer
}

type tradingDataServiceExportServer interface {
	Send(*httpbody.HttpBody) error
}

func (w *httpBodyWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	w.buf.Write(p)

	if w.buf.Len() >= w.chunkSize {
		if err := w.sendChunk(); err != nil {
			return 0, err
		}
	}

	return n, nil
}

func (w *httpBodyWriter) sendChunk() error {
	msg := &httpbody.HttpBody{
		ContentType: w.contentType,
		Data:        w.buf.Bytes(),
	}

	if err := w.stream.Send(msg); err != nil {
		return fmt.Errorf("error sending chunk: %w", err)
	}

	w.buf.Reset()
	return nil
}

func (w *httpBodyWriter) Close() error {
	if w.buf.Len() > 0 {
		return w.sendChunk()
	}
	return nil
}
