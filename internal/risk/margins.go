package risk

import (
	"errors"
	"math"

	"code.vegaprotocol.io/vega/internal/events"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrNoFactorForAsset = errors.New("no risk factors found for given asset")
)

const (
	marginShort marginSide = iota
	marginLong
)

type marginSide int

type limits struct {
	lower, initial, top uint64
}

type limit struct {
	long, short, base, step float64
}

// there is a system limit, and a trader one
type marginAmount struct {
	system limit
	trader limit
}

// Margins: There are 2 sets of margins: system margins (the minimum collateral required by the system)
//          and trader facing marings. If a trader falls short of the trader base, we will restore the initial margin
//          should this not be possible, we need to check if the trader is above the system base, if so, we don't close out the trader
//          If they are, we close them out. Both these values are used in the calculations and transfer requests, where the amount
//          is the TRADER facing amount (trader initial balance), and the minimum amount is the SYSTEM base.
func (e *Engine) getMargins(asset string) (*marginAmount, error) {
	factor, ok := e.factors.RiskFactors[asset]
	if !ok {
		return nil, ErrNoFactorForAsset
	}
	sys := limit{
		long:  factor.Long,
		short: factor.Short,
	}
	if sys.long == sys.short {
		sys.step = sys.long / 10
		sys.base = sys.long
	} else if sys.long > sys.short {
		sys.step = sys.long - sys.short
		sys.base = sys.long
	} else {
		sys.step = sys.short - sys.long
		sys.base = sys.short
	}
	// just halve the step to 5% or half the delta
	sys.step /= 2
	trader := sys
	// trader values are a bit higher than the standard ones
	trader.short += sys.step
	trader.long += sys.step
	trader.base += sys.step
	return &marginAmount{
		system: sys,
		trader: trader,
	}, nil
}

func (m marginAmount) getTransfer(evt events.Margin, markPrice uint64) *marginChange {
	// get longest/shortest position
	long, short := int64(math.Abs(float64(evt.Size()+evt.Buy()))), int64(math.Abs(float64(evt.Size()-evt.Sell())))
	notionalLong := int64(markPrice) * long
	notionalShort := int64(markPrice) * short
	longReq := uint64(float64(notionalLong) * m.system.long)
	shortReq := uint64(float64(notionalShort) * m.system.short)
	// marginBalance := evt.MarginBalance()
	if longReq > shortReq {
		// use long as min to calculate required margin
		return nil
	}
	// use short as starting point
	return nil
}

func (m marginAmount) getChange(evt events.Margin, markPrice uint64) *marginChange {
	vol := evt.Size()
	if vol == 0 {
		return nil
	}
	absVol := vol
	if vol < 0 {
		absVol *= -1
	}
	notional := int64(markPrice) * absVol
	var (
		req uint64
	)
	// this is always the minimum required margin for the system
	sysReq := uint64(float64(notional) * m.system.base)
	// trader is short
	if vol < 0 {
		req = uint64(float64(notional) * m.trader.short)
	} else {
		req = uint64(float64(notional) * m.trader.long)
	}
	balance := evt.MarginBalance()
	// spot on, no further action needed
	if balance == req {
		return nil
	}
	// *if* an amounts needs to be moved, this is the amount...
	transfer := types.Transfer{
		Owner: evt.Party(),
		Size:  1,
		Amount: &types.FinancialAmount{
			Asset:     evt.Asset(),
			Amount:    int64(float64(notional)*(m.trader.base+m.trader.step)) - int64(balance), // add step to trader base again
			MinAmount: int64(sysReq) - int64(balance),                                          // this is always the minimal amount required by the system
		},
		// Type: types.TransferType_MARGIN_{LOW,HIGH}, <-- these types need to be added to the proto
	}
	// balance of trader is below the minimum trader requirement for long/short position
	if balance < req {
		transfer.Type = types.TransferType_MARGIN_LOW
		return &marginChange{
			Margin:   evt,
			amount:   transfer.Amount.Amount,
			transfer: &transfer,
		}
	}
	high := uint64(float64(notional) * (m.trader.base + 2*m.trader.step)) // twice the step is our margin "overflow" value for now
	if balance <= high {
		// enough margin, not too much
		return nil
	}
	transfer.Amount.Amount *= -1 // make absolute value, the balance was more than the margin value after all
	transfer.Amount.MinAmount *= -1
	transfer.Type = types.TransferType_MARGIN_HIGH
	// balance > high -> too much collateral either way, we need to move money
	return &marginChange{
		Margin:   evt,
		amount:   transfer.Amount.Amount,
		transfer: &transfer,
	}
}
