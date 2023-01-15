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

// AggregatedLedgerEntry represents the the summed amount of ledger entries for a set of accounts within a given time range.
// VegaTime and Quantity will always be set. The others will be nil unless when
// querying grouping by one of the corresponding fields is requested.
type AggregatedLedgerEntry struct {
	VegaTime     time.Time
	Quantity     decimal.Decimal
	TransferType *LedgerMovementType
	AssetID      *AssetID

	FromAccountPartyID  *PartyID
	ToAccountPartyID    *PartyID
	FromAccountMarketID *MarketID
	ToAccountMarketID   *MarketID
	FromAccountType     *types.AccountType
	ToAccountType       *types.AccountType
	FromAccountBalance  decimal.Decimal
	ToAccountBalance    decimal.Decimal
}

func (ledgerEntries *AggregatedLedgerEntry) ToProto() *v2.AggregatedLedgerEntry {
	lep := &v2.AggregatedLedgerEntry{}

	lep.Quantity = ledgerEntries.Quantity.String()
	lep.Timestamp = ledgerEntries.VegaTime.UnixNano()

	if ledgerEntries.TransferType != nil {
		lep.TransferType = vega.TransferType(*ledgerEntries.TransferType)
	}

	if ledgerEntries.AssetID != nil {
		assetIDString := ledgerEntries.AssetID.String()
		if assetIDString != "" {
			lep.AssetId = &assetIDString
		}
	}

	if ledgerEntries.FromAccountPartyID != nil {
		partyIDString := ledgerEntries.FromAccountPartyID.String()
		if partyIDString != "" {
			lep.FromAccountPartyId = &partyIDString
		}
	}

	if ledgerEntries.ToAccountPartyID != nil {
		partyIDString := ledgerEntries.ToAccountPartyID.String()
		if partyIDString != "" {
			lep.ToAccountPartyId = &partyIDString
		}
	}

	if ledgerEntries.FromAccountMarketID != nil {
		marketIDString := ledgerEntries.FromAccountMarketID.String()
		if marketIDString != "" {
			lep.FromAccountMarketId = &marketIDString
		}
	}

	if ledgerEntries.ToAccountMarketID != nil {
		marketIDString := ledgerEntries.ToAccountMarketID.String()
		if marketIDString != "" {
			lep.ToAccountMarketId = &marketIDString
		}
	}

	if ledgerEntries.FromAccountType != nil {
		lep.FromAccountType = *ledgerEntries.FromAccountType
	}

	if ledgerEntries.ToAccountType != nil {
		lep.ToAccountType = *ledgerEntries.ToAccountType
	}

	lep.FromAccountBalance = ledgerEntries.FromAccountBalance.String()
	lep.ToAccountBalance = ledgerEntries.ToAccountBalance.String()

	return lep
}

func (ledgerEntries AggregatedLedgerEntry) Cursor() *Cursor {
	return NewCursor(AggregatedLedgerEntriesCursor{
		VegaTime: ledgerEntries.VegaTime,
	}.String())
}

func (ledgerEntries AggregatedLedgerEntry) ToProtoEdge(_ ...any) (*v2.AggregatedLedgerEntriesEdge, error) {
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
