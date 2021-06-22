package types

import (
	"fmt"

	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types/num"
)

type FeeFactors struct {
	MakerFee          num.Decimal
	InfrastructureFee num.Decimal
	LiquidityFee      num.Decimal
}

func FeeFactorsFromProto(f *proto.FeeFactors) *FeeFactors {
	mf, err := num.DecimalFromString(f.MakerFee)
	if err != nil {
		panic(fmt.Sprintf("Could not convert maker fee %s to float: %+v", f.MakerFee, err))
	}
	inf, err := num.DecimalFromString(f.InfrastructureFee)
	if err != nil {
		panic(fmt.Sprintf("Could not convert infrastructure fee %s to float: %+v", f.InfrastructureFee, err))
	}
	lf, err := num.DecimalFromString(f.LiquidityFee)
	if err != nil {
		panic(fmt.Sprintf("Could not convert liquidity fee %s to float: %+v", f.LiquidityFee, err))
	}
	return &FeeFactors{
		MakerFee:          mf,
		InfrastructureFee: inf,
		LiquidityFee:      lf,
	}
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

func FeesFromProto(f *proto.Fees) *Fees {
	return &Fees{
		Factors: FeeFactorsFromProto(f.Factors),
	}
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
