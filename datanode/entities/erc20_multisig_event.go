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

package entities

import (
	"fmt"
	"strconv"
	"time"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type ERC20MultiSigSignerEventType string

const (
	ERC20MultiSigSignerEventTypeAdded   ERC20MultiSigSignerEventType = "SIGNER_ADDED"
	ERC20MultiSigSignerEventTypeRemoved ERC20MultiSigSignerEventType = "SIGNER_REMOVED"
)

type ERC20MultiSigSignerEventID struct{ ID }

func NewERC20MultiSigSignerEventID(id string) ERC20MultiSigSignerEventID {
	return ERC20MultiSigSignerEventID{ID: ID(id)}
}

type ERC20MultiSigSignerEvent struct {
	ID           ERC20MultiSigSignerEventID
	ValidatorID  NodeID
	SignerChange EthereumAddress
	Submitter    EthereumAddress
	Nonce        string
	VegaTime     time.Time
	EpochID      int64
	Event        ERC20MultiSigSignerEventType
}

func ERC20MultiSigSignerEventFromAddedProto(e *eventspb.ERC20MultiSigSignerAdded) (*ERC20MultiSigSignerEvent, error) {
	epochID, err := strconv.ParseInt(e.EpochSeq, 10, 64)
	if err != nil {
		return &ERC20MultiSigSignerEvent{}, fmt.Errorf("parsing epoch '%v': %w", e.EpochSeq, err)
	}
	return &ERC20MultiSigSignerEvent{
		ID:           NewERC20MultiSigSignerEventID(e.SignatureId),
		ValidatorID:  NewNodeID(e.ValidatorId),
		SignerChange: EthereumAddress(e.NewSigner),
		Submitter:    EthereumAddress(e.Submitter),
		Nonce:        e.Nonce,
		VegaTime:     time.Unix(0, e.Timestamp),
		EpochID:      epochID,
		Event:        ERC20MultiSigSignerEventTypeAdded,
	}, nil
}

func ERC20MultiSigSignerEventFromRemovedProto(e *eventspb.ERC20MultiSigSignerRemoved) ([]*ERC20MultiSigSignerEvent, error) {
	ents := []*ERC20MultiSigSignerEvent{}

	epochID, err := strconv.ParseInt(e.EpochSeq, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing epoch '%v': %w", e.EpochSeq, err)
	}
	for _, s := range e.SignatureSubmitters {
		ents = append(ents, &ERC20MultiSigSignerEvent{
			ID:           NewERC20MultiSigSignerEventID(s.SignatureId),
			Submitter:    EthereumAddress(s.Submitter),
			SignerChange: EthereumAddress(e.OldSigner),
			ValidatorID:  NewNodeID(e.ValidatorId),
			Nonce:        e.Nonce,
			VegaTime:     time.Unix(0, e.Timestamp),
			EpochID:      epochID,
			Event:        ERC20MultiSigSignerEventTypeRemoved,
		},
		)
	}

	return ents, nil
}
