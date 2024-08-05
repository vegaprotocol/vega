// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	TreasuryFee       num.Decimal
	BuyBackFee        num.Decimal
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
	bbf, err := num.DecimalFromString(f.BuyBackFee)
	if err != nil {
		bbf = num.DecimalZero()
	}
	tf, err := num.DecimalFromString(f.TreasuryFee)
	if err != nil {
		tf = num.DecimalZero()
	}

	return &FeeFactors{
		MakerFee:          mf,
		InfrastructureFee: inf,
		LiquidityFee:      lf,
		BuyBackFee:        bbf,
		TreasuryFee:       tf,
	}
}

func (f FeeFactors) IntoProto() *proto.FeeFactors {
	return &proto.FeeFactors{
		MakerFee:          f.MakerFee.String(),
		InfrastructureFee: f.InfrastructureFee.String(),
		LiquidityFee:      f.LiquidityFee.String(),
		BuyBackFee:        f.BuyBackFee.String(),
		TreasuryFee:       f.TreasuryFee.String(),
	}
}

func (f FeeFactors) DeepClone() *FeeFactors {
	return &FeeFactors{
		MakerFee:          f.MakerFee,
		InfrastructureFee: f.InfrastructureFee,
		LiquidityFee:      f.LiquidityFee,
		BuyBackFee:        f.BuyBackFee,
		TreasuryFee:       f.TreasuryFee,
	}
}

func (f FeeFactors) String() string {
	return fmt.Sprintf(
		"makerFee(%s) liquidityFee(%s) infrastructureFee(%s) buyBackFee(%s) treasuryFee(%s)",
		f.MakerFee.String(),
		f.LiquidityFee.String(),
		f.InfrastructureFee.String(),
		f.BuyBackFee.String(),
		f.TreasuryFee.String(),
	)
}

type Fees struct {
	Factors              *FeeFactors
	LiquidityFeeSettings *LiquidityFeeSettings
}

func FeesFromProto(f *proto.Fees) *Fees {
	if f == nil {
		return nil
	}
	return &Fees{
		Factors:              FeeFactorsFromProto(f.Factors),
		LiquidityFeeSettings: LiquidityFeeSettingsFromProto(f.LiquidityFeeSettings),
	}
}

func (f Fees) IntoProto() *proto.Fees {
	return &proto.Fees{
		Factors:              f.Factors.IntoProto(),
		LiquidityFeeSettings: f.LiquidityFeeSettings.IntoProto(),
	}
}

func (f Fees) DeepClone() *Fees {
	return &Fees{
		Factors:              f.Factors.DeepClone(),
		LiquidityFeeSettings: f.LiquidityFeeSettings.DeepClone(),
	}
}

func (f Fees) String() string {
	return fmt.Sprintf(
		"factors(%s)",
		stringer.PtrToString(f.Factors),
	)
}

type Fee struct {
	MakerFee           *num.Uint
	InfrastructureFee  *num.Uint
	LiquidityFee       *num.Uint
	TreasuryFee        *num.Uint
	BuyBackFee         *num.Uint
	HighVolumeMakerFee *num.Uint

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
		BuyBackFee:                        num.UintToString(f.BuyBackFee),
		TreasuryFee:                       num.UintToString(f.TreasuryFee),
		HighVolumeMakerFee:                num.UintToString(f.HighVolumeMakerFee),
		MakerFeeVolumeDiscount:            num.UintToString(f.MakerFeeVolumeDiscount),
		InfrastructureFeeVolumeDiscount:   num.UintToString(f.InfrastructureFeeVolumeDiscount),
		LiquidityFeeVolumeDiscount:        num.UintToString(f.LiquidityFeeVolumeDiscount),
		MakerFeeReferrerDiscount:          num.UintToString(f.MakerFeeReferrerDiscount),
		InfrastructureFeeReferrerDiscount: num.UintToString(f.InfrastructureFeeReferrerDiscount),
		LiquidityFeeReferrerDiscount:      num.UintToString(f.LiquidityFeeReferrerDiscount),
	}

	return fee
}

