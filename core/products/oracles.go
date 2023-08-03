package products

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type scheduledOracle struct {
	settlementSubscriptionID spec.SubscriptionID
	scheduleSubscriptionID   spec.SubscriptionID
	binding                  scheduledBinding
	settleUnsub              spec.Unsubscriber
	scheduleUnsub            spec.Unsubscriber
}

type terminatingOracle struct {
	settlementSubscriptionID  spec.SubscriptionID
	terminationSubscriptionID spec.SubscriptionID
	settleUnsub               spec.Unsubscriber
	terminationUnsub          spec.Unsubscriber
	binding                   terminatingBinding
	data                      oracleData
}

type scheduledBinding struct {
	settlementProperty string
	settlementType     datapb.PropertyKey_Type
	settlementDecimals uint64

	scheduleProperty string
	scheduleType     datapb.PropertyKey_Type
}

type terminatingBinding struct {
	settlementProperty string
	settlementType     datapb.PropertyKey_Type
	settlementDecimals uint64

	terminationProperty string
	terminationType     datapb.PropertyKey_Type
}

type oracleData struct {
	settlData         *num.Numeric
	tradingTerminated bool
}

func newFutureOracle(f *types.Future) (terminatingOracle, error) {
	bind, err := newFutureBinding(f)
	if err != nil {
		return terminatingOracle{}, err
	}
	return terminatingOracle{
		binding: bind,
	}, nil
}

func newFutureBinding(f *types.Future) (terminatingBinding, error) {
	settlementProperty := strings.TrimSpace(f.DataSourceSpecBinding.SettlementDataProperty)
	if len(settlementProperty) == 0 {
		return terminatingBinding{}, errors.New("binding for settlement data cannot be blank")
	}
	tradingTerminationProperty := strings.TrimSpace(f.DataSourceSpecBinding.TradingTerminationProperty)
	if len(tradingTerminationProperty) == 0 {
		return terminatingBinding{}, errors.New("binding for trading termination market cannot be blank")
	}
	// assume bool for now, check for built-in timestamp
	// this can be set to anything else by the caller
	termType := datapb.PropertyKey_TYPE_BOOLEAN
	if tradingTerminationProperty == spec.BuiltinTimestamp {
		termType = datapb.PropertyKey_TYPE_TIMESTAMP
	}
	t, dec := getSettleTypeAndDec(f.DataSourceSpecForSettlementData)

	return terminatingBinding{
		settlementProperty:  settlementProperty,
		settlementType:      t,
		settlementDecimals:  dec,
		terminationProperty: tradingTerminationProperty,
		terminationType:     termType,
	}, nil
}

func (t *terminatingOracle) bindAll(ctx context.Context, oe OracleEngine, settle, term *spec.Spec, settleCB, termCB spec.OnMatchedData) error {
	if err := t.bindSettlement(ctx, oe, settle, settleCB); err != nil {
		return err
	}
	return t.bindTermination(ctx, oe, term, termCB)
}

func (t *terminatingOracle) bindSettlement(ctx context.Context, oe OracleEngine, osForSettle *spec.Spec, cb spec.OnMatchedData) error {
	err := osForSettle.EnsureBoundableProperty(t.binding.settlementProperty, t.binding.settlementType)
	if err != nil {
		return fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
	}
	if t.settlementSubscriptionID, t.settleUnsub, err = oe.Subscribe(ctx, *osForSettle, cb); err != nil {
		return fmt.Errorf("could not subscribe to oracle engine for settlement data: %w", err)
	}
	return nil
}

func (t *terminatingOracle) bindTermination(ctx context.Context, oe OracleEngine, osForTerm *spec.Spec, cb spec.OnMatchedData) error {
	err := osForTerm.EnsureBoundableProperty(t.binding.terminationProperty, t.binding.terminationType)
	if err != nil {
		return fmt.Errorf("invalid oracle spec binding for trading termination: %w", err)
	}
	if t.terminationSubscriptionID, t.terminationUnsub, err = oe.Subscribe(ctx, *osForTerm, cb); err != nil {
		return fmt.Errorf("could not subscribe to oracle engine for trading termination: %w", err)
	}
	return nil
}

func (t *terminatingOracle) unsubSettle(ctx context.Context) {
	if t.settleUnsub != nil {
		t.settleUnsub(ctx, t.settlementSubscriptionID)
		t.settleUnsub = nil
	}
}

func (t *terminatingOracle) unsubTerm(ctx context.Context) {
	if t.terminationUnsub != nil {
		t.terminationUnsub(ctx, t.terminationSubscriptionID)
		t.terminationUnsub = nil
	}
}

func newPerpOracle(p *types.Perps) (scheduledOracle, error) {
	bind, err := newPerpBinding(p)
	if err != nil {
		return scheduledOracle{}, err
	}
	return scheduledOracle{
		binding: bind,
	}, nil
}

