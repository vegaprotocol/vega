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

package evtforward

import (
	"github.com/emirpasic/gods/sets/treeset"
	"github.com/emirpasic/gods/utils"
)

type ackedEvtBucket struct {
	ts     int64
	hashes []string
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
		return bucket.ts == ts
	})

	if value != nil {
		bucket := value.(*ackedEvtBucket)
		for _, newHash := range hashes {
			found := false
			for _, v := range bucket.hashes {
				// hash already exists
				if v == newHash {
					found = true
					break
				}
			}

			if !found {
				bucket.hashes = append(bucket.hashes, newHash)
			}
		}

		return
	}

	a.events.Add(&ackedEvtBucket{ts: ts, hashes: append([]string{}, hashes...)})
}

func (a *ackedEvents) Add(hash string) {
	a.AddAt(a.timeService.GetTimeNow().Unix(), hash)
}

func (a *ackedEvents) Contains(hash string) bool {
	_, value := a.events.Find(func(index int, value interface{}) bool {
		bucket := value.(*ackedEvtBucket)
		for _, v := range bucket.hashes {
			if hash == v {
				return true
			}
		}

		return false
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
