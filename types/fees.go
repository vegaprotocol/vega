package types

import (
	proto "code.vegaprotocol.io/protos/vega"
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
		mf = num.DecimalZero()
	}
	inf, err := num.DecimalFromString(f.InfrastructureFee)
	if err != nil {
		inf = num.DecimalZero()
	}
	lf, err := num.DecimalFromString(f.LiquidityFee)
	if err != nil {
		lf = num.DecimalZero()
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

func (f FeeFactors) DeepClone() *FeeFactors {
	return &FeeFactors{
		MakerFee:          f.MakerFee,
		InfrastructureFee: f.InfrastructureFee,
		LiquidityFee:      f.LiquidityFee,
	}
}

type Fees struct {
	Factors *FeeFactors
}

func FeesFromProto(f *proto.Fees) *Fees {
	if f == nil {
		return nil
	}
	return &Fees{
		Factors: FeeFactorsFromProto(f.Factors),
	}
}

func (f Fees) IntoProto() *proto.Fees {
	return &proto.Fees{
		Factors: f.Factors.IntoProto(),
	}
}

func (f Fees) DeepClone() *Fees {
	return &Fees{
		Factors: f.Factors.DeepClone(),
	}
}

type Fee struct {
	MakerFee          *num.Uint
	InfrastructureFee *num.Uint
	LiquidityFee      *num.Uint
}

func (f Fee) IntoProto() *proto.Fee {
	return &proto.Fee{
		MakerFee:          num.UintToString(f.MakerFee),
		InfrastructureFee: num.UintToString(f.InfrastructureFee),
		LiquidityFee:      num.UintToString(f.LiquidityFee),
	}
}

func (f Fee) Clone() *Fee {
	return &Fee{
		MakerFee:          f.MakerFee.Clone(),
		InfrastructureFee: f.InfrastructureFee.Clone(),
		LiquidityFee:      f.LiquidityFee.Clone(),
	}
}

// NewFee returns a new fee object, with all fields initialised.
func NewFee() *Fee {
	return &Fee{
		MakerFee:          num.Zero(),
		InfrastructureFee: num.Zero(),
		LiquidityFee:      num.Zero(),
	}
}
