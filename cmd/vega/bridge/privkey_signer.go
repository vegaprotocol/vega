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

package bridge

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
)

type PrivKeySigner struct {
	privateKey *ecdsa.PrivateKey
}

func NewPrivKeySigner(hexPrivKey string) (*PrivKeySigner, error) {
	privateKey, err := crypto.HexToECDSA(hexPrivKey)
	if err != nil {
		return nil, err
	}

	return &PrivKeySigner{
		privateKey: privateKey,
	}, nil
}

func (p *PrivKeySigner) Sign(hash []byte) ([]byte, error) {
	return crypto.Sign(hash, p.privateKey)
}

func (p *PrivKeySigner) Algo() string {
	return ""
}
