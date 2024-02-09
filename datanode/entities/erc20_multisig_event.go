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

package entities

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type NotaryService interface {
	GetByResourceID(ctx context.Context, id string, pagination CursorPagination) ([]NodeSignature, PageInfo, error)
}

type ERC20MultiSigSignerEventType string

const (
	ERC20MultiSigSignerEventTypeAdded   ERC20MultiSigSignerEventType = "SIGNER_ADDED"
	ERC20MultiSigSignerEventTypeRemoved ERC20MultiSigSignerEventType = "SIGNER_REMOVED"
)

type _ERC20MultiSigSignerEvent struct{}

type ERC20MultiSigSignerEventID = ID[_ERC20MultiSigSignerEvent]

type ERC20MultiSigSignerEvent struct {
	ID           ERC20MultiSigSignerEventID
	ValidatorID  NodeID
	SignerChange EthereumAddress
	Submitter    EthereumAddress
	Nonce        string
	TxHash       TxHash
	VegaTime     time.Time
	EpochID      int64
	Event        ERC20MultiSigSignerEventType
}

func (e ERC20MultiSigSignerEvent) Cursor() *Cursor {
	ec := ERC20MultiSigSignerEventCursor{
		VegaTime: e.VegaTime,
		ID:       e.ID,
	}

	return NewCursor(ec.String())
}

func ERC20MultiSigSignerEventFromAddedProto(e *eventspb.ERC20MultiSigSignerAdded, txHash TxHash) (*ERC20MultiSigSignerEvent, error) {
	epochID, err := strconv.ParseInt(e.EpochSeq, 10, 64)
	if err != nil {
		return &ERC20MultiSigSignerEvent{}, fmt.Errorf("parsing epoch '%v': %w", e.EpochSeq, err)
	}
	return &ERC20MultiSigSignerEvent{
		ID:           ERC20MultiSigSignerEventID(e.SignatureId),
		ValidatorID:  NodeID(e.ValidatorId),
		SignerChange: EthereumAddress(e.NewSigner),
		Submitter:    EthereumAddress(e.Submitter),
		Nonce:        e.Nonce,
		TxHash:       txHash,
		VegaTime:     time.Unix(0, e.Timestamp),
		EpochID:      epochID,
		Event:        ERC20MultiSigSignerEventTypeAdded,
	}, nil
}

func ERC20MultiSigSignerEventFromRemovedProto(e *eventspb.ERC20MultiSigSignerRemoved, txHash TxHash) ([]*ERC20MultiSigSignerEvent, error) {
	events := []*ERC20MultiSigSignerEvent{}

	epochID, err := strconv.ParseInt(e.EpochSeq, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing epoch '%v': %w", e.EpochSeq, err)
	}
	for _, s := range e.SignatureSubmitters {
		events = append(events, &ERC20MultiSigSignerEvent{
			ID:           ERC20MultiSigSignerEventID(s.SignatureId),
			Submitter:    EthereumAddress(s.Submitter),
			SignerChange: EthereumAddress(e.OldSigner),
			ValidatorID:  NodeID(e.ValidatorId),
			Nonce:        e.Nonce,
			TxHash:       txHash,
			VegaTime:     time.Unix(0, e.Timestamp),
			EpochID:      epochID,
			Event:        ERC20MultiSigSignerEventTypeRemoved,
		},
		)
	}

	return events, nil
}

type ERC20MultiSigSignerEventCursor struct {
	VegaTime time.Time                  `json:"vega_time"`
	ID       ERC20MultiSigSignerEventID `json:"id"`
}

func (c ERC20MultiSigSignerEventCursor) String() string {
	bs, err := json.Marshal(c)
	// This should never fail so we should panic if it does
	if err != nil {
		panic(fmt.Errorf("failed to convert"))
	}
	return string(bs)
}

func (c *ERC20MultiSigSignerEventCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), c)
}

type ERC20MultiSigSignerAddedEvent struct {
	ERC20MultiSigSignerEvent
}

