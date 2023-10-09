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
	"fmt"
	"strconv"
	"strings"
	"time"

	tmTypes "github.com/tendermint/tendermint/abci/types"
	"google.golang.org/protobuf/proto"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/ptr"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	pb "code.vegaprotocol.io/vega/protos/blockexplorer/api/v1"
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
		BlockNumber: blockNumber, // increase by one again to make the behaviour consistent
		TxIndex:     uint32(txIndex),
	}, nil
}

func (c *TxCursor) String() string {
	return fmt.Sprintf("%d.%d", c.BlockNumber, c.TxIndex)
}