func FeeFromProto(f *proto.Fee) *Fee {
	bbf := num.UintZero()
	if len(f.BuyBackFee) > 0 {
		bbf = num.MustUintFromString(f.BuyBackFee, 10)
	}
	tf := num.UintZero()
	if len(f.TreasuryFee) > 0 {
		tf = num.MustUintFromString(f.TreasuryFee, 10)
	}
	hvmf := num.UintZero()
	if len(f.HighVolumeMakerFee) > 0 {
		hvmf = num.MustUintFromString(f.HighVolumeMakerFee, 10)
	}

	return &Fee{
		MakerFee:                          num.MustUintFromString(f.MakerFee, 10),
		HighVolumeMakerFee:                hvmf,
		InfrastructureFee:                 num.MustUintFromString(f.InfrastructureFee, 10),
		LiquidityFee:                      num.MustUintFromString(f.LiquidityFee, 10),
		BuyBackFee:                        bbf,
		TreasuryFee:                       tf,
		MakerFeeVolumeDiscount:            num.MustUintFromString(f.MakerFeeVolumeDiscount, 10),
		InfrastructureFeeVolumeDiscount:   num.MustUintFromString(f.InfrastructureFeeVolumeDiscount, 10),
		LiquidityFeeVolumeDiscount:        num.MustUintFromString(f.LiquidityFeeVolumeDiscount, 10),
		MakerFeeReferrerDiscount:          num.MustUintFromString(f.MakerFeeReferrerDiscount, 10),
		InfrastructureFeeReferrerDiscount: num.MustUintFromString(f.InfrastructureFeeReferrerDiscount, 10),
		LiquidityFeeReferrerDiscount:      num.MustUintFromString(f.LiquidityFeeReferrerDiscount, 10),
	}
}

func (f Fee) Clone() *Fee {
	fee := &Fee{
		MakerFee:           f.MakerFee.Clone(),
		InfrastructureFee:  f.InfrastructureFee.Clone(),
		LiquidityFee:       f.LiquidityFee.Clone(),
		BuyBackFee:         f.BuyBackFee.Clone(),
		TreasuryFee:        f.TreasuryFee.Clone(),
		HighVolumeMakerFee: f.HighVolumeMakerFee.Clone(),
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
		"makerFee(%s) liquidityFee(%s) infrastructureFee(%s) buyBackFee(%s) treasuryFee(%s) highVolumeMakerFee(%s) makerFeeReferrerDiscount(%s) liquidityFeeReferrerDiscount(%s) infrastructureFeeReferrerDiscount(%s) makerFeeVolumeDiscount(%s) liquidityFeeVolumeDiscount(%s) infrastructureFeeVolumeDiscount(%s)",
		stringer.PtrToString(f.MakerFee),
		stringer.PtrToString(f.LiquidityFee),
		stringer.PtrToString(f.InfrastructureFee),
		stringer.PtrToString(f.BuyBackFee),
		stringer.PtrToString(f.TreasuryFee),
		stringer.PtrToString(f.HighVolumeMakerFee),
		stringer.PtrToString(f.MakerFeeReferrerDiscount),
		stringer.PtrToString(f.LiquidityFeeReferrerDiscount),
		stringer.PtrToString(f.InfrastructureFeeReferrerDiscount),
		stringer.PtrToString(f.MakerFeeVolumeDiscount),
		stringer.PtrToString(f.LiquidityFeeVolumeDiscount),
		stringer.PtrToString(f.InfrastructureFeeVolumeDiscount),
	)
}

// NewFee returns a new fee object, with all fields initialised.
func NewFee() *Fee {
	return &Fee{
		MakerFee:           num.UintZero(),
		InfrastructureFee:  num.UintZero(),
		LiquidityFee:       num.UintZero(),
		BuyBackFee:         num.UintZero(),
		TreasuryFee:        num.UintZero(),
		HighVolumeMakerFee: num.UintZero(),
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
		stringer.PtrToString(rf.MakerFeeReferrerReward),
		stringer.PtrToString(rf.LiquidityFeeReferrerReward),
		stringer.PtrToString(rf.InfrastructureFeeReferrerReward),
	)
}

func NewReferrerReward() *ReferrerReward {
	return &ReferrerReward{
		MakerFeeReferrerReward:          num.UintZero(),
		InfrastructureFeeReferrerReward: num.UintZero(),
		LiquidityFeeReferrerReward:      num.UintZero(),
	}
}
