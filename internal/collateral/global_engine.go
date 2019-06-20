package collateral

import "code.vegaprotocol.io/vega/internal/logging"

type AccountBuffer interface {
	Add(owner, marketID, asset string, balance int64)
}

type accountKey struct {
	marketID, partyID, asset string
}

type GlobalCollateral struct {
	log  *logging.Logger
	accs map[accountKey]int64
	buf  AccountBuffer
}

func NewGlobalCollateral(log *logging.Logger, buf AccountBuffer) *GlobalCollateral {
	return &GlobalCollateral{
		log:  log,
		accs: map[accountKey]int64{},
		buf:  buf,
	}
}

func (c *GlobalCollateral) CreateTraderAccount(partyID, marketID, asset string) error {
	key := accountKey{marketID, partyID, asset}
	_, ok := c.accs[key]
	if !ok {
		c.accs[key] = 0
		c.buf.Add(partyID, marketID, asset, 0)
	}
	return nil
}

func (c *GlobalCollateral) Credit(partyID, asset string, amount int64) int64 {
	key := accountKey{"", partyID, asset}
	balance, ok := c.accs[key]
	if !ok {
		c.CreateTraderAccount(partyID, "", asset)
		balance = 0
	}
	newBalance := balance + amount
	c.accs[key] = newBalance
	c.buf.Add(partyID, "", asset, 0)
	return newBalance
}
