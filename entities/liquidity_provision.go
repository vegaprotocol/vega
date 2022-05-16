package entities

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/protos/vega"
	"github.com/jackc/pgtype"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/encoding/protojson"
)

type LiquidityProvisionID struct {
	ID
}

func NewLiquidityProvisionID(id string) LiquidityProvisionID {
	return LiquidityProvisionID{
		ID: ID(id),
	}
}

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
	VegaTime         time.Time
}

func LiquidityProvisionFromProto(lpProto *vega.LiquidityProvision, vegaTime time.Time) (LiquidityProvision, error) {
	lpID := NewLiquidityProvisionID(lpProto.Id)
	partyID := NewPartyID(lpProto.PartyId)
	marketID := NewMarketID(lpProto.MarketId)

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
		CreatedAt:        time.Unix(0, lpProto.CreatedAt),
		UpdatedAt:        time.Unix(0, lpProto.UpdatedAt),
		MarketID:         marketID,
		CommitmentAmount: commitmentAmount,
		Fee:              fee,
		Sells:            sells,
		Buys:             buys,
		Version:          int64(lpProto.Version),
		Status:           LiquidityProvisionStatus(lpProto.Status),
		Reference:        lpProto.Reference,
		VegaTime:         vegaTime,
	}, nil
}

func (lp *LiquidityProvision) ToProto() *vega.LiquidityProvision {
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
	"status", "reference", "vega_time"}

func (lp LiquidityProvision) ToRow() []interface{} {
	return []interface{}{
		lp.ID, lp.PartyID, lp.CreatedAt, lp.UpdatedAt, lp.MarketID,
		lp.CommitmentAmount, lp.Fee, lp.Sells, lp.Buys, lp.Version,
		lp.Status, lp.Reference, lp.VegaTime}
}
