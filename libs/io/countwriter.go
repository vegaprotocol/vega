// Copyright (C) 2023  Gobalsky Labs Limited
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
