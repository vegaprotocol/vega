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
)

type _RedemptionRequestID struct{}

type RedemptionRequestID = ID[_RedemptionRequestID]

type RedemptionRequest struct {
	RequestID       RedemptionRequestID
	VaultID         VaultID
	PartyID         PartyID
	Asset           AssetID
	RequestedAmount num.Decimal
	RemainingAmount num.Decimal
	EligibilityDate time.Time
	LastUpdated     time.Time
	Status          RedeemStatus
	VegaTime        time.Time
}

func RedemptionRequestFromProto(request *eventspb.RedemptionRequest, vegaTime time.Time) (*RedemptionRequest, error) {
	return &RedemptionRequest{
		RequestID:       ID[_RedemptionRequestID](request.VaultId),
		PartyID:         ID[_Party](request.PartyId),
		VaultID:         ID[_Vault](request.VaultId),
		Asset:           ID[_Asset](request.Asset),
		RequestedAmount: num.MustDecimalFromString(request.RequestedAmount),
		RemainingAmount: num.MustDecimalFromString(request.RemainingAmount),
		Status:          RedeemStatus(request.Status),
		EligibilityDate: time.Unix(0, request.Date),
		LastUpdated:     time.Unix(0, request.LastUpdate),
		VegaTime:        vegaTime,
	}, nil
}

func (vs RedemptionRequest) ToProto() *eventspb.RedemptionRequest {
	return &eventspb.RedemptionRequest{
		RequestId:       vs.RequestID.String(),
		VaultId:         vs.VaultID.String(),
		PartyId:         vs.PartyID.String(),
		Asset:           vs.Asset.String(),
		Date:            vs.EligibilityDate.UnixNano(),
		LastUpdate:      vs.LastUpdated.UnixNano(),
		RequestedAmount: vs.RequestedAmount.String(),
		RemainingAmount: vs.RemainingAmount.String(),
		Status:          vega.RedeemStatus(vs.Status),
	}
}

func (vs RedemptionRequest) Cursor() *Cursor {
	cursor := RedemptionRequestCursor{
		RequestID: vs.RequestID.String(),
	}
	return NewCursor(cursor.String())
}

func (rr RedemptionRequest) ToProtoEdge(_ ...any) (*v2.RedemptionRequestEdge, error) {
	return &v2.RedemptionRequestEdge{
		Node:   rr.ToProto(),
		Cursor: rr.Cursor().Encode(),
	}, nil
}

type RedemptionRequestCursor struct {
	RequestID string `json:"request_id"`
}

func (rrc RedemptionRequestCursor) String() string {
	bs, err := json.Marshal(rrc)
	if err != nil {
		panic(fmt.Errorf("marshalling vault redemption request cursor: %w", err))
	}
	return string(bs)
}

func (rrc *RedemptionRequestCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), rrc)
}
