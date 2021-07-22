//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	proto "code.vegaprotocol.io/data-node/proto/vega"
	commandspb "code.vegaprotocol.io/data-node/proto/vega/commands/v1"
	"code.vegaprotocol.io/data-node/types/num"
)

type LiquidityProvision_Status = proto.LiquidityProvision_Status

const (
	// The default value
	LiquidityProvision_STATUS_UNSPECIFIED LiquidityProvision_Status = 0
	// The liquidity provision is active
	LiquidityProvision_STATUS_ACTIVE LiquidityProvision_Status = 1
	// The liquidity provision was stopped by the network
	LiquidityProvision_STATUS_STOPPED LiquidityProvision_Status = 2
	// The liquidity provision was cancelled by the liquidity provider
	LiquidityProvision_STATUS_CANCELLED LiquidityProvision_Status = 3
	// The liquidity provision was invalid and got rejected
	LiquidityProvision_STATUS_REJECTED LiquidityProvision_Status = 4
	// The liquidity provision is valid and accepted by network, but orders aren't deployed
	LiquidityProvision_STATUS_UNDEPLOYED LiquidityProvision_Status = 5
	// The liquidity provision is valid and accepted by network
	// but have never been deployed. I when it's possible to deploy them for the first time
	// margin check fails, then they will be cancelled without any penalties.
	LiquidityProvision_STATUS_PENDING LiquidityProvision_Status = 6
)

type TargetStakeParameters struct {
	TimeWindow    int64
	ScalingFactor num.Decimal
}

func (t TargetStakeParameters) IntoProto() *proto.TargetStakeParameters {
	sf, _ := t.ScalingFactor.Float64()
	return &proto.TargetStakeParameters{
		TimeWindow:    t.TimeWindow,
		ScalingFactor: sf,
	}
}

func TargetStakeParametersFromProto(p *proto.TargetStakeParameters) *TargetStakeParameters {
	return &TargetStakeParameters{
		TimeWindow:    p.TimeWindow,
		ScalingFactor: num.DecimalFromFloat(p.ScalingFactor),
	}
}

func (t TargetStakeParameters) String() string {
	return t.IntoProto().String()
}

func (t TargetStakeParameters) DeepClone() *TargetStakeParameters {
	return &TargetStakeParameters{
		TimeWindow:    t.TimeWindow,
		ScalingFactor: t.ScalingFactor,
	}
}

type LiquidityProvisionSubmission struct {
	// Market identifier for the order, required field
	MarketId string
	// Specified as a unitless number that represents the amount of settlement asset of the market
	CommitmentAmount *num.Uint
	// Nominated liquidity fee factor, which is an input to the calculation of taker fees on the market, as per setting fees and rewarding liquidity providers
	Fee num.Decimal
	// A set of liquidity sell orders to meet the liquidity provision obligation
	Sells []*LiquidityOrder
	// A set of liquidity buy orders to meet the liquidity provision obligation
	Buys []*LiquidityOrder
	// A reference to be added to every order created out of this liquidityProvisionSubmission
	Reference string
}

func (l LiquidityProvisionSubmission) IntoProto() *commandspb.LiquidityProvisionSubmission {
	lps := &commandspb.LiquidityProvisionSubmission{
		MarketId:         l.MarketId,
		CommitmentAmount: l.CommitmentAmount.Uint64(),
		Fee:              l.Fee.String(),
		Sells:            make([]*proto.LiquidityOrder, 0, len(l.Sells)),
		Buys:             make([]*proto.LiquidityOrder, 0, len(l.Buys)),
		Reference:        l.Reference,
	}

	for _, sell := range l.Sells {
		order := &proto.LiquidityOrder{
			Reference:  sell.Reference,
			Proportion: sell.Proportion,
			Offset:     sell.Offset,
		}
		lps.Sells = append(lps.Sells, order)
	}

	for _, buy := range l.Buys {
		order := &proto.LiquidityOrder{
			Reference:  buy.Reference,
			Proportion: buy.Proportion,
			Offset:     buy.Offset,
		}
		lps.Buys = append(lps.Buys, order)
	}
	return lps
}

