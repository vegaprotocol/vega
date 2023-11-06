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

// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities

import (
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/jackc/pgtype"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/encoding/protojson"
)

type _LiquidityProvision struct{}

type LiquidityProvisionID = ID[_LiquidityProvision]

type LiquidityOrderReference struct {
	*vega.LiquidityOrderReference
}

func (l LiquidityOrderReference) EncodeBinary(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	protoBytes, err := protojson.Marshal(l.LiquidityOrderReference)
	if err != nil {
		return buf, fmt.Errorf("failed to marshal LiquidityOrderReference: %w", err)
	}
	return append(buf, protoBytes...), nil
}

func (l *LiquidityOrderReference) DecodeBinary(_ *pgtype.ConnInfo, src []byte) error {
	return protojson.Unmarshal(src, l)
}

type LiquidityProvision struct {
	ID               LiquidityProvisionID
	PartyID          PartyID
	CreatedAt        time.Time
	UpdatedAt        time.Time
	MarketID         MarketID
	CommitmentAmount decimal.Decimal
	Fee              decimal.Decimal
	Sells            []LiquidityOrderReference
	Buys             []LiquidityOrderReference
	Version          int64
	Status           LiquidityProvisionStatus
	Reference        string
	TxHash           TxHash
	VegaTime         time.Time
}

type CurrentAndPreviousLiquidityProvisions struct {
	ID                       LiquidityProvisionID
	PartyID                  PartyID
	CreatedAt                time.Time
	UpdatedAt                time.Time
	MarketID                 MarketID
	CommitmentAmount         decimal.Decimal
	Fee                      decimal.Decimal
	Sells                    []LiquidityOrderReference
	Buys                     []LiquidityOrderReference
	Version                  int64
	Status                   LiquidityProvisionStatus
	Reference                string
	TxHash                   TxHash
	VegaTime                 time.Time
	PreviousID               LiquidityProvisionID
	PreviousPartyID          PartyID
	PreviousCreatedAt        *time.Time
	PreviousUpdatedAt        *time.Time
	PreviousMarketID         MarketID
	PreviousCommitmentAmount *decimal.Decimal
	PreviousFee              *decimal.Decimal
	PreviousSells            []LiquidityOrderReference
	PreviousBuys             []LiquidityOrderReference
	PreviousVersion          *int64
	PreviousStatus           *LiquidityProvisionStatus
	PreviousReference        *string
	PreviousTxHash           TxHash
	PreviousVegaTime         *time.Time
}

func LiquidityProvisionFromProto(lpProto *vega.LiquidityProvision, txHash TxHash, vegaTime time.Time) (LiquidityProvision, error) {
	lpID := LiquidityProvisionID(lpProto.Id)
	partyID := PartyID(lpProto.PartyId)
	marketID := MarketID(lpProto.MarketId)

	commitmentAmount, err := decimal.NewFromString(lpProto.CommitmentAmount)
	if err != nil {
		return LiquidityProvision{}, fmt.Errorf("liquidity provision has invalid commitement amount: %w", err)
	}

	fee, err := decimal.NewFromString(lpProto.Fee)
	if err != nil {
		return LiquidityProvision{}, fmt.Errorf("liquidity provision has invalid fee amount: %w", err)
	}

	sells := make([]LiquidityOrderReference, 0, len(lpProto.Sells))
	buys := make([]LiquidityOrderReference, 0, len(lpProto.Buys))

	for _, sell := range lpProto.Sells {
		sells = append(sells, LiquidityOrderReference{sell})
	}

	for _, buy := range lpProto.Buys {
		buys = append(buys, LiquidityOrderReference{buy})
	}

	return LiquidityProvision{
		ID:               lpID,
		PartyID:          partyID,
		CreatedAt:        NanosToPostgresTimestamp(lpProto.CreatedAt),
		UpdatedAt:        NanosToPostgresTimestamp(lpProto.UpdatedAt),
		MarketID:         marketID,
		CommitmentAmount: commitmentAmount,
		Fee:              fee,
		Sells:            sells,
		Buys:             buys,
		Version:          int64(lpProto.Version),
		Status:           LiquidityProvisionStatus(lpProto.Status),
		Reference:        lpProto.Reference,
		TxHash:           txHash,
		VegaTime:         vegaTime,
	}, nil
}

func (lp LiquidityProvision) ToProto() *vega.LiquidityProvision {
	sells := make([]*vega.LiquidityOrderReference, 0, len(lp.Sells))
	buys := make([]*vega.LiquidityOrderReference, 0, len(lp.Buys))

	for _, sell := range lp.Sells {
		sells = append(sells, sell.LiquidityOrderReference)
	}
	for _, buy := range lp.Buys {
		buys = append(buys, buy.LiquidityOrderReference)
	}

	return &vega.LiquidityProvision{
		Id:               lp.ID.String(),
		PartyId:          lp.PartyID.String(),
		CreatedAt:        lp.CreatedAt.UnixNano(),
		UpdatedAt:        lp.UpdatedAt.UnixNano(),
		MarketId:         lp.MarketID.String(),
		CommitmentAmount: lp.CommitmentAmount.String(),
		Fee:              lp.Fee.String(),
		Sells:            sells,
		Buys:             buys,
		Version:          uint64(lp.Version),
		Status:           vega.LiquidityProvision_Status(lp.Status),
		Reference:        lp.Reference,
	}
}

