package execution

import (
	"errors"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"
)

const (
	topUpAmount = 1000000000000
)

var (
	ErrPartyDoNotExist = errors.New("party do not exist")
)

// Collateral ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/collateral_mock.go -package mocks code.vegaprotocol.io/vega/execution Collateral
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

// Party holds the list of parties in the system
type Party struct {
	log           *logging.Logger
	collateral    Collateral
	markets       []proto.Market
	partyBuf      PartyBuf
	partyByMarket map[string]map[string]struct{}
}

// NewParty instanciate a new party
func NewParty(
	log *logging.Logger, col Collateral, markets []proto.Market, partyBuf PartyBuf,
) *Party {
	partyByMarket := map[string]map[string]struct{}{}

	for _, v := range markets {
		partyByMarket[v.Id] = map[string]struct{}{}
	}

	return &Party{
		log:           log,
		collateral:    col,
		markets:       markets,
		partyBuf:      partyBuf,
		partyByMarket: partyByMarket,
	}
}

// GetForMarket returns the list of all the parties in a given market
func (p *Party) GetForMarket(mktID string) []string {
	parties := p.partyByMarket[mktID]
	out := make([]string, 0, len(parties))
	for k := range parties {
		out = append(out, k)
	}
	return out
}

func (p *Party) GetByMarketAndID(marketID, partyID string) (*types.Party, error) {
	if _, ok := p.partyByMarket[marketID][partyID]; ok {
		return &types.Party{Id: partyID}, nil
	}

	return nil, ErrPartyDoNotExist
}

func (p *Party) addParty(ptyID, mktID string) {
	p.partyByMarket[mktID][ptyID] = struct{}{}
}

// NotifyTraderAccountWithTopUpAmount will create a new party in the system
// and topup it general account with the given amount
func (p *Party) NotifyTraderAccountWithTopUpAmount(
	notif *proto.NotifyTraderAccount, amount int64) error {
	return p.notifyTraderAccount(notif, amount)
}

// NotifyTraderAccount will create a new party in the system
// and topup it general account with the default amount
func (p *Party) NotifyTraderAccount(notif *proto.NotifyTraderAccount) error {
	if notif.Amount == 0 {
		return p.notifyTraderAccount(notif, topUpAmount)
	}
	return p.notifyTraderAccount(notif, int64(notif.Amount))
}

func (p *Party) notifyTraderAccount(notif *proto.NotifyTraderAccount, amount int64) error {
	alreadyTopUp := map[string]struct{}{}

	// ignore erros as they can only happen when the party already exists
	p.partyBuf.Add(types.Party{Id: notif.TraderID})
	// err := p.store.Post(&types.Party{Id: notif.TraderID})
	// if err == nil {
	// 	p.log.Info("New party created",
	// 		logging.String("party-id", notif.TraderID))
	// }

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
			err = p.collateral.IncrementBalance(generalID, amount)
			if err != nil {
				p.log.Error("unable to topup trader account",
					logging.Error(err))
				return err
			}
			acc, err := p.collateral.GetAccountByID(generalID)
			if err != nil {
				p.log.Error("unable to get trader account",
					logging.String("party-id", notif.TraderID),
					logging.String("asset", asset),
					logging.Error(err))
				return err
			}
			if p.log.GetLevel() == logging.DebugLevel {
				p.log.Debug("party account top-up",
					logging.String("asset", asset),
					logging.String("party-id", notif.TraderID),
					logging.Int64("top-up-amount", amount),
					logging.Int64("new-balance", acc.Balance))
			}
		}

		// now add the trader to the given market (move monies is margin account)
		err = p.collateral.AddTraderToMarket(mkt.Id, notif.TraderID, asset)
		if err != nil {
			p.log.Error("unable to add party to market",
				logging.String("party-id", notif.TraderID),
				logging.String("asset", asset),
				logging.String("market-id", mkt.Id),
				logging.Error(err))
		}
	}

	return nil
}
