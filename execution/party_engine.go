package execution

import (
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

var ErrPartyDoesNotExist = errors.New("party does not exist in party engine")
var ErrNotifyPartyIdMissing = errors.New("notify party id is missing")
var ErrInvalidPartyId = errors.New("party id is not valid")

// Collateral ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/collateral_mock.go -package mocks code.vegaprotocol.io/vega/execution Collateral
type Collateral interface {
	CreatePartyGeneralAccount(partyID, asset string) string
	IncrementBalance(id string, amount uint64) error
	DecrementBalance(id string, amount uint64) error
	GetAccountByID(id string) (*types.Account, error)
	GetPartyGeneralAccount(partyID, asset string) (*types.Account, error)
	GetPartyTokenAccount(string) (*types.Account, error)
}

// PartyEngine holds the list of parties in the system
type PartyEngine struct {
	log        *logging.Logger
	collateral Collateral
	buf        PartyBuf

	markets map[string]types.Market
	Parties []string
}

// NewPartyEngine instantiates a new party engine
func NewPartyEngine(log *logging.Logger, col Collateral, markets []types.Market, partyBuf PartyBuf) *PartyEngine {
	allMarkets := map[string]types.Market{}
	for _, m := range markets {
		allMarkets[m.Id] = m
	}
	return &PartyEngine{
		log:        log,
		collateral: col,
		markets:    allMarkets,
		buf:        partyBuf,
	}
}

func (p *PartyEngine) partyExists(partyID string) bool {
	idx := sort.SearchStrings(p.Parties, partyID)
	return idx >= 0 && idx < len(p.Parties) && p.Parties[idx] == partyID
}

// Add stores party in memory, allocates necessary accounts and notifies the buffer
func (p *PartyEngine) Add(partyID string) (bool, error) {
	if len(partyID) <= 0 {
		return false, ErrInvalidPartyId
	}
	if !p.partyExists(partyID) {
		if _, err := p.makeGeneralAccounts(partyID); err != nil {
			return false, err
		}
		p.buf.Add(types.Party{Id: partyID})
		p.Parties = append(p.Parties, partyID)
		sort.Strings(p.Parties)
		return true, nil // added
	}
	return false, nil // could not add, party already exists
}

// Find looks up existing party by ID
func (p *PartyEngine) GetByID(partyID string) (*types.Party, error) {
	if p.partyExists(partyID) {
		return &types.Party{Id: partyID}, nil
	}
	return nil, ErrPartyDoesNotExist
}

func (p *PartyEngine) addMarket(market types.Market) {
	if _, found := p.markets[market.Id]; found {
		p.log.Debug("overwriting market in party engine", logging.Market(market))
		// it will be OK to overwrite market on update market proposal enactment
	}
	p.markets[market.Id] = market
}

// makeGeneralAccounts creates general accounts on every market for the given party id
func (p *PartyEngine) makeGeneralAccounts(partyID string) (int, error) {
	added := map[string]struct{}{}
	for _, market := range p.markets {
		asset, err := market.GetAsset()
		if err != nil {
			p.log.Error("unable to get market asset", logging.Error(err))
			return 0, err
		}
		general := p.collateral.CreatePartyGeneralAccount(partyID, asset)
		if _, exists := added[general]; !exists {
			if p.log.GetLevel() == logging.DebugLevel {
				p.log.Debug("created general account",
					logging.String("market-id", market.Id),
					logging.String("asset", asset),
					logging.String("party-id", partyID))
				added[general] = struct{}{}
			}
		}
	}
	return len(added), nil
}

func (p *PartyEngine) creditGeneralAccounts(partyID string, amount uint64) error {

	for _, market := range p.markets {
		asset, err := market.GetAsset()
		if err != nil {
			p.log.Error("unable to get market asset", logging.Error(err))
			return err
		}
		account, err := p.collateral.GetPartyGeneralAccount(partyID, asset)
		if err != nil {
			return err
		}
		if err := p.collateral.IncrementBalance(account.Id, amount); err != nil {
			p.log.Errorf("unable to top-up general account %s: %s", logging.Error(err))
			return err
		}
	}
	return nil
}

func (p *PartyEngine) creditTokenAccount(partyID string, amount uint64) error {
	account, err := p.collateral.GetPartyTokenAccount(partyID)
	if err != nil {
		return err
	}
	if err := p.collateral.IncrementBalance(account.Id, amount); err != nil {
		p.log.Error("unable to top-up token account", logging.Error(err))
		return err
	}
	return nil
}

// DefaultCredit is arbitrary selected value to credit newly created accounts by default
const DefaultCredit = 1000000000 // 10000.00000

// NotifyTraderAccount will create a new party in the system
// and top-up its general account with the default amount
func (p *PartyEngine) NotifyTraderAccount(notify *types.NotifyTraderAccount) error {

	if notify == nil {
		return ErrNotifyPartyIdMissing
	}
	if _, err := p.Add(notify.TraderID); err != nil {
		return err
	}
	credit := notify.Amount
	if credit == 0 {
		credit = DefaultCredit
	}
	if err := p.creditGeneralAccounts(notify.TraderID, credit); err != nil {
		return nil
	}
	if err := p.creditTokenAccount(notify.TraderID, credit); err != nil {
		return err
	}
	return nil
}
