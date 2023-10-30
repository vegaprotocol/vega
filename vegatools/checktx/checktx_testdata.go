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

package checktx

import (
	"encoding/base64"
	"fmt"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/libs/proto"
	v1 "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

const (
	TxVersion      = 3
	TestAlgoName   = "testAlgo"
	TestPubKeyName = "testPubKey"
)

var TestInputData = v1.InputData{Nonce: 123, BlockHeight: 456, Command: &v1.InputData_Transfer{Transfer: &v1.Transfer{
	FromAccountType: 1,
	To:              "dave",
	ToAccountType:   2,
	Asset:           "test asset",
	Amount:          "123",
	Reference:       "test ref",
}}}

func CreateTransaction() (*v1.Transaction, error) {
	marshalledInputData, err := commands.MarshalInputData(&TestInputData)
	if err != nil {
		return &v1.Transaction{}, fmt.Errorf("error occurred when mashalling test input data\nerr: %w", err)
	}
	return commands.NewTransaction(TestPubKeyName, marshalledInputData, commands.NewSignature([]byte("sig"), TestAlgoName, TxVersion)), nil
}

func CreatedEncodedTransactionData() (string, error) {
	transaction, err := CreateTransaction()
	if err != nil {
		return "", fmt.Errorf("error occurred when creating a transaction \nerr: %w", err)
	}
	transactionProto, err := proto.Marshal(transaction)
	if err != nil {
		return "", fmt.Errorf("error occurred when marshalling the transaction to a proto\nerr: %w", err)
	}

	encodedTransaction := base64.StdEncoding.EncodeToString(transactionProto)
	return encodedTransaction, nil
}
