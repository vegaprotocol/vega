package subscribers

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type Store interface {
	SaveMarginLevelsBatch(batch []types.MarginLevels)
}

type MarginLevelSub struct {
	*Base
	store Store
	buf   map[string]map[string]types.MarginLevels
}

func NewMarginLevelSub(ctx context.Context, store Store) *MarginLevelSub {
	m := MarginLevelSub{
		Base:  newBase(ctx, 10),
		store: store,
		buf:   map[string]map[string]types.MarginLevels{},
	}
	return &m
}
