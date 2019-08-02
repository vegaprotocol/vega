package execution

import (
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"
)

const (
	topUpAmount = 1000000000000
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/collateral_mock.go -package mocks code.vegaprotocol.io/vega/internal/execution Collateral
type Collateral interface {
	CreateTraderAccount(partyID, marketID, asset string) (string, string)
	IncrementBalance(id string, amount int64) error
	GetAccountByID(id string) (*proto.Account, error)
	AddTraderToMarket(mkt, trader, asset string) error
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
	store      PartyStore
	parties    map[string]map[string]struct{}
}

func NewParty(
	log *logging.Logger, col Collateral, markets []proto.Market, store PartyStore,
) *Party {
	parties := map[string]map[string]struct{}{}

	for _, v := range markets {
		parties[v.Id] = map[string]struct{}{}
	}

	return &Party{
		log:        log,
		collateral: col,
		markets:    markets,
		store:      store,
		parties:    parties,
	}
}

func (p *Party) GetForMarket(mktID string) []string {
	parties := p.parties[mktID]
	out := make([]string, 0, len(parties))
	for k, _ := range parties {
		out = append(out, k)
	}
	return out
}

func (p *Party) addParty(ptyID, mktID string) {
	p.parties[mktID][ptyID] = struct{}{}
}

func (p *Party) NotifyTraderAccount(notif *proto.NotifyTraderAccount) error {
	alreadyTopUp := map[string]struct{}{}

	// ignore erros as they can only happen when the party already exists
	err := p.store.Post(&types.Party{Id: notif.TraderID})
	if err == nil {
		p.log.Info("New party created",
			logging.String("party-id", notif.TraderID))
	}

	for _, mkt := range p.markets {
		p.addParty(notif.TraderID, mkt.Id)
		asset, err := mkt.GetAsset()
		if err != nil {
			p.log.Error("unable to get market asset",
				logging.Error(err))
			return err
		}
		// create account
		_, generalID := p.collateral.CreateTraderAccount(notif.TraderID, mkt.Id, asset)

		if _, ok := alreadyTopUp[generalID]; !ok {
			alreadyTopUp[generalID] = struct{}{}
			// then credit the general account
			err = p.collateral.IncrementBalance(generalID, topUpAmount)
			if err != nil {
				p.log.Error("unable to topup trader account",
					logging.Error(err))
				return err
			}
		}

		acc, err := p.collateral.GetAccountByID(generalID)
		if err != nil {
			p.log.Error("unable to get trader account",
				logging.String("traderID", notif.TraderID),
				logging.String("asset", asset),
				logging.Error(err))
			return err
		}
		p.log.Info("trader account topup",
			logging.String("asset", asset),
			logging.String("traderID", notif.TraderID),
			logging.Int64("topup-amount", topUpAmount),
			logging.Int64("new-balance", acc.Balance))

		// now add the trader to the given market (move monies is margin account)
		err = p.collateral.AddTraderToMarket(mkt.Id, notif.TraderID, asset)
		if err != nil {
			p.log.Error("unable to add trader to market",
				logging.String("traderID", notif.TraderID),
				logging.String("asset", asset),
				logging.String("marketID", mkt.Id),
				logging.Error(err))
		}
	}

	return nil
}