func newPerpBinding(p *types.Perps) (scheduledBinding, error) {
	settleDataProp := strings.TrimSpace(p.DataSourceSpecBinding.SettlementDataProperty)
	settleScheduleProp := strings.TrimSpace(p.DataSourceSpecBinding.SettlementScheduleProperty)
	if len(settleDataProp) == 0 {
		return scheduledBinding{}, errors.New("binding for settlement data cannot be blank")
	}
	if len(settleScheduleProp) == 0 {
		return scheduledBinding{}, errors.New("binding for settlement schedule cannot be blank")
	}
	setT, dec := getSettleTypeAndDec(p.DataSourceSpecForSettlementData)

	return scheduledBinding{
		settlementProperty: settleDataProp,
		settlementType:     setT,
		settlementDecimals: dec,
		scheduleProperty:   settleScheduleProp,
		scheduleType:       datapb.PropertyKey_TYPE_TIMESTAMP, // default to timestamp
	}, nil
}

func (s *scheduledOracle) bindAll(ctx context.Context, oe OracleEngine, settle, schedule *spec.Spec, settleCB, scheduleCB spec.OnMatchedData) error {
	if err := s.bindSettlement(ctx, oe, settle, settleCB); err != nil {
		return err
	}
	return s.bindSchedule(ctx, oe, schedule, scheduleCB)
}

func (s *scheduledOracle) bindSettlement(ctx context.Context, oe OracleEngine, osForSettle *spec.Spec, cb spec.OnMatchedData) error {
	err := osForSettle.EnsureBoundableProperty(s.binding.settlementProperty, s.binding.settlementType)
	if err != nil {
		return fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
	}
	if s.settlementSubscriptionID, s.settleUnsub, err = oe.Subscribe(ctx, *osForSettle, cb); err != nil {
		return fmt.Errorf("could not subscribe to oracle engine for settlement data: %w", err)
	}
	return nil
}

func (s *scheduledOracle) bindSchedule(ctx context.Context, oe OracleEngine, osForSchedule *spec.Spec, cb spec.OnMatchedData) error {
	err := osForSchedule.EnsureBoundableProperty(s.binding.scheduleProperty, s.binding.scheduleType)
	if err != nil {
		return fmt.Errorf("invalid  oracle spec binding for schedule data: %w", err)
	}
	if s.scheduleSubscriptionID, s.scheduleUnsub, err = oe.Subscribe(ctx, *osForSchedule, cb); err != nil {
		return fmt.Errorf("could not subscribe to oracle engine for schedule data: %w", err)
	}
	return nil
}

func (s *scheduledOracle) unsubAll(ctx context.Context) {
	if s.settleUnsub != nil {
		s.settleUnsub(ctx, s.settlementSubscriptionID)
		s.settleUnsub = nil
	}
	if s.scheduleUnsub != nil {
		s.scheduleUnsub(ctx, s.scheduleSubscriptionID)
		s.scheduleUnsub = nil
	}
}

// SettlementData returns oracle data settlement data scaled as Uint.
func (o *oracleData) SettlementData(op, ap uint32) (*num.Uint, error) {
	if o.settlData.Decimal() == nil && o.settlData.Uint() == nil {
		return nil, ErrDataSourceSettlementDataNotSet
	}

	if !o.settlData.SupportDecimalPlaces(int64(ap)) {
		return nil, ErrSettlementDataDecimalsNotSupportedByAsset
	}

	// scale to given target decimals by multiplying by 10^(targetDP - oracleDP)
	// if targetDP > oracleDP - this scales up the decimals of settlement data
	// if targetDP < oracleDP - this scaled down the decimals of settlement data and can lead to loss of accuracy
	// if there're equal - no scaling happens
	return o.settlData.ScaleTo(int64(op), int64(ap))
}

// IsTradingTerminated returns true when oracle has signalled termination of trading.
func (o *oracleData) IsTradingTerminated() bool {
	return o.tradingTerminated
}

func getSettleTypeAndDec(s *datasource.Spec) (datapb.PropertyKey_Type, uint64) {
	def := s.GetDefinition()
	defType := datapb.PropertyKey_TYPE_INTEGER
	dec := uint64(0)
	for _, f := range def.GetFilters() {
		if f.Key.Type == datapb.PropertyKey_TYPE_INTEGER && f.Key.NumberDecimalPlaces != nil {
			dec = *f.Key.NumberDecimalPlaces
			break
		} else if f.Key.Type == datapb.PropertyKey_TYPE_DECIMAL {
			defType = f.Key.Type
			if f.Key.NumberDecimalPlaces != nil {
				dec = *f.Key.NumberDecimalPlaces
			}
			break
		}
	}
	return defType, dec
}
