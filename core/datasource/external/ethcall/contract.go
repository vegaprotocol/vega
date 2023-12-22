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

package ethcall

import (
	"encoding/hex"
	"errors"
	"strconv"

	"code.vegaprotocol.io/vega/libs/crypto"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ContractCallEvent struct {
	BlockHeight uint64
	BlockTime   uint64
	SpecId      string
	Result      []byte
	Error       *string
	L2ChainID   *uint64
}

func EthereumContractCallResultFromProto(
	qr *vegapb.EthContractCallEvent,
) (ContractCallEvent, error) {
	if len(qr.SpecId) <= 0 || qr.BlockHeight == 0 || qr.BlockTime <= 0 ||
		(len(qr.Result) <= 0 && qr.Error == nil) {
		return ContractCallEvent{}, errors.New("invalid contract call payload")
	}

	return ContractCallEvent{
		SpecId:      qr.SpecId,
		BlockHeight: qr.BlockHeight,
		BlockTime:   qr.BlockTime,
		Result:      qr.Result,
		Error:       qr.Error,
		L2ChainID:   qr.L2ChainId,
	}, nil
}

func (q *ContractCallEvent) IntoProto() *vegapb.EthContractCallEvent {
	return &vegapb.EthContractCallEvent{
		SpecId:      q.SpecId,
		BlockHeight: q.BlockHeight,
		BlockTime:   q.BlockTime,
		Result:      q.Result,
		Error:       q.Error,
		L2ChainId:   q.L2ChainID,
	}
}

func (q ContractCallEvent) Hash() string {
	blockHeight := strconv.FormatUint(q.BlockHeight, 10)
	blockTime := strconv.FormatUint(q.BlockTime, 10)
	bytes := []byte(blockHeight + blockTime + q.SpecId)
	bytes = append(bytes, q.Result...)
	if q.Error != nil {
		bytes = append(bytes, []byte(*q.Error)...)
	}

	return hex.EncodeToString(
		crypto.Hash(bytes),
	)
}
