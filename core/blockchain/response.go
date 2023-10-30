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

package blockchain

import "github.com/tendermint/tendermint/abci/types"

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
