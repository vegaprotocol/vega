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

package tools

import (
	"testing"

	"code.vegaprotocol.io/vega/vegatools/checktx"

	"github.com/stretchr/testify/assert"
)

func TestTxReturnsNoErrorWhenCheckingCompatibleTransaction(t *testing.T) {
	encodedTransaction, err := checktx.CreatedEncodedTransactionData()
	assert.NoErrorf(t, err, "error was returned when creating test data\nerr: %v", err)

	cmd := checkTxCmd{
		EncodedTransaction: encodedTransaction,
	}

	err = cmd.Execute(nil)
	assert.NoErrorf(t, err, "error was returned when the transaction should have been valid")
}

func TestTxReturnsErrorWhenCheckingIncompatibleTransaction(t *testing.T) {
	cmd := checkTxCmd{
		EncodedTransaction: "12345",
	}

	err := cmd.Execute(nil)
	assert.Error(t, err)
}
