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

package banking

import (
	"encoding/hex"
	"errors"
	"fmt"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	vgproto "code.vegaprotocol.io/vega/libs/proto"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

var ErrChainEventAlreadySeen = errors.New("chain event already processed")

// deduplicateAssetAction returns true if the action has been deduplicated, false
// if the action has never been seen before.
func (e *Engine) deduplicateAssetAction(action *assetAction) error {
	ref := action.getRef()

	refKey, err := buildRefKey(ref)
	if err != nil {
		return fmt.Errorf("could not build reference key: %w", err)
	}

	// Check if this action have been seen before.
	if e.seenAssetActions.Contains(refKey) {
		return ErrChainEventAlreadySeen
	}

	// Prior the introduction of the second bridge, the TxRef did
	// not track the chain ID. Now that all TxRef have the chain ID filled,
	// we must ensure an older TxRef is not applied twice because it got its
	// chain ID valued, during a replay.
	//
	// This verification is only meaningful on actions coming from Ethereum
	// Mainnet, hence the condition.
	if ref.ChainId != "" && ref.ChainId == e.primaryEthChainID {
		ref.ChainId = ""

		refKeyWithChainID, err := buildRefKey(ref)
		if err != nil {
			return fmt.Errorf("could not build reference key: %w", err)
		}

		if e.seenAssetActions.Contains(refKeyWithChainID) {
			return ErrChainEventAlreadySeen
		}
	}

	// First time we see this transaction, so we keep track of it.
	e.seenAssetActions.Add(refKey)

	return nil
}

func buildRefKey(ref snapshot.TxRef) (string, error) {
	buf, err := vgproto.Marshal(&ref)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(vgcrypto.Hash(buf)), nil
}
