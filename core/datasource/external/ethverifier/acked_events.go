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

package ethverifier

import (
	"github.com/emirpasic/gods/sets/treeset"
	"github.com/emirpasic/gods/utils"
)

const oneHour = 3600 // seconds

type ackedEvtBucket struct {
	ts     int64
	endTs  int64
	hashes map[string]struct{}
}

func ackedEvtBucketComparator(a, b interface{}) int {
	bucket1 := a.(*ackedEvtBucket)
	bucket2 := b.(*ackedEvtBucket)
	return utils.Int64Comparator(bucket1.ts, bucket2.ts)
}

type ackedEvents struct {
	timeService TimeService
	events      *treeset.Set // we only care about the key
}

func (a *ackedEvents) AddAt(ts int64, hashes ...string) {
	_, value := a.events.Find(func(i int, value interface{}) bool {
		bucket := value.(*ackedEvtBucket)
		return bucket.ts <= ts && bucket.endTs >= ts
	})

	if value != nil {
		bucket := value.(*ackedEvtBucket)
		for _, newHash := range hashes {
			bucket.hashes[newHash] = struct{}{}
		}

		return
	}

	hashesM := map[string]struct{}{}
	for _, v := range hashes {
		hashesM[v] = struct{}{}
	}

	a.events.Add(&ackedEvtBucket{ts: ts, endTs: ts + oneHour, hashes: hashesM})
}

// RestoreExactAt - is to be used when loading a snapshot only
// this prevent restoring in different buckets, which could happen
// when events are received out of sync (e.g: timestamps 100 before 90) which could make gap between buckets.
func (a *ackedEvents) RestoreExactAt(ts int64, hashes ...string) {
	hashesM := map[string]struct{}{}
	for _, v := range hashes {
		hashesM[v] = struct{}{}
	}

	a.events.Add(&ackedEvtBucket{ts: ts, endTs: ts + oneHour, hashes: hashesM})
}

func (a *ackedEvents) Add(hash string) {
	a.AddAt(a.timeService.GetTimeNow().Unix(), hash)
}

func (a *ackedEvents) Contains(hash string) bool {
	_, value := a.events.Find(func(index int, value interface{}) bool {
		bucket := value.(*ackedEvtBucket)
		_, ok := bucket.hashes[hash]
		return ok
	})

	return value != nil
}

func (a *ackedEvents) RemoveBefore(ts int64) {
	set := a.events.Select(func(index int, value interface{}) bool {
		bucket := value.(*ackedEvtBucket)
		return bucket.ts <= ts
	})

	a.events.Remove(set.Values()...)
}

func (a *ackedEvents) Size() int {
	return a.events.Size()
}
