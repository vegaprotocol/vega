// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package types

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	proto "code.vegaprotocol.io/vega/protos/vega"
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

func (f FeeFactors) String() string {
	return fmt.Sprintf(
		"makerFee(%s) liquidityFee(%s) infrastructureFee(%s)",
		f.MakerFee.String(),
		f.LiquidityFee.String(),
		f.InfrastructureFee.String(),
	)
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

func (f Fees) String() string {
	return fmt.Sprintf(
		"factors(%s)",
		stringer.ReflectPointerToString(f.Factors),
	)
}

type Fee struct {
	MakerFee          *num.Uint
	InfrastructureFee *num.Uint
	LiquidityFee      *num.Uint

	MakerFeeVolumeDiscount          *num.Uint
	InfrastructureFeeVolumeDiscount *num.Uint
	LiquidityFeeVolumeDiscount      *num.Uint

	MakerFeeReferrerDiscount          *num.Uint
	InfrastructureFeeReferrerDiscount *num.Uint
	LiquidityFeeReferrerDiscount      *num.Uint
}

func (f Fee) IntoProto() *proto.Fee {
	fee := &proto.Fee{
		MakerFee:                          num.UintToString(f.MakerFee),
		InfrastructureFee:                 num.UintToString(f.InfrastructureFee),
		LiquidityFee:                      num.UintToString(f.LiquidityFee),
		MakerFeeVolumeDiscount:            num.UintToString(f.MakerFeeVolumeDiscount),
		InfrastructureFeeVolumeDiscount:   num.UintToString(f.InfrastructureFeeVolumeDiscount),
		LiquidityFeeVolumeDiscount:        num.UintToString(f.LiquidityFeeVolumeDiscount),
		MakerFeeReferrerDiscount:          num.UintToString(f.MakerFeeReferrerDiscount),
		InfrastructureFeeReferrerDiscount: num.UintToString(f.InfrastructureFeeReferrerDiscount),
		LiquidityFeeReferrerDiscount:      num.UintToString(f.LiquidityFeeReferrerDiscount),
	}

	return fee
}

func (f Fee) Clone() *Fee {
	fee := &Fee{
		MakerFee:          f.MakerFee.Clone(),
		InfrastructureFee: f.InfrastructureFee.Clone(),
		LiquidityFee:      f.LiquidityFee.Clone(),
	}
	if f.MakerFeeVolumeDiscount != nil {
		fee.MakerFeeVolumeDiscount = f.MakerFeeVolumeDiscount.Clone()
	}
	if f.InfrastructureFeeVolumeDiscount != nil {
		fee.InfrastructureFeeVolumeDiscount = f.InfrastructureFeeVolumeDiscount.Clone()
	}
	if f.LiquidityFeeVolumeDiscount != nil {
		fee.LiquidityFeeVolumeDiscount = f.LiquidityFeeVolumeDiscount.Clone()
	}
	if f.MakerFeeReferrerDiscount != nil {
		fee.MakerFeeReferrerDiscount = f.MakerFeeReferrerDiscount.Clone()
	}
	if f.InfrastructureFeeReferrerDiscount != nil {
		fee.InfrastructureFeeReferrerDiscount = f.InfrastructureFeeReferrerDiscount.Clone()
	}
	if f.LiquidityFeeReferrerDiscount != nil {
		fee.LiquidityFeeReferrerDiscount = f.LiquidityFeeReferrerDiscount.Clone()
	}
	return fee
}

func (f *Fee) String() string {
	return fmt.Sprintf(
		"makerFee(%s) liquidityFee(%s) infrastructureFee(%s) makerFeeReferrerDiscount(%s) liquidityFeeReferrerDiscount(%s) infrastructureFeeReferrerDiscount(%s) makerFeeVolumeDiscount(%s) liquidityFeeVolumeDiscount(%s) infrastructureFeeVolumeDiscount(%s)",
		stringer.UintPointerToString(f.MakerFee),
		stringer.UintPointerToString(f.LiquidityFee),
		stringer.UintPointerToString(f.InfrastructureFee),
		stringer.UintPointerToString(f.MakerFeeReferrerDiscount),
		stringer.UintPointerToString(f.LiquidityFeeReferrerDiscount),
		stringer.UintPointerToString(f.InfrastructureFeeReferrerDiscount),
		stringer.UintPointerToString(f.MakerFeeVolumeDiscount),
		stringer.UintPointerToString(f.LiquidityFeeVolumeDiscount),
		stringer.UintPointerToString(f.InfrastructureFeeVolumeDiscount),
	)
}

// NewFee returns a new fee object, with all fields initialised.
func NewFee() *Fee {
	return &Fee{
		MakerFee:          num.UintZero(),
		InfrastructureFee: num.UintZero(),
		LiquidityFee:      num.UintZero(),
	}
}

type ReferrerReward struct {
	MakerFeeReferrerReward          *num.Uint
	InfrastructureFeeReferrerReward *num.Uint
	LiquidityFeeReferrerReward      *num.Uint
}

func (rf ReferrerReward) Clone() *ReferrerReward {
	return &ReferrerReward{
		MakerFeeReferrerReward:          rf.MakerFeeReferrerReward.Clone(),
		InfrastructureFeeReferrerReward: rf.InfrastructureFeeReferrerReward.Clone(),
		LiquidityFeeReferrerReward:      rf.LiquidityFeeReferrerReward.Clone(),
	}
}

func (rf *ReferrerReward) String() string {
	return fmt.Sprintf(
		"makerFeeReferrerReward(%s) liquidityFeeReferrerReward(%s) infrastructureFeeReferrerReward(%s)",
		stringer.UintPointerToString(rf.MakerFeeReferrerReward),
		stringer.UintPointerToString(rf.LiquidityFeeReferrerReward),
		stringer.UintPointerToString(rf.InfrastructureFeeReferrerReward),
	)
}

func NewReferrerReward() *ReferrerReward {
	return &ReferrerReward{
		MakerFeeReferrerReward:          num.UintZero(),
		InfrastructureFeeReferrerReward: num.UintZero(),
		LiquidityFeeReferrerReward:      num.UintZero(),
	}
}
