package matching

import (
	"sort"

	types "code.vegaprotocol.io/vega/proto"
)

type sortedorders []types.Order

func newSortedOrders(initialCap int) sortedorders {
	return sortedorders(make([]types.Order, 0, initialCap))
}

func (so sortedorders) insert(ord types.Order) sortedorders {
	s := []types.Order(so)
	if len(s) <= 0 {
		s = append(s, ord)
		return sortedorders(s)
	}

	// first find the position where this should be inserted
	i := sort.Search(len(s), func(i int) bool { return s[i].ExpiresAt >= ord.ExpiresAt })

	// append new elem first to make sure we have enough place
	// this would reallocate sufficiently then
	// no risk of this being a empty order, as it's overwritten just next with
	// the slice insertttt
	s = append(s, types.Order{})
	copy(s[i+1:], s[i:])
	s[i] = ord

	return sortedorders(s)
}

func (so sortedorders) removeExpired(expirationTs int64) ([]types.Order, sortedorders) {
	if len(so) <= 0 {
		return []types.Order{}, so
	}
	s := []types.Order(so)
	// find the index of the last ts of expired order
	i := sort.Search(len(s), func(i int) bool { return s[i].ExpiresAt >= expirationTs })

	// we need to iterate a few because the previous search have this behavior
	// in the case we have multiple orders expiring at the same time
	// [1, 2, 3, 3, 3, 4, 4, 5, 6]
	// ~~~~~~~^
	// the search algo will stop at the first 3, with and exirationTs of 3,
	// so we just need to iterat a little
	// altho this may be unlikely to happen often as we are precise at the nanosec
	for ; i < len(s) && expirationTs == s[i].ExpiresAt; i += 1 {
	}
	expired := s[:i]
	pending := s[i:]
	return expired, sortedorders(pending)
}
