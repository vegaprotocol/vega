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

import "google.golang.org/protobuf/proto"

type Entities interface {
	Market | Party | Trade | Order | MarketData | Reward | Candle | Deposit |
		Withdrawal | Asset | OracleSpec | OracleData | Position | LiquidityProvision | Vote |
		AccountBalance | Proposal | Delegation | Node | NetworkParameter | Checkpoint |
		StakeLinking | NodeSignature | KeyRotation | ERC20MultiSigSignerAddedEvent |
		ERC20MultiSigSignerRemovedEvent | EthereumKeyRotation | AggregatedBalance | AggregatedLedgerEntry |
		ProtocolUpgradeProposal | CoreSnapshotData | EpochRewardSummary
}

type PagedEntity[T proto.Message] interface {
	Entities | Transfer | MarginLevels

	// ToProtoEdge may need some optional arguments in order to generate the proto, for example margin levels
	// requires an account source. This is not ideal, but we can come back to this later if a better solution can be found.
	ToProtoEdge(...any) (T, error)
	Cursor() *Cursor
}

type ProtoEntity[T proto.Message] interface {
	Entities | Account | NodeBasic
	ToProto() T
}

func PageEntities[T proto.Message, U PagedEntity[T]](items []U, pagination CursorPagination) ([]U, PageInfo) {
	var pagedItems []U
	var limit int
	var pageInfo PageInfo

	if len(items) == 0 {
		return pagedItems, pageInfo
	}

	if pagination.HasForward() && pagination.Forward.Limit != nil {
		limit = int(*pagination.Forward.Limit)
		switch len(items) {
		case limit + 2:
			pagedItems = items[1 : limit+1]
			pageInfo.HasNextPage = true
			pageInfo.HasPreviousPage = true
		case limit + 1:
			if !pagination.Forward.HasCursor() {
				pagedItems = items[0:limit]
				pageInfo.HasNextPage = true
				pageInfo.HasPreviousPage = false
			} else {
				pagedItems = items[1:]
				pageInfo.HasNextPage = false
				pageInfo.HasPreviousPage = true
			}
		default:
			// if the pagination for the first item is the same as the after pagination, then we have a previous page, and we shouldn't include it
			if pagination.HasForward() && pagination.Forward.HasCursor() && pagination.Forward.Cursor.Value() == items[0].Cursor().Value() {
				pagedItems = items[1:]
				pageInfo.HasNextPage = false
				pageInfo.HasPreviousPage = true
			} else {
				pagedItems = items
				pageInfo.HasNextPage = false
				pageInfo.HasPreviousPage = false
			}
		}
	} else if pagination.HasBackward() && pagination.Backward.Limit != nil {
		limit = int(*pagination.Backward.Limit)
		switch len(items) {
		case limit + 2:
			pagedItems = ReverseSlice(items[1 : limit+1])
			pageInfo.HasNextPage = true
			pageInfo.HasPreviousPage = true
		case limit + 1:
			if !pagination.Backward.HasCursor() {
				pagedItems = ReverseSlice(items[0:limit])
				pageInfo.HasNextPage = false
				pageInfo.HasPreviousPage = true
			} else {
				pagedItems = ReverseSlice(items[1:])
				pageInfo.HasNextPage = true
				pageInfo.HasPreviousPage = false
			}
		default:
			if pagination.HasBackward() && pagination.Backward.HasCursor() && pagination.Backward.Cursor.Value() == items[0].Cursor().Value() {
				pagedItems = ReverseSlice(items[1:])
				pageInfo.HasNextPage = true
				pageInfo.HasPreviousPage = false
			} else {
				pagedItems = ReverseSlice(items)
				pageInfo.HasNextPage = false
				pageInfo.HasPreviousPage = false
			}
		}
	} else {
		pagedItems = items
		pageInfo.HasNextPage = false
		pageInfo.HasPreviousPage = false
	}

	if len(pagedItems) > 0 {
		startCursor := pagedItems[0].Cursor()
		endCursor := pagedItems[len(pagedItems)-1].Cursor()
		pageInfo.StartCursor = startCursor.Encode()
		pageInfo.EndCursor = endCursor.Encode()
	}

	return pagedItems, pageInfo
}

func ReverseSlice[T any](input []T) (reversed []T) {
	reversed = make([]T, len(input))
	copy(reversed, input)
	for i, j := 0, len(input)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = input[j], input[i]
	}
	return
}
