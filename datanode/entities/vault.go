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

package entities

import (
	"encoding/json"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/shopspring/decimal"
)

type _Vault struct{}

type VaultID = ID[_Vault]

type VaultPartyShare struct {
	VaultID  VaultID
	PartyID  PartyID
	Share    decimal.Decimal
	VegaTime time.Time
}

type VaultState struct {
	VaultID            VaultID
	Vault              *vega.Vault
	PartyShares        []*VaultPartyShare
	InvestedAmount     num.Decimal
	Status             VaultStatus
	NextFeeCalc        time.Time
	NextRedemptionDate time.Time
	VegaTime           time.Time
}

func VaultStateFromProto(vs *eventspb.VaultState, vegaTime time.Time) (*VaultState, error) {
	ps := make([]*VaultPartyShare, 0, len(vs.PartyShares))
	for _, partyShare := range vs.PartyShares {
		share, err := num.DecimalFromString(partyShare.Share)
		if err != nil {
			return nil, err
		}
		ps = append(ps, &VaultPartyShare{
			VaultID: (ID[_Vault])(vs.Vault.VaultId),
			PartyID: (ID[_Party])(partyShare.Party),
			Share:   share,
		})
	}
	vaultState := &VaultState{
		VaultID:            ID[_Vault](vs.Vault.VaultId),
		Vault:              vs.Vault,
		InvestedAmount:     num.MustDecimalFromString(vs.InvestedAmount),
		Status:             VaultStatus(vs.Status),
		NextFeeCalc:        time.Unix(0, vs.NextFeeCalc),
		NextRedemptionDate: time.Unix(0, vs.NextRedemptionDate),
		PartyShares:        ps,
	}

	return vaultState, nil
}

func (vs VaultState) ToProto() *eventspb.VaultState {
	partyShares := make([]*eventspb.VaultShareHolder, 0, len(vs.PartyShares))
	for _, ps := range vs.PartyShares {
		partyShares = append(partyShares, &eventspb.VaultShareHolder{
			Party: ps.PartyID.String(),
			Share: ps.Share.String(),
		})
	}
	return &eventspb.VaultState{
		Vault:              vs.Vault,
		InvestedAmount:     vs.InvestedAmount.String(),
		NextRedemptionDate: vs.NextRedemptionDate.UnixNano(),
		Status:             vega.VaultStatus(vs.Status),
		NextFeeCalc:        vs.NextFeeCalc.UnixNano(),
		PartyShares:        partyShares,
	}
}

func (vs VaultState) Cursor() *Cursor {
	cursor := VaultCursor{
		VaultID: vs.VaultID.String(),
	}
	return NewCursor(cursor.String())
}

func (vs VaultState) ToProtoEdge(_ ...any) (*v2.VaultEdge, error) {
	return &v2.VaultEdge{
		Node:   vs.ToProto(),
		Cursor: vs.Cursor().Encode(),
	}, nil
}

type VaultCursor struct {
	VaultID string `json:"vault_id"`
}

func (vc VaultCursor) String() string {
	bs, err := json.Marshal(vc)
	if err != nil {
		panic(fmt.Errorf("marshalling vault cursor: %w", err))
	}
	return string(bs)
}

func (vc *VaultCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), vc)
}
