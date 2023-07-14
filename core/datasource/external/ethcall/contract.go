// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package ethcall

import (
	"encoding/hex"
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
}

func EthereumContractCallResultFromProto(
	qr *vegapb.EthContractCallEvent,
) ContractCallEvent {
	return ContractCallEvent{
		SpecId:      qr.SpecId,
		BlockHeight: qr.BlockHeight,
		BlockTime:   qr.BlockTime,
		Result:      qr.Result,
		Error:       qr.Error,
	}
}

func (q *ContractCallEvent) IntoProto() *vegapb.EthContractCallEvent {
	return &vegapb.EthContractCallEvent{
		SpecId:      q.SpecId,
		BlockHeight: q.BlockHeight,
		BlockTime:   q.BlockTime,
		Result:      q.Result,
		Error:       q.Error,
	}
}

func (q ContractCallEvent) Hash() string {
	blockHeight := strconv.FormatUint(q.BlockHeight, 10)
	blockTime := strconv.FormatUint(q.BlockHeight, 10)
	bytes := []byte(blockHeight + blockTime + q.SpecId)
	bytes = append(bytes, q.Result...)
	if q.Error != nil {
		bytes = append(bytes, []byte(*q.Error)...)
	}

	return hex.EncodeToString(
		crypto.Hash(bytes),
	)
}
