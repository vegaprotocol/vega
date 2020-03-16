package gql

import (
	"fmt"
	"math"
	"strconv"

	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/gqlerror"
	"google.golang.org/grpc/status"

	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

func safeStringUint64(input string) (uint64, error) {
	if i, err := strconv.ParseUint(input, 10, 64); err == nil {
		return i, nil
	}
	// A conversion error occurred, return the error
	return 0, fmt.Errorf("invalid input string for uint64 conversion %s", input)
}

func convertInterval(interval Interval) (types.Interval, error) {
	switch interval {
	case IntervalI15m:
		return types.Interval_I15M, nil
	case IntervalI1d:
		return types.Interval_I1D, nil
	case IntervalI1h:
		return types.Interval_I1H, nil
	case IntervalI1m:
		return types.Interval_I1M, nil
	case IntervalI5m:
		return types.Interval_I5M, nil
	case IntervalI6h:
		return types.Interval_I6H, nil
	default:
		err := fmt.Errorf("invalid interval when subscribing to candles, falling back to default: I15M, (%v)", interval)

		return types.Interval_I15M, err
	}
}

func parseOrderTimeInForce(timeInForce OrderTimeInForce) (types.Order_TimeInForce, error) {
	switch timeInForce {
	case OrderTimeInForceGtc:
		return types.Order_GTC, nil
	case OrderTimeInForceGtt:
		return types.Order_GTT, nil
	case OrderTimeInForceIoc:
		return types.Order_IOC, nil
	case OrderTimeInForceFok:
		return types.Order_FOK, nil
	default:
		return types.Order_GTC, fmt.Errorf("unknown type: %s", timeInForce.String())
	}
}

func parseOrderType(ty OrderType) (types.Order_Type, error) {
	switch ty {
	case OrderTypeLimit:
		return types.Order_LIMIT, nil
	case OrderTypeMarket:
		return types.Order_MARKET, nil
	default:
		// handle types.Order_NETWORK as an error here, as we do not expected
		// it to be set by through the API, only by the core internally
		return 0, fmt.Errorf("unknown type: %s", ty.String())
	}
}

func parseOrderStatus(orderStatus *OrderStatus) (types.Order_Status, error) {
	switch *orderStatus {
	case OrderStatusActive:
		return types.Order_Active, nil
	case OrderStatusExpired:
		return types.Order_Expired, nil
	case OrderStatusCancelled:
		return types.Order_Cancelled, nil
	case OrderStatusFilled:
		return types.Order_Filled, nil
	case OrderStatusRejected:
		return types.Order_Rejected, nil
	default:
		return types.Order_Active, fmt.Errorf("unknown status: %s", orderStatus.String())
	}
}

func parseSide(side *Side) (types.Side, error) {
	switch *side {
	case SideBuy:
		return types.Side_Buy, nil
	case SideSell:
		return types.Side_Sell, nil
	default:
		return types.Side_Buy, fmt.Errorf("unknown side: %s", side.String())
	}
}

// customErrorFromStatus provides a richer error experience from grpc ErrorDetails
// which is provided by the Vega grpc API. This helper takes in the error provided
// by a grpc client and either returns a custom graphql error or the raw error string.
func customErrorFromStatus(err error) error {
	st, ok := status.FromError(err)
	if ok {
		customCode := ""
		customDetail := ""
		customInner := ""
		customMessage := st.Message()
		errorDetails := st.Details()
		if errorDetails != nil {
			for _, s := range errorDetails {
				det := s.(*types.ErrorDetail)
				customDetail = det.Message
				customCode = fmt.Sprintf("%d", det.Code)
				customInner = det.Inner
				break
			}
		}
		return &gqlerror.Error{
			Message: customMessage,
			Extensions: map[string]interface{}{
				"detail": customDetail,
				"code":   customCode,
				"inner":  customInner,
			},
		}
	}
	return err
}

func timestampToString(timestampInSeconds int64) string {
	return vegatime.Format(vegatime.Unix(timestampInSeconds, 0))
}

func parseTimestamp(timestamp string) (int64, error) {
	converted, err := vegatime.Parse(timestamp)
	if err != nil {
		return 0, err
	}
	return converted.UTC().Unix(), nil
}

func removePointers(input []*string) []string {
	result := make([]string, 0, len(input))
	for _, sPtr := range input {
		if sPtr != nil {
			result = append(result, *sPtr)
		}
	}
	return result
}

func convertProposalNewMarketTerms(changes *MarketInput) (*types.Market, error) {
	initMarkPrice, err := safeStringUint64(changes.TradableInstrument.Instrument.InitialMarkPrice)
	if err != nil {
		return nil, errors.Wrap(err, "initialMarkPrice is invalid")
	}

	if changes.DecimalPlaces < 0 {
		return nil, errors.Wrap(err, "decimalPlaces is invalid")
	}

	result := &types.Market{
		Id:   changes.ID,
		Name: changes.Name,
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				Id:        changes.TradableInstrument.Instrument.ID,
				Code:      changes.TradableInstrument.Instrument.Code,
				Name:      changes.TradableInstrument.Instrument.Name,
				BaseName:  changes.TradableInstrument.Instrument.BaseName,
				QuoteName: changes.TradableInstrument.Instrument.QuoteName,
				Metadata: &types.InstrumentMetadata{
					Tags: removePointers(changes.TradableInstrument.Instrument.Metadata.Tags),
				},
				InitialMarkPrice: initMarkPrice,
				Product:          nil,
			},
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: &types.ScalingFactors{
					SearchLevel:       changes.TradableInstrument.MarginCalculator.ScalingFactors.SearchLevel,
					InitialMargin:     changes.TradableInstrument.MarginCalculator.ScalingFactors.InitialMargin,
					CollateralRelease: changes.TradableInstrument.MarginCalculator.ScalingFactors.CollateralRelease,
				},
			},
			RiskModel: nil,
		},
		DecimalPlaces: uint64(changes.DecimalPlaces),
		TradingMode:   nil,
	}
	if future := changes.TradableInstrument.Instrument.FutureProduct; future != nil {
		result.TradableInstrument.Instrument.Product = &types.Instrument_Future{
			Future: &types.Future{
				Maturity: future.Maturity,
				Asset:    future.Asset,
				Oracle: &types.Future_EthereumEvent{
					EthereumEvent: &types.EthereumEvent{
						ContractID: future.EthereumOracle.ContractID,
						Event:      future.EthereumOracle.Event,
					},
				},
			},
		}
	}

	if continuous := changes.ContinuousTradingMode; continuous != nil {
		if continuous.TickSize < 0 {
			return nil, errors.Wrap(err, "tickSize is invalid")
		}
		result.TradingMode = &types.Market_Continuous{
			Continuous: &types.ContinuousTrading{
				TickSize: uint64(continuous.TickSize),
			},
		}
	} else if discrete := changes.DiscreteTradingMode; discrete != nil {
		result.TradingMode = &types.Market_Discrete{
			Discrete: &types.DiscreteTrading{
				Duration: int64(discrete.Duration),
			},
		}
	}

	if simple := changes.TradableInstrument.SimpleRiskModel; simple != nil {
		result.TradableInstrument.RiskModel = &types.TradableInstrument_SimpleRiskModel{
			SimpleRiskModel: &types.SimpleRiskModel{
				Params: &types.SimpleModelParams{
					FactorLong:  simple.Params.FactorLong,
					FactorShort: simple.Params.FactorShort,
				},
			},
		}
	} else if logNormal := changes.TradableInstrument.LogNormalRiskModel; logNormal != nil {
		result.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
			LogNormalRiskModel: &types.LogNormalRiskModel{
				RiskAversionParameter: logNormal.RiskAversionParameter,
				Tau:                   logNormal.Tau,
				Params: &types.LogNormalModelParams{
					Mu:    logNormal.Params.Mu,
					R:     logNormal.Params.R,
					Sigma: logNormal.Params.Sigma,
				},
			},
		}
	}

	return result, nil
}

