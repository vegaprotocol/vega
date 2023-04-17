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
