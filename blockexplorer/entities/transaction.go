// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tmTypes "github.com/cometbft/cometbft/abci/types"
	"google.golang.org/protobuf/proto"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/ptr"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	pb "code.vegaprotocol.io/vega/protos/blockexplorer"
)

type TxResultRow struct {
	RowID     int64     `db:"rowid"`
	BlockID   int64     `db:"block_id"`
	Index     int64     `db:"index"`
	CreatedAt time.Time `db:"created_at"`
	TxHash    string    `db:"tx_hash"`
	TxResult  []byte    `db:"tx_result"`
	Submitter string    `db:"submitter"`
	CmdType   string    `db:"cmd_type"`
}

func (t *TxResultRow) ToProto() (*pb.Transaction, error) {
	txResult := tmTypes.TxResult{}
	if err := txResult.Unmarshal(t.TxResult); err != nil {
		return nil, fmt.Errorf("unmarshalling tendermint tx result: %w", err)
	}

	cTx := commandspb.Transaction{}
	if err := proto.Unmarshal(txResult.Tx, &cTx); err != nil {
		return nil, fmt.Errorf("unmarshalling vega transaction: %w", err)
	}

	inputData, err := commands.UnmarshalInputData(cTx.InputData)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling vega input data: %w", err)
	}

	cursor := t.Cursor()

	var strErr *string
	if txResult.Result.Code != 0 {
		strErr = ptr.From(string(txResult.Result.Data))
	}

	return &pb.Transaction{
		Block:     uint64(t.BlockID),
		Index:     uint32(t.Index),
		Type:      extractAttribute(&txResult, "command", "type"),
		Submitter: extractAttribute(&txResult, "tx", "submitter"),
		Code:      txResult.Result.Code,
		Error:     strErr,
		Hash:      t.TxHash,
		Cursor:    cursor.String(),
		Command:   inputData,
		Signature: cTx.Signature,
	}, nil
}

func (t *TxResultRow) Cursor() TxCursor {
	return TxCursor{
		BlockNumber: uint64(t.BlockID),
		TxIndex:     uint32(t.Index),
	}
}

func extractAttribute(r *tmTypes.TxResult, eType, key string) string {
	for _, e := range r.Result.Events {
		if e.Type == eType {
			for _, a := range e.Attributes {
				if string(a.Key) == key {
					return string(a.Value)
				}
			}
		}
	}
	return ""
}

type TxCursor struct {
	BlockNumber uint64
	TxIndex     uint32
}

func TxCursorFromString(s string) (TxCursor, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return TxCursor{}, fmt.Errorf("invalid cursor string: %s", s)
	}

	blockNumber, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return TxCursor{}, fmt.Errorf("invalid block number: %w", err)
	}

	txIndex, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return TxCursor{}, fmt.Errorf("invalid transaction index: %w", err)
	}

	return TxCursor{
		BlockNumber: blockNumber,
		TxIndex:     uint32(txIndex),
	}, nil
}

func (c *TxCursor) String() string {
	return fmt.Sprintf("%d.%d", c.BlockNumber, c.TxIndex)
}
