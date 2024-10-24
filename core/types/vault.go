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
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
)

type Vault struct {
	ID                   string
	Owner                string
	Asset                string
	MetaData             *vega.VaultMetaData
	FeePeriod            time.Duration
	ManagementFeeFactor  num.Decimal
	PerformanceFeeFactor num.Decimal
	CutOffPeriodLength   int64
	RedemptionDates      []*RedemptionDate
}

func (v *Vault) IntoProto() *vega.Vault {
	redemptionDates := make([]*vega.RedemptionDate, 0, len(v.RedemptionDates))
	for _, rd := range v.RedemptionDates {
		redemptionDates = append(redemptionDates, &vega.RedemptionDate{
			RedemptionDate: rd.RedemptionDate.Unix(),
			RedemptionType: rd.RedemptionType,
			MaxFraction:    rd.MaxFraction.String(),
		})
	}
	return &vega.Vault{
		VaultId:              v.ID,
		Asset:                v.Asset,
		Owner:                v.Owner,
		VaultMetadata:        v.MetaData,
		FeePeriod:            v.FeePeriod.String(),
		ManagementFeeFactor:  v.ManagementFeeFactor.String(),
		PerformanceFeeFactor: v.PerformanceFeeFactor.String(),
		RedemptionDates:      redemptionDates,
		CutOffPeriodLength:   v.CutOffPeriodLength,
	}
}

func VaultFromProto(v *vega.Vault) *Vault {
	feePeriod, _ := time.ParseDuration(v.FeePeriod)
	managementFeeFactor, _ := num.DecimalFromString(v.ManagementFeeFactor)
	performanceFeeFactor, _ := num.DecimalFromString(v.PerformanceFeeFactor)
	redemptionDates := make([]*RedemptionDate, 0, len(v.RedemptionDates))
	for _, rd := range v.RedemptionDates {
		redemptionDates = append(redemptionDates, &RedemptionDate{
			RedemptionType: rd.RedemptionType,
			RedemptionDate: time.Unix(rd.RedemptionDate, 0),
			MaxFraction:    num.MustDecimalFromString(rd.MaxFraction),
		})
	}
	return &Vault{
		ID:                   v.VaultId,
		Owner:                v.Owner,
		Asset:                v.Asset,
		MetaData:             v.VaultMetadata,
		FeePeriod:            feePeriod,
		ManagementFeeFactor:  managementFeeFactor,
		PerformanceFeeFactor: performanceFeeFactor,
		CutOffPeriodLength:   v.CutOffPeriodLength,
		RedemptionDates:      redemptionDates,
	}
}

type RedemptionType = vega.RedemptionType

const (
	// Default value.
	RedemptionTypeUnspecified RedemptionType = vega.RedemptionType_REDEMPTION_TYPE_UNSPECIFIED
	// Consider only general account balance.
	RedemptionTypeFreeCashOnly RedemptionType = vega.RedemptionType_REDEMPTION_TYPE_FREE_CASH_ONLY
	// Consider all vault accounts balance.
	RedemptionTypeNormal RedemptionType = vega.RedemptionType_REDEMPTION_TYPE_NORMAL
)

type RedemptionDate struct {
	RedemptionType RedemptionType
	RedemptionDate time.Time
	MaxFraction    num.Decimal
}

type VaultStatus = vega.VaultStatus

const (
	VaultStatusUnspecified VaultStatus = vega.VaultStatus_VAULT_STATUS_UNSPECIFIED
	VaultStatusActive      VaultStatus = vega.VaultStatus_VAULT_STATUS_ACTIVE
	VaultStatusStopping    VaultStatus = vega.VaultStatus_VAULT_STATUS_STOPPING
	VaultStatusStopped     VaultStatus = vega.VaultStatus_VAULT_STATUS_STOPPED
)

type RedeemStatus = vega.RedeemStatus

const (
	RedeemStatusUnspecified RedeemStatus = vega.RedeemStatus_REDEEM_STATUS_UNSPECIFIED
	RedeemStatusPending     RedeemStatus = vega.RedeemStatus_REDEEM_STATUS_PENDING
	RedeemStatusLate        RedeemStatus = vega.RedeemStatus_REDEEM_STATUS_LATE
	RedeemStatusCompleted   RedeemStatus = vega.RedeemStatus_REDEEM_STATUS_COMPLETED
)
