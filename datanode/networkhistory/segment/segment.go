package segment

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
)

// Base is the base struct for all segment types.
type Base struct {
	HeightFrom      int64
	HeightTo        int64
	DatabaseVersion int64
	ChainID         string
}

func (m Base) String() string {
	return fmt.Sprintf("{Network History Segment for Chain ID:%s Height From:%d Height To:%d}", m.ChainID, m.HeightFrom, m.HeightTo)
}

func (m Base) ZipFileName() string {
	return fmt.Sprintf("%s-%d-%d-%d.zip", m.ChainID, m.DatabaseVersion, m.HeightFrom, m.HeightTo)
}

func NewFromZipFileName(fileName string) (Base, error) {
	re, err := regexp.Compile(`(.*)-(\d+)-(\d+)-(\d+).zip`)
	if err != nil {
		return Base{}, fmt.Errorf("failed to compile reg exp:%w", err)
	}

	matches := re.FindStringSubmatch(fileName)
	if len(matches) != 5 {
		return Base{}, fmt.Errorf("failed to find matches in zip file name:%s", fileName)
	}

	dbVersion, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return Base{}, err
	}

	heightFrom, err := strconv.ParseInt(matches[3], 10, 64)
	if err != nil {
		return Base{}, err
	}

	heightTo, err := strconv.ParseInt(matches[4], 10, 64)
	if err != nil {
		return Base{}, err
	}

	result := Base{
		ChainID:         matches[1],
		HeightFrom:      heightFrom,
		HeightTo:        heightTo,
		DatabaseVersion: dbVersion,
	}
	return result, nil
}

func (m Base) GetFromHeight() int64 {
	return m.HeightFrom
}

func (m Base) GetToHeight() int64 {
	return m.HeightTo
}

func (m Base) GetDatabaseVersion() int64 {
	return m.DatabaseVersion
}

func (m Base) GetChainId() string {
	return m.ChainID
}

// MetaData adds a PreviousHistorySegmentID, and is the struct that gets serialized into
// the JSON metadata when a segment is added to the store.
type MetaData struct {
	Base
	PreviousHistorySegmentID string
}

func (m MetaData) GetPreviousHistorySegmentId() string {
	return m.PreviousHistorySegmentID
}

// Full is a segment that has been added to the store and has been assigned a segment ID.
type Full struct {
	MetaData
	HistorySegmentID string
}

func (f Full) GetHistorySegmentId() string {
	return f.HistorySegmentID
}

// Staged is a segment which has been added to the store and then fetched back again.
type Staged struct {
	Full
	Directory string
}

func (s Staged) ZipFilePath() string {
	return path.Join(s.Directory, s.ZipFileName())
}

// Unpublished is a segment that has just been dumped from the database into a zip file but
// hasn't yet been added to the store so doesn't have any extra metadata.
type Unpublished struct {
	Base
	Directory string
}

func (s Unpublished) ZipFilePath() string {
	return path.Join(s.Directory, s.ZipFileName())
}

func (s Unpublished) InProgressFilePath() string {
	return path.Join(s.Directory, fmt.Sprintf("%s-%d.snapshotinprogress", s.ChainID, s.HeightTo))
}

// Segments is just a list of segments with a bit of syntactic sugar for getting contiguous
// histories of segments in a nice way.
type Segments[T blockSpanner] []T

func (s Segments[T]) MostRecentContiguousHistory() (ContiguousHistory[T], error) {
	all := s.AllContigousHistories()
	if len(all) == 0 {
		return ContiguousHistory[T]{}, fmt.Errorf("no segments")
	}
	return all[len(all)-1], nil
}

func (s Segments[T]) AllContigousHistories() []ContiguousHistory[T] {
	sort.Slice(s, func(i, j int) bool {
		return s[i].GetFromHeight() < s[j].GetFromHeight()
	})

	var histories []ContiguousHistory[T]
	for _, segment := range s {
		added := false
		for i := range histories {
			if histories[i].Add(segment) {
				added = true
				break
			}
		}
		if !added {
			ch := ContiguousHistory[T]{}
			ch.Add(segment)
			histories = append(histories, ch)
		}
	}
	return histories
}

func (s Segments[T]) ContiguousHistoryInRange(fromHeight int64, toHeight int64) (ContiguousHistory[T], error) {
	c := s.AllContigousHistories()
	for _, ch := range c {
		if ch.HeightFrom <= fromHeight && ch.HeightTo >= toHeight {
			fromSegmentFound := false
			toSegmentFound := false
			for _, segment := range ch.Segments {
				if segment.GetFromHeight() == fromHeight {
					fromSegmentFound = true
				}

				if segment.GetToHeight() == toHeight {
					toSegmentFound = true
				}
			}

			if fromSegmentFound && toSegmentFound {
				return ch.Slice(fromHeight, toHeight), nil
			}
		}
	}

	return ContiguousHistory[T]{}, fmt.Errorf("no contiguous segment range found with height %d to %d", fromHeight, toHeight)
}
