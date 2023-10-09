// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
