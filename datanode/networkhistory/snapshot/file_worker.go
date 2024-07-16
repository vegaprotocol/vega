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

package snapshot

import (
	"bytes"
	"fmt"
	"os"
	"sync"
)

type bufAndPath struct {
	buf            *bytes.Buffer
	isProgressFile bool
	path           string
}

type FileWorker struct {
	mu    sync.Mutex
	queue []*bufAndPath
}

func NewFileWorker() *FileWorker {
	return &FileWorker{
		queue: []*bufAndPath{},
	}
}

func (fw *FileWorker) Add(buf *bytes.Buffer, path string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	fw.queue = append(fw.queue, &bufAndPath{buf, false, path})
}

func (fw *FileWorker) AddLockFile(path string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	fw.queue = append(fw.queue, &bufAndPath{nil, true, path})
}

func (fw *FileWorker) peek() (bp *bufAndPath) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if len(fw.queue) <= 0 {
		return
	}

	bp, fw.queue = fw.queue[0], fw.queue[1:]

	return
}

func (fw *FileWorker) Empty() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	return len(fw.queue) <= 0
}

func (fw *FileWorker) Consume() error {
	bp := fw.peek()
	if bp == nil {
		return nil // nothing to do
	}

	if bp.isProgressFile {
		return fw.removeLockFile(bp.path)
	}

	return fw.writeSegment(bp)
}

func (fw *FileWorker) removeLockFile(path string) error {
	return os.Remove(path)
}

func (fw *FileWorker) writeSegment(bp *bufAndPath) error {
	file, err := os.Create(bp.path)
	if err != nil {
		return fmt.Errorf("failed to create file %s : %w", bp.path, err)
	}

	defer file.Close()

	_, err = bp.buf.WriteTo(file)
	if err != nil {
		return fmt.Errorf("couldn't writer to file %v : %w", bp.path, err)
	}

	return nil
}
