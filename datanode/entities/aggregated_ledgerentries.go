package entities

import (
	"encoding/json"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/shopspring/decimal"
)

// AggregatedLedgerEntries represents the the summed amount of ledger entries for a set of accounts within a given time range.
// VegaTime and Quantity will always be set. The others will be nil unless when
// querying grouping by one of the corresponding fields is requested.
type AggregatedLedgerEntries struct {
	VegaTime     time.Time
	Quantity     decimal.Decimal
	TransferType *LedgerMovementType
	PartyID      *PartyID
	AssetID      *AssetID
	MarketID     *MarketID
	AccountType  *types.AccountType
}

func (ledgerEntries *AggregatedLedgerEntries) ToProto() *v2.AggregatedLedgerEntries {
	lep := &v2.AggregatedLedgerEntries{}

	partyIDString := ledgerEntries.PartyID.String()
	if partyIDString != "" {
		lep.PartyId = &partyIDString
	}

	assetIDString := ledgerEntries.AssetID.String()
	if assetIDString != "" {
		lep.AssetId = &assetIDString
	}

	marketIDString := ledgerEntries.MarketID.String()
	if marketIDString != "" {
		lep.MarketId = &marketIDString
	}

	if ledgerEntries.AccountType != nil {
		lep.AccountType = *ledgerEntries.AccountType
	}

	if ledgerEntries.TransferType != nil {
		lep.TransferType = vega.TransferType(*ledgerEntries.TransferType)
	}

	return lep
}

func (ledgerEntries AggregatedLedgerEntries) Cursor() *Cursor {
	return NewCursor(AggregatedLedgerEntriesCursor{
		VegaTime: ledgerEntries.VegaTime,
	}.String())
}

func (ledgerEntries AggregatedLedgerEntries) ToProtoEdge(_ ...any) (*v2.AggregatedLedgerEntriesEdge, error) {
	return &v2.AggregatedLedgerEntriesEdge{
		Node:   ledgerEntries.ToProto(),
		Cursor: ledgerEntries.Cursor().Encode(),
	}, nil
}

type AggregatedLedgerEntriesCursor struct {
	VegaTime time.Time `json:"vega_time"`
}

func (c AggregatedLedgerEntriesCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal aggregate ledger entries cursor: %w", err))
	}
	return string(bs)
}

func (c *AggregatedLedgerEntriesCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}