func LiquidityProvisionSubmissionFromProto(p *commandspb.LiquidityProvisionSubmission) (*LiquidityProvisionSubmission, error) {
	fee, err := num.DecimalFromString(p.Fee)
	if err != nil {
		return nil, err
	}

	l := LiquidityProvisionSubmission{
		Fee:              fee,
		MarketId:         p.MarketId,
		CommitmentAmount: num.NewUint(p.CommitmentAmount),
		Sells:            make([]*LiquidityOrder, 0, len(p.Sells)),
		Buys:             make([]*LiquidityOrder, 0, len(p.Buys)),
		Reference:        p.Reference,
	}

	for _, sell := range p.Sells {
		order := &LiquidityOrder{
			Reference:  sell.Reference,
			Proportion: sell.Proportion,
			Offset:     sell.Offset,
		}
		l.Sells = append(l.Sells, order)
	}

	for _, buy := range p.Buys {
		order := &LiquidityOrder{
			Reference:  buy.Reference,
			Proportion: buy.Proportion,
			Offset:     buy.Offset,
		}
		l.Buys = append(l.Buys, order)
	}
	return &l, nil
}

func (l LiquidityProvisionSubmission) String() string {
	return l.IntoProto().String()
}

type LiquidityProvision struct {
	// Unique identifier
	Id string
	// Unique party identifier for the creator of the provision
	PartyId string
	// Timestamp for when the order was created at, in nanoseconds since the epoch
	// - See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`
	CreatedAt int64
	// Timestamp for when the order was updated at, in nanoseconds since the epoch
	// - See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`
	UpdatedAt int64
	// Market identifier for the order, required field
	MarketId string
	// Specified as a unitless number that represents the amount of settlement asset of the market
	CommitmentAmount *num.Uint
	// Nominated liquidity fee factor, which is an input to the calculation of taker fees on the market, as per seeting fees and rewarding liquidity providers
	Fee num.Decimal
	// A set of liquidity sell orders to meet the liquidity provision obligation
	Sells []*LiquidityOrderReference
	// A set of liquidity buy orders to meet the liquidity provision obligation
	Buys []*LiquidityOrderReference
	// Version of this liquidity provision order
	Version string
	// Status of this liquidity provision order
	Status LiquidityProvision_Status
	// A reference shared between this liquidity provision and all it's orders
	Reference string
}

func (l LiquidityProvision) String() string {
	return l.IntoProto().String()
}

func (l LiquidityProvision) IntoProto() *proto.LiquidityProvision {
	lp := &proto.LiquidityProvision{
		Id:               l.Id,
		PartyId:          l.PartyId,
		CreatedAt:        l.CreatedAt,
		UpdatedAt:        l.UpdatedAt,
		MarketId:         l.MarketId,
		CommitmentAmount: l.CommitmentAmount.Uint64(),
		Fee:              l.Fee.String(),
		Version:          l.Version,
		Status:           l.Status,
		Reference:        l.Reference,
		Sells:            make([]*proto.LiquidityOrderReference, 0, len(l.Sells)),
		Buys:             make([]*proto.LiquidityOrderReference, 0, len(l.Buys)),
	}

	for _, sell := range l.Sells {
		lor := &proto.LiquidityOrderReference{
			OrderId: sell.OrderId,
			LiquidityOrder: &proto.LiquidityOrder{
				Reference:  sell.LiquidityOrder.Reference,
				Proportion: sell.LiquidityOrder.Proportion,
				Offset:     sell.LiquidityOrder.Offset,
			},
		}
		lp.Sells = append(lp.Sells, lor)
	}

	for _, buy := range l.Buys {
		lor := &proto.LiquidityOrderReference{
			OrderId: buy.OrderId,
			LiquidityOrder: &proto.LiquidityOrder{
				Reference:  buy.LiquidityOrder.Reference,
				Proportion: buy.LiquidityOrder.Proportion,
				Offset:     buy.LiquidityOrder.Offset,
			},
		}
		lp.Buys = append(lp.Buys, lor)
	}
	return lp
}

func LiquidityProvisionFromProto(p *proto.LiquidityProvision) *LiquidityProvision {
	fee, _ := num.DecimalFromString(p.Fee)
	l := LiquidityProvision{
		CommitmentAmount: num.NewUint(p.CommitmentAmount),
		CreatedAt:        p.CreatedAt,
		Id:               p.Id,
		MarketId:         p.MarketId,
		PartyId:          p.PartyId,
		Fee:              fee,
		Reference:        p.Reference,
		Status:           p.Status,
		UpdatedAt:        p.UpdatedAt,
		Version:          p.Version,
		Sells:            make([]*LiquidityOrderReference, 0, len(p.Sells)),
		Buys:             make([]*LiquidityOrderReference, 0, len(p.Buys)),
	}

	for _, sell := range p.Sells {
		lor := &LiquidityOrderReference{
			OrderId: sell.OrderId,
			LiquidityOrder: &LiquidityOrder{
				Reference:  sell.LiquidityOrder.Reference,
				Proportion: sell.LiquidityOrder.Proportion,
				Offset:     sell.LiquidityOrder.Offset,
			},
		}
		l.Sells = append(l.Sells, lor)
	}

	for _, buy := range p.Buys {
		lor := &LiquidityOrderReference{
			OrderId: buy.OrderId,
			LiquidityOrder: &LiquidityOrder{
				Reference:  buy.LiquidityOrder.Reference,
				Proportion: buy.LiquidityOrder.Proportion,
				Offset:     buy.LiquidityOrder.Offset,
			},
		}
		l.Buys = append(l.Buys, lor)
	}

	return &l
}

