package types

import (
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types/num"
)

type FeeFactors struct {
	MakerFee          num.Decimal
	InfrastructureFee num.Decimal
	LiquidityFee      num.Decimal
}

func (f FeeFactors) IntoProto() *proto.FeeFactors {
	return &proto.FeeFactors{
		MakerFee:          f.MakerFee.String(),
		InfrastructureFee: f.InfrastructureFee.String(),
		LiquidityFee:      f.LiquidityFee.String(),
	}
}

type Fees struct {
	Factors *FeeFactors
}

func (f Fees) IntoProto() *proto.Fees {
	return &proto.Fees{
		Factors: f.Factors.IntoProto(),
	}
}

type Fee struct {
	MakerFee          *num.Uint
	InfrastructureFee *num.Uint
	LiquidityFee      *num.Uint
}

func (f Fee) IntoProto() *proto.Fee {
	return &proto.Fee{
		MakerFee:          f.MakerFee.Uint64(),
		InfrastructureFee: f.InfrastructureFee.Uint64(),
		LiquidityFee:      f.LiquidityFee.Uint64(),
	}
}

func (f Fee) Clone() *Fee {
	return &Fee{
		MakerFee:          f.MakerFee.Clone(),
		InfrastructureFee: f.InfrastructureFee.Clone(),
		LiquidityFee:      f.LiquidityFee.Clone(),
	}
}

// NewFee returns a new fee object, with all fields initialised
func NewFee() *Fee {
	f := &Fee{}
	f.Init()
	return f
}

func (f *Fee) Init() {
	f.MakerFee = num.NewUint(0)
	f.InfrastructureFee = num.NewUint(0)
	f.LiquidityFee = num.NewUint(0)
}
