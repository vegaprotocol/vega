// Copyright (c) 2022 Gobalsky Labs Limited
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

package blockchain

import "github.com/cometbft/cometbft/abci/types"

const (
	// AbciTxnValidationFailure ...
	AbciTxnValidationFailure uint32 = 51

	// AbciTxnDecodingFailure code is returned when CheckTx or DeliverTx fail to decode the Txn.
	AbciTxnDecodingFailure uint32 = 60

	// AbciTxnInternalError code is returned when CheckTx or DeliverTx fail to process the Txn.
	AbciTxnInternalError uint32 = 70

	// AbciTxnPartialProcessingError code is return when a batch instruction partially fail.
	AbciTxnPartialProcessingError uint32 = 71

	// AbciUnknownCommandError code is returned when the app doesn't know how to handle a given command.
	AbciUnknownCommandError uint32 = 80

	// AbciSpamError code is returned when CheckTx or DeliverTx fail spam protection tests.
	AbciSpamError uint32 = 89
)

func NewResponseCheckTx(code uint32, info string) *types.ResponseCheckTx {
	return &types.ResponseCheckTx{
		Code: code,
		Info: info,
	}
}

func NewResponseCheckTxError(code uint32, err error) *types.ResponseCheckTx {
	return &types.ResponseCheckTx{
		Code: code,
		Info: err.Error(),
		Data: []byte(err.Error()),
	}
}

func NewResponseDeliverTx(code uint32, info string) *types.ExecTxResult {
	return &types.ExecTxResult{
		Code: code,
		Info: info,
	}
}

func NewResponseDeliverTxError(code uint32, err error) *types.ExecTxResult {
	return &types.ExecTxResult{
		Code: code,
		Info: err.Error(),
		Data: []byte(err.Error()),
	}
}