type LiquidityOrderReference struct {
	// Unique identifier of the pegged order generated by the core to fulfil this liquidity order
	OrderId string
	// The liquidity order from the original submission
	LiquidityOrder *LiquidityOrder
}

func (l LiquidityOrderReference) IntoProto() *proto.LiquidityOrderReference {
	var order *proto.LiquidityOrder
	if l.LiquidityOrder != nil {
		order = l.LiquidityOrder.IntoProto()
	}
	return &proto.LiquidityOrderReference{
		OrderId:        l.OrderId,
		LiquidityOrder: order,
	}
}

func LiquidityOrderReferenceFromProto(p *proto.LiquidityOrderReference) *LiquidityOrderReference {
	return &LiquidityOrderReference{
		OrderId: p.OrderId,
		LiquidityOrder: &LiquidityOrder{
			Reference:  p.LiquidityOrder.Reference,
			Proportion: p.LiquidityOrder.Proportion,
			Offset:     p.LiquidityOrder.Offset,
		},
	}
}

type LiquidityOrder struct {
	// The pegged reference point for the order
	Reference PeggedReference
	// The relative proportion of the commitment to be allocated at a price level
	Proportion uint32
	// The offset/amount of units away for the order
	Offset int64
}

func (l LiquidityOrder) DeepClone() *LiquidityOrder {
	return &LiquidityOrder{
		Reference:  l.Reference,
		Proportion: l.Proportion,
		Offset:     l.Offset,
	}
}

func (l LiquidityOrder) IntoProto() *proto.LiquidityOrder {
	return &proto.LiquidityOrder{
		Reference:  l.Reference,
		Proportion: l.Proportion,
		Offset:     l.Offset,
	}
}

func LiquidityOrderFromProto(p *proto.LiquidityOrder) *LiquidityOrder {
	return &LiquidityOrder{
		Offset:     p.Offset,
		Proportion: p.Proportion,
		Reference:  p.Reference,
	}
}

type LiquidityMonitoringParameters struct {
	// Specifies parameters related to target stake calculation
	TargetStakeParameters *TargetStakeParameters
	// Specifies the triggering ratio for entering liquidity auction
	TriggeringRatio num.Decimal
	// Specifies by how many seconds an auction should be extended if leaving the auction were to trigger a liquidity auction
	AuctionExtension int64
}

func (l LiquidityMonitoringParameters) IntoProto() *proto.LiquidityMonitoringParameters {
	tr, _ := l.TriggeringRatio.Float64()
	var params *proto.TargetStakeParameters
	if l.TargetStakeParameters != nil {
		params = l.TargetStakeParameters.IntoProto()
	}
	return &proto.LiquidityMonitoringParameters{
		TargetStakeParameters: params,
		TriggeringRatio:       tr,
		AuctionExtension:      l.AuctionExtension,
	}
}

func (l LiquidityMonitoringParameters) DeepClone() *LiquidityMonitoringParameters {
	var params *TargetStakeParameters
	if l.TargetStakeParameters != nil {
		params = l.TargetStakeParameters.DeepClone()
	}
	return &LiquidityMonitoringParameters{
		TriggeringRatio:       l.TriggeringRatio,
		AuctionExtension:      l.AuctionExtension,
		TargetStakeParameters: params,
	}
}

func LiquidityMonitoringParametersFromProto(p *proto.LiquidityMonitoringParameters) *LiquidityMonitoringParameters {
	var params *TargetStakeParameters
	if p.TargetStakeParameters != nil {
		params = TargetStakeParametersFromProto(p.TargetStakeParameters)
	}
	return &LiquidityMonitoringParameters{
		TargetStakeParameters: params,
		AuctionExtension:      p.AuctionExtension,
		TriggeringRatio:       num.DecimalFromFloat(p.TriggeringRatio),
	}
}

func LiquidityProvisionSubmissionFromMarketCommitment(
	nmc *NewMarketCommitment,
	market string,
) *LiquidityProvisionSubmission {
	return &LiquidityProvisionSubmission{
		MarketId:         market,
		CommitmentAmount: nmc.CommitmentAmount,
		Fee:              nmc.Fee,
		Sells:            nmc.Sells,
		Buys:             nmc.Buys,
		Reference:        nmc.Reference,
	}
}