func (e ERC20MultiSigSignerAddedEvent) Cursor() *Cursor {
	ec := ERC20MultiSigSignerEventCursor{
		VegaTime: e.VegaTime,
		ID:       e.ID,
	}

	return NewCursor(ec.String())
}

func (e ERC20MultiSigSignerAddedEvent) ToProto() *eventspb.ERC20MultiSigSignerAdded {
	return &eventspb.ERC20MultiSigSignerAdded{
		SignatureId: e.ID.String(),
		ValidatorId: e.ValidatorID.String(),
		Timestamp:   e.VegaTime.UnixNano(),
		NewSigner:   e.SignerChange.String(),
		Submitter:   e.Submitter.String(),
		Nonce:       e.Nonce,
		EpochSeq:    strconv.FormatInt(e.EpochID, 10),
	}
}

func (e ERC20MultiSigSignerAddedEvent) ToDataNodeApiV2Proto(ctx context.Context, notaryService NotaryService) (*v2.ERC20MultiSigSignerAddedBundle, error) {
	signatures, _, err := notaryService.GetByResourceID(ctx, e.ID.String(), CursorPagination{})
	if err != nil {
		return nil, err
	}

	return &v2.ERC20MultiSigSignerAddedBundle{
		NewSigner:  e.SignerChange.String(),
		Submitter:  e.Submitter.String(),
		Nonce:      e.Nonce,
		Timestamp:  e.VegaTime.UnixNano(),
		Signatures: PackNodeSignatures(signatures),
		EpochSeq:   strconv.FormatInt(e.EpochID, 10),
	}, nil
}

func (e ERC20MultiSigSignerAddedEvent) ToProtoEdge(_ ...any) (*v2.ERC20MultiSigSignerAddedEdge, error) {
	return &v2.ERC20MultiSigSignerAddedEdge{
		Node:   e.ToProto(),
		Cursor: e.Cursor().Encode(),
	}, nil
}

type ERC20MultiSigSignerRemovedEvent struct {
	ERC20MultiSigSignerEvent
}

func (e ERC20MultiSigSignerRemovedEvent) Cursor() *Cursor {
	ec := ERC20MultiSigSignerEventCursor{
		VegaTime: e.VegaTime,
		ID:       e.ID,
	}

	return NewCursor(ec.String())
}

func (e ERC20MultiSigSignerRemovedEvent) ToProto() *eventspb.ERC20MultiSigSignerRemoved {
	return &eventspb.ERC20MultiSigSignerRemoved{
		SignatureSubmitters: nil,
		ValidatorId:         e.ValidatorID.String(),
		Timestamp:           e.VegaTime.UnixNano(),
		OldSigner:           e.SignerChange.String(),
		Nonce:               e.Nonce,
		EpochSeq:            strconv.FormatInt(e.EpochID, 10),
	}
}

func (e ERC20MultiSigSignerRemovedEvent) ToDataNodeApiV2Proto(ctx context.Context, notaryService NotaryService) (*v2.ERC20MultiSigSignerRemovedBundle, error) {
	signatures, _, err := notaryService.GetByResourceID(ctx, e.ID.String(), CursorPagination{})
	if err != nil {
		return nil, err
	}

	return &v2.ERC20MultiSigSignerRemovedBundle{
		OldSigner:  e.SignerChange.String(),
		Submitter:  e.Submitter.String(),
		Nonce:      e.Nonce,
		Timestamp:  e.VegaTime.UnixNano(),
		Signatures: PackNodeSignatures(signatures),
		EpochSeq:   strconv.FormatInt(e.EpochID, 10),
	}, nil
}

func (e ERC20MultiSigSignerRemovedEvent) ToProtoEdge(_ ...any) (*v2.ERC20MultiSigSignerRemovedEdge, error) {
	return &v2.ERC20MultiSigSignerRemovedEdge{
		Node:   e.ToProto(),
		Cursor: e.Cursor().Encode(),
	}, nil
}
