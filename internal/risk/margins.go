package risk

import (
	"errors"

	"code.vegaprotocol.io/vega/internal/events"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrNoFactorForAsset = errors.New("no risk factors found for given asset")
)

// long, short, base, max, optimal are all for volume 1
type marginAmount struct {
	long, short, base, step float64
}

func (e *Engine) getMargins(asset string) (*marginAmount, error) {
	factor, ok := e.factors.RiskFactors[asset]
	if !ok {
		return nil, ErrNoFactorForAsset
	}
	m := marginAmount{
		long:  factor.Long,
		short: factor.Short,
	}
	if m.long == m.short {
		m.step = m.long / 10
		m.base = m.long // we need a base value to calculate optimal/over
		return &m, nil
	}
	// get the abs value of the delta in other cases
	// for optimal margin value, use the highest of the 2 values
	if m.long > m.short {
		m.step = m.long - m.short
		m.base = m.long
	} else {
		m.step = m.short - m.long
		m.base = m.short
	}
	return &m, nil
}

// get amount of money to move to match the risk assessment
func (m marginAmount) getChange(evt events.Margin, markPrice uint64) *marginChange {
	volume := evt.Size()
	absVol := volume
	if absVol > 0 {
		absVol *= -1
	}
	notional := int64(markPrice) * absVol
	var required uint64
	if volume < 0 {
		// trader is short
		required = uint64(float64(notional) * m.short)
	} else {
		required = uint64(float64(notional) * m.long)
	}
	balance := evt.MarginBalance()
	// we've got enough margin, no further action required
	if balance == required {
		return nil
	}
	// *if* an amounts needs to be moved, this is the amount...
	transfer := types.Transfer{
		Owner: evt.Party(),
		Size:  1,
		Amount: &types.FinancialAmount{
			Asset:  evt.Asset(),
			Amount: int64(float64(notional)*(m.base+m.step)) - int64(balance),
		},
		// Type: types.TransferType_MARGIN_{LOW,HIGH}, <-- these types need to be added to the proto
	}
	// step := uint64(float64(notional) * m.step)
	if balance < required {
		// not enough moneyz... calculate how much we need to transfer
		// optimal := uint64(float(notional) * (m.base + m.step))
		// get the optimal margin, subtract the amount
		transfer.Type = types.TransferType_MARGIN_LOW
		return &marginChange{
			Margin:   evt,
			amount:   transfer.Amount.Amount,
			transfer: &transfer,
		}
	}
	high := uint64(float64(notional) * (m.base + 2*m.step))
	if balance <= high {
		// more than enough margin, but not enough to move to general account
		return nil
	}
	transfer.Amount.Amount *= -1 // make absolute value
	transfer.Type = types.TransferType_MARGIN_HIGH
	// balance > high -> too much collateral either way, we need to move money
	return &marginChange{
		Margin:   evt,
		amount:   transfer.Amount.Amount,
		transfer: &transfer,
	}
}