func (lp CurrentAndPreviousLiquidityProvisions) ToProto() *v2.LiquidityProvision {
	sells := make([]*vega.LiquidityOrderReference, 0)
	buys := make([]*vega.LiquidityOrderReference, 0)

	for _, sell := range lp.Sells {
		sells = append(sells, sell.LiquidityOrderReference)
	}
	for _, buy := range lp.Buys {
		buys = append(buys, buy.LiquidityOrderReference)
	}

	if lp.Status == LiquidityProvisionStatusPending && (lp.PreviousStatus != nil && *lp.PreviousStatus == LiquidityProvisionStatusActive) {
		// check to see if the previous state is active, if so, then that is still the active state,
		// and the current state is pending
		return lp.currentWithPendingLP(sells, buys)
	}

	return &v2.LiquidityProvision{
		Current: &vega.LiquidityProvision{
			Id:               lp.ID.String(),
			PartyId:          lp.PartyID.String(),
			CreatedAt:        lp.CreatedAt.UnixNano(),
			UpdatedAt:        lp.UpdatedAt.UnixNano(),
			MarketId:         lp.MarketID.String(),
			CommitmentAmount: lp.CommitmentAmount.String(),
			Fee:              lp.Fee.String(),
			Sells:            sells,
			Buys:             buys,
			Version:          uint64(lp.Version),
			Status:           vega.LiquidityProvision_Status(lp.Status),
			Reference:        lp.Reference,
		},
	}
}

func (lp CurrentAndPreviousLiquidityProvisions) currentWithPendingLP(sells, buys []*vega.LiquidityOrderReference) *v2.LiquidityProvision {
	previousSells := make([]*vega.LiquidityOrderReference, 0)
	previousBuys := make([]*vega.LiquidityOrderReference, 0)

	if lp.PreviousSells != nil {
		for _, sell := range lp.PreviousSells {
			previousSells = append(previousSells, sell.LiquidityOrderReference)
		}
	}
	if lp.PreviousBuys != nil {
		for _, buy := range lp.PreviousBuys {
			previousBuys = append(previousBuys, buy.LiquidityOrderReference)
		}
	}
	pending := vega.LiquidityProvision{
		Id:               lp.ID.String(),
		PartyId:          lp.PartyID.String(),
		CreatedAt:        lp.CreatedAt.UnixNano(),
		UpdatedAt:        lp.UpdatedAt.UnixNano(),
		MarketId:         lp.MarketID.String(),
		CommitmentAmount: lp.CommitmentAmount.String(),
		Fee:              lp.Fee.String(),
		Sells:            sells,
		Buys:             buys,
		Version:          uint64(lp.Version),
		Status:           vega.LiquidityProvision_Status(lp.Status),
		Reference:        lp.Reference,
	}

	return &v2.LiquidityProvision{
		Current: &vega.LiquidityProvision{
			Id:               (lp.PreviousID).String(),
			PartyId:          (lp.PreviousPartyID).String(),
			CreatedAt:        (lp.PreviousCreatedAt).UnixNano(),
			UpdatedAt:        (lp.PreviousUpdatedAt).UnixNano(),
			MarketId:         (lp.PreviousMarketID).String(),
			CommitmentAmount: (lp.PreviousCommitmentAmount).String(),
			Fee:              (lp.PreviousFee).String(),
			Sells:            previousSells,
			Buys:             previousBuys,
			Version:          uint64(*lp.PreviousVersion),
			Status:           vega.LiquidityProvision_Status(*lp.PreviousStatus),
			Reference:        *lp.PreviousReference,
		},
		Pending: &pending,
	}
}

type LiquidityProvisionKey struct {
	ID       LiquidityProvisionID
	VegaTime time.Time
}

func (lp LiquidityProvision) Key() LiquidityProvisionKey {
	return LiquidityProvisionKey{lp.ID, lp.VegaTime}
}

var LiquidityProvisionColumns = []string{
	"id", "party_id", "created_at", "updated_at", "market_id",
	"commitment_amount", "fee", "sells", "buys", "version",
	"status", "reference", "tx_hash", "vega_time",
}

func (lp LiquidityProvision) ToRow() []interface{} {
	return []interface{}{
		lp.ID, lp.PartyID, lp.CreatedAt, lp.UpdatedAt, lp.MarketID,
		lp.CommitmentAmount, lp.Fee, lp.Sells, lp.Buys, lp.Version,
		lp.Status, lp.Reference, lp.TxHash, lp.VegaTime,
	}
}

func (lp LiquidityProvision) Cursor() *Cursor {
	lc := LiquidityProvisionCursor{
		VegaTime: lp.VegaTime,
		ID:       lp.ID,
	}
	return NewCursor(lc.String())
}

func (lp CurrentAndPreviousLiquidityProvisions) Cursor() *Cursor {
	lc := LiquidityProvisionCursor{
		VegaTime: lp.VegaTime,
		ID:       lp.ID,
	}
	return NewCursor(lc.String())
}

func (lp LiquidityProvision) ToProtoEdge(_ ...any) (*v2.LiquidityProvisionsEdge, error) {
	return &v2.LiquidityProvisionsEdge{
		Node:   lp.ToProto(),
		Cursor: lp.Cursor().Encode(),
	}, nil
}

func (lp CurrentAndPreviousLiquidityProvisions) ToProtoEdge(_ ...any) (*v2.LiquidityProvisionWithPendingEdge, error) {
	return &v2.LiquidityProvisionWithPendingEdge{
		Node:   lp.ToProto(),
		Cursor: lp.Cursor().Encode(),
	}, nil
}

type LiquidityProvisionCursor struct {
	VegaTime time.Time            `json:"vegaTime"`
	ID       LiquidityProvisionID `json:"id"`
}

func (lc LiquidityProvisionCursor) String() string {
	bs, err := json.Marshal(lc)
	if err != nil {
		panic(fmt.Errorf("could not marshal liquidity provision cursor: %w", err))
	}
	return string(bs)
}

func (lc *LiquidityProvisionCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), lc)
}
