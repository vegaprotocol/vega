package execution

import (
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/proto"
)

const (
	topUpAmout   = 10000000
	defaultAsset = "ETH"
)

type Collateral interface {
	CreateTraderAccount(partyID, marketID, asset string) error
	Credit(partyID, asset string, amount int64)
}

type accountKey struct {
	marketID string
	partyID  string
	asset    string
}

type Party struct {
	log        *logging.Logger
	collateral Collateral
	markets    []proto.Market
}

func NewParty(log *logging.Logger, col Collateral, markets []proto.Market) *Party {
	return &Party{
		log:        log,
		collateral: col,
		markets:    markets,
	}
}

func (p *Party) NotifyTraderAccount(notif *proto.NotifyTraderAccount) error {
	// first creat general account
	err := p.collateral.CreateTraderAccount(notif.TraderID, "", defaultAsset)
	if err != nil {
		return err
	}
	// then credit the general account
	p.collateral.Credit(notif.TraderID, defaultAsset, topUpAmout)

	// now the markets specific accounts
	for _, mkt := range p.markets {
		asset, err := mkt.GetAsset()
		if err != nil {
			p.log.Error("unable to get market asset",
				logging.Error(err))
			return err
		}
		err = p.collateral.CreateTraderAccount(notif.TraderID, mkt.Id, asset)
		if err != nil {
			p.log.Error("unable to create margin account for party",
				logging.String("partyID", notif.TraderID),
				logging.String("market", mkt.Name),
				logging.String("asset", asset))
			return err
		}
	}

	return nil
}
