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

package segment

type blockSpanner interface {
	GetFromHeight() int64
	GetToHeight() int64
}

// ContiguousHistory is a list of ordered contiguous segments.
type ContiguousHistory[T blockSpanner] struct {
	HeightFrom int64
	HeightTo   int64
	Segments   []T
}

// NewChunkFromSegment returns a chunk containing a single segment.
func NewChunkFromSegment[T blockSpanner](segment T) ContiguousHistory[T] {
	return ContiguousHistory[T]{
		HeightFrom: segment.GetFromHeight(),
		HeightTo:   segment.GetToHeight(),
		Segments:   []T{segment},
	}
}

// Add attempts to insert new segment to the chunk, either at the beginning or at the end.
// It returns true if the segment was added, false if the new segment doesn't lead or follow our current range.
func (c *ContiguousHistory[T]) Add(new T) bool {
	if len(c.Segments) == 0 {
		c.Segments = []T{new}
		c.HeightFrom = new.GetFromHeight()
		c.HeightTo = new.GetToHeight()
		return true
	}

	if new.GetToHeight() == c.HeightFrom-1 {
		c.Segments = append([]T{new}, c.Segments...)
		c.HeightFrom = new.GetFromHeight()
		return true
	}

	if new.GetFromHeight() == c.HeightTo+1 {
		c.Segments = append(c.Segments, new)
		c.HeightTo = new.GetToHeight()
		return true
	}

	return false
}

// Slice returns a new chunk containing the segments which partially or fully fall into the specified range.
func (c ContiguousHistory[T]) Slice(from int64, to int64) ContiguousHistory[T] {
	var new ContiguousHistory[T]

	for _, segment := range c.Segments {
		if segment.GetToHeight() < from {
			continue
		}
		if segment.GetFromHeight() > to {
			continue
		}

		new.Add(segment)
	}
	return new
}
