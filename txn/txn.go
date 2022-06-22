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

package txn

import (
	"errors"

	uuid "github.com/satori/go.uuid"
)

// Encode takes a Tx payload and a command and builds a raw Tx.
func Encode(input []byte, cmd Command) ([]byte, error) {
	prefix := uuid.NewV4().String()
	prefixBytes := []byte(prefix)
	commandInput := append([]byte{byte(cmd)}, input...)
	return append(prefixBytes, commandInput...), nil
}

// Decode takes the raw payload bytes and decodes the contents using a pre-defined
// strategy, we have a simple and efficient encoding at present. A partner encode function
// can be found in the blockchain client.
func Decode(input []byte) ([]byte, Command, error) {
	// Input is typically the bytes that arrive in raw format after consensus is reached.
	// Split the transaction dropping the unification bytes (uuid&pipe)
	if len(input) >= 37 {
		// obtain command from byte slice (0 indexed)
		// remaining bytes are payload
		return input[37:], Command(input[36]), nil
	}
	return nil, 0, errors.New("payload size is incorrect, should be > 38 bytes")
}
