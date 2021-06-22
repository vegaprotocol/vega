//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/types/num"
)

//type LiquidityMonitoringParameters = proto.LiquidityMonitoringParameters
//type LiquidityProvisionSubmission = commandspb.LiquidityProvisionSubmission
//type LiquidityProvision = proto.LiquidityProvision
//type LiquidityOrder = proto.LiquidityOrder
//type LiquidityOrderReference = proto.LiquidityOrderReference

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

func (t *TargetStakeParameters) FromProto(p *proto.TargetStakeParameters) {
	t.ScalingFactor = num.DecimalFromFloat(p.ScalingFactor)
	t.TimeWindow = p.TimeWindow
}

func (t TargetStakeParameters) String() string {
	return t.IntoProto().String()
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

func NewLiquidityProvisionSubmissionFromProto(p *commandspb.LiquidityProvisionSubmission) (*LiquidityProvisionSubmission, error) {
	lps := &LiquidityProvisionSubmission{}
	if err := lps.FromProto(p); err != nil {
		return nil, err
	}
	return lps, nil
}

func (l *LiquidityProvisionSubmission) FromProto(p *commandspb.LiquidityProvisionSubmission) error {
	var err error
	l := LiquidityProvisionSubmission{}
	l.MarketId = p.MarketId
	// TODO UINT after proto is updated
	l.CommitmentAmount = num.NewUint(p.CommitmentAmount)
	l.Fee, err = num.DecimalFromString(p.Fee)
	if err != nil {
		return nil, err
	}

	l.Sells = make([]*LiquidityOrder, 0, len(p.Sells))
	for _, sell := range p.Sells {
		order := &LiquidityOrder{
			Reference:  sell.Reference,
			Proportion: sell.Proportion,
			Offset:     sell.Offset,
		}
		l.Sells = append(l.Sells, order)
	}

	l.Buys = make([]*LiquidityOrder, 0, len(p.Buys))
	for _, buy := range p.Buys {
		order := &LiquidityOrder{
			Reference:  buy.Reference,
			Proportion: buy.Proportion,
			Offset:     buy.Offset,
		}
		l.Buys = append(l.Buys, order)
	}
	l.Reference = p.Reference
	return &l, err
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
	}

	lp.Sells = make([]*proto.LiquidityOrderReference, 0)
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

	lp.Buys = make([]*proto.LiquidityOrderReference, 0)
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

func (l *LiquidityProvision) FromProto(p *proto.LiquidityProvision) {
	l.CommitmentAmount = num.NewUint(p.CommitmentAmount)
	l.CreatedAt = p.CreatedAt
	l.Fee, _ = num.DecimalFromString(p.Fee)
	l.Id = p.Id
	l.MarketId = p.MarketId
	l.PartyId = p.PartyId
	l.Reference = p.Reference
	l.Status = p.Status
	l.UpdatedAt = p.UpdatedAt
	l.Version = p.Version

	l.Sells = make([]*LiquidityOrderReference, 0, len(p.Sells))
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

	l.Buys = make([]*LiquidityOrderReference, 0, len(p.Buys))
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
}

type LiquidityOrderReference struct {
	// Unique identifier of the pegged order generated by the core to fulfil this liquidity order
	OrderId string
	// The liquidity order from the original submission
	LiquidityOrder *LiquidityOrder
}

func (l LiquidityOrderReference) IntoProto() *proto.LiquidityOrderReference {
	lor := &proto.LiquidityOrderReference{
		OrderId:        l.OrderId,
		LiquidityOrder: l.LiquidityOrder.IntoProto(),
	}
	return lor
}

func (l *LiquidityOrderReference) FromProto(p *proto.LiquidityOrderReference) {
	l.OrderId = p.OrderId
	l.LiquidityOrder = &LiquidityOrder{
		Reference:  p.LiquidityOrder.Reference,
		Proportion: p.LiquidityOrder.Proportion,
		Offset:     p.LiquidityOrder.Offset,
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

func (l LiquidityOrder) IntoProto() *proto.LiquidityOrder {
	lo := &proto.LiquidityOrder{
		Reference:  l.Reference,
		Proportion: l.Proportion,
		Offset:     l.Offset,
	}
	return lo
}

func (l *LiquidityOrder) FromProto(p *proto.LiquidityOrder) {
	l.Offset = p.Offset
	l.Proportion = p.Proportion
	l.Reference = p.Reference
}

type LiquidityMonitoringParameters struct {
	// Specifies parameters related to target stake calculation
	TargetStakeParameters *TargetStakeParameters
	// Specifies the triggering ratio for entering liquidity auction
	TriggeringRatio num.Decimal
	// Specifies by how many seconds an auction should be extended if leaving the auction were to trigger a liquidity auction
	AuctionExtension int64
}

func LiquidityMonitoringParametersFromProto(p *proto.LiquidityMonitoringParameters) *LiquidityMonitoringParameters {
	l := &LiquidityMonitoringParameters{}
	l.FromProto(p)
	return l
}

func (l LiquidityMonitoringParameters) IntoProto() *proto.LiquidityMonitoringParameters {
	tr, _ := l.TriggeringRatio.Float64()
	lmp := &proto.LiquidityMonitoringParameters{
		TargetStakeParameters: l.TargetStakeParameters.IntoProto(),
		TriggeringRatio:       tr,
		AuctionExtension:      l.AuctionExtension,
	}
	return lmp
}

func (l *LiquidityMonitoringParameters) FromProto(p *proto.LiquidityMonitoringParameters) {
	l.AuctionExtension = p.AuctionExtension
	l.TriggeringRatio = num.DecimalFromFloat(p.TriggeringRatio)
	l.TargetStakeParameters = &TargetStakeParameters{}
	l.TargetStakeParameters.FromProto(p.TargetStakeParameters)
}
