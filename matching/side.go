package matching

import (
	"sort"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	// ErrPriceNotFound signals that a price was not found on the book side
	ErrPriceNotFound = errors.New("price-volume pair not found")
	// ErrNoOrder signals that there's no orders on the book side.
	ErrNoOrder = errors.New("no orders in the book side")
)

type baseSide struct {
	log    *logging.Logger
	levels []*PriceLevel
}

func (s baseSide) BestPriceAndVolume(side types.Side) (uint64, uint64) {
	if len(s.levels) <= 0 {
		return 0, 0
	}
	return s.levels[len(s.levels)-1].price, s.levels[len(s.levels)-1].volume
}

func (s *baseSide) amendOrder(orderAmended *types.Order) error {
	priceLevelIndex := -1
	orderIndex := -1
	var oldOrder *types.Order

	for idx, priceLevel := range s.levels {
		if priceLevel.price == orderAmended.Price {
			priceLevelIndex = idx
			for j, order := range priceLevel.orders {
				if order.Id == orderAmended.Id {
					orderIndex = j
					oldOrder = order
					break
				}
			}
			break
		}
	}

	if oldOrder == nil || priceLevelIndex == -1 || orderIndex == -1 {
		return types.ErrOrderNotFound
	}

	if oldOrder.PartyID != orderAmended.PartyID {
		return types.ErrOrderAmendFailure
	}

	if oldOrder.Size < orderAmended.Size {
		return types.ErrOrderAmendFailure
	}

	if oldOrder.Reference != orderAmended.Reference {
		return types.ErrOrderAmendFailure
	}

	s.levels[priceLevelIndex].orders[orderIndex] = orderAmended
	return nil
}

// RemoveOrder will remove an order from the book
func (s *baseSide) removeOrderAtPriceLevelIndex(i int, o *types.Order) (*types.Order, error) {
	// we did not found the level
	// then the order do not exists in the price level
	if i >= len(s.levels) {
		return nil, types.ErrOrderNotFound
	}

	// orders are order by timestamp (CreatedAt)
	oidx := sort.Search(len(s.levels[i].orders), func(j int) bool {
		return s.levels[i].orders[j].CreatedAt >= o.CreatedAt
	})
	// we did not find the order
	if oidx >= len(s.levels[i].orders) {
		return nil, types.ErrOrderNotFound
	}

	// now we may have a few orders with the same timestamp
	// lets iterate over them in order to find the right one
	finaloidx := -1
	for oidx < len(s.levels[i].orders) && s.levels[i].orders[oidx].CreatedAt == o.CreatedAt {
		if s.levels[i].orders[oidx].Id == o.Id {
			finaloidx = oidx
			break
		}
		oidx++
	}

	var order *types.Order
	// remove the order from the
	if finaloidx != -1 {
		order = s.levels[i].orders[finaloidx]
		s.levels[i].removeOrder(finaloidx)
	}

	if len(s.levels[i].orders) <= 0 {
		s.levels = s.levels[:i+copy(s.levels[i:], s.levels[i+1:])]
	}

	return order, nil
}

func (s *baseSide) Print() {
	for _, priceLevel := range s.levels {
		if len(priceLevel.orders) > 0 {
			priceLevel.print(s.log)
		}
	}
}