func convertProposalTermsInput(terms ProposalTermsInput) (*types.ProposalTerms, error) {
	closing, err := parseTimestamp(terms.ClosingTimestamp)
	if err != nil {
		return nil, errors.Wrap(err, "closingTimestamp is invalid")
	}
	enactment, err := parseTimestamp(terms.EnactmentTimestamp)
	if err != nil {
		return nil, errors.Wrap(err, "enactmentTimestamp is invalid")
	}

	result := &types.ProposalTerms{
		ClosingTimestamp:      closing,
		EnactmentTimestamp:    enactment,
		MinParticipationStake: uint64(terms.MinParticipationStake),
	}
	if terms.UpdateMarket != nil {
		result.Change = &types.ProposalTerms_UpdateMarket{}
	} else if terms.NewMarket != nil {
		market, err := convertProposalNewMarketTerms(terms.NewMarket.Market)
		if err != nil {
			return nil, err
		}
		result.Change = &types.ProposalTerms_NewMarket{
			NewMarket: &types.NewMarket{
				Changes: market,
			},
		}
	} else if terms.UpdateNetwork != nil {
		result.Change = &types.ProposalTerms_UpdateMarket{}
	} else {
		return nil, errors.New("updateMarket, newMarket or updateNetwork must be set")
	}

	return result, nil
}

func convertProposalTerms(terms *types.ProposalTerms) (*ProposalTerms, error) {
	if terms.MinParticipationStake > math.MaxInt32 {
		return nil, errors.New("minParticipationStake contains too large value")
	}
	result := &ProposalTerms{
		ClosingTimestamp:      timestampToString(terms.ClosingTimestamp),
		EnactmentTimestamp:    timestampToString(terms.EnactmentTimestamp),
		MinParticipationStake: int(terms.MinParticipationStake),
	}
	if terms.GetUpdateMarket() != nil {
		result.Change = nil
	} else if terms.GetNewMarket() != nil {
		result.Change = nil
	} else if terms.GetUpdateNetwork() != nil {
		result.Change = nil
	}
	return result, nil
}
