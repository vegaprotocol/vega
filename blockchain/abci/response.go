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

package abci

import "github.com/tendermint/tendermint/abci/types"

func NewResponseCheckTx(code uint32, info string) types.ResponseCheckTx {
	return types.ResponseCheckTx{
		Code: code,
		Info: info,
	}
}

func NewResponseCheckTxError(code uint32, err error) types.ResponseCheckTx {
	return types.ResponseCheckTx{
		Code: code,
		Info: err.Error(),
		Data: []byte(err.Error()),
	}
}

func NewResponseDeliverTx(code uint32, info string) types.ResponseDeliverTx {
	return types.ResponseDeliverTx{
		Code: code,
		Info: info,
	}
}

func NewResponseDeliverTxError(code uint32, err error) types.ResponseDeliverTx {
	return types.ResponseDeliverTx{
		Code: code,
		Info: err.Error(),
		Data: []byte(err.Error()),
	}
}
