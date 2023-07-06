package entities

import (
	"encoding/json"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/protos/vega"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type SuccessorMarket struct {
	Market    Market
	Proposals []*Proposal
}

func (s SuccessorMarket) ToProtoEdge(...any) (*v2.SuccessorMarketEdge, error) {
	props := make([]*vega.GovernanceData, len(s.Proposals))

	for i, p := range s.Proposals {
		props[i] = &vega.GovernanceData{
			Proposal: p.ToProto(),
		}
	}

	e := &v2.SuccessorMarketEdge{
		Node: &v2.SuccessorMarket{
			Market:    s.Market.ToProto(),
			Proposals: props,
		},
		Cursor: s.Market.Cursor().Encode(),
	}

	return e, nil
}

func (s SuccessorMarket) Cursor() *Cursor {
	c := SuccessorMarketCursor{
		VegaTime: s.Market.VegaTime,
	}
	return NewCursor(c.String())
}

type SuccessorMarketCursor struct {
	VegaTime time.Time `json:"vegaTime"`
}

func (mc SuccessorMarketCursor) String() string {
	bs, err := json.Marshal(mc)
	if err != nil {
		panic(fmt.Errorf("could not marshal market cursor: %w", err))
	}
	return string(bs)
}

func (mc *SuccessorMarketCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), mc)
}
