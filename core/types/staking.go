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

package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	vgproto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/core/libs/crypto"
	"code.vegaprotocol.io/vega/core/types/num"
)

type StakeLinkingType = eventspb.StakeLinking_Type

const (
	StakeLinkingTypeUnspecified StakeLinkingType = eventspb.StakeLinking_TYPE_UNSPECIFIED
	StakeLinkingTypeDeposited                    = eventspb.StakeLinking_TYPE_LINK
	StakeLinkingTypeRemoved                      = eventspb.StakeLinking_TYPE_UNLINK
)

type StakeLinkingStatus = eventspb.StakeLinking_Status

const (
	StakeLinkingStatusUnspecified StakeLinkingStatus = eventspb.StakeLinking_STATUS_UNSPECIFIED
	StakeLinkingStatusPending                        = eventspb.StakeLinking_STATUS_PENDING
	StakeLinkingStatusAccepted                       = eventspb.StakeLinking_STATUS_ACCEPTED
	StakeLinkingStatusRejected                       = eventspb.StakeLinking_STATUS_REJECTED
)

type StakeTotalSupply struct {
	TokenAddress string
	TotalSupply  *num.Uint
}

func (s *StakeTotalSupply) IntoProto() *vgproto.StakeTotalSupply {
	return &vgproto.StakeTotalSupply{
		TokenAddress: s.TokenAddress,
		TotalSupply:  s.TotalSupply.String(),
	}
}

func (s *StakeTotalSupply) String() string {
	return fmt.Sprintf(
		"tokenAddress(%s) totalSupply(%s)",
		s.TokenAddress,
		uintPointerToString(s.TotalSupply),
	)
}

func StakeTotalSupplyFromProto(s *vgproto.StakeTotalSupply) (*StakeTotalSupply, error) {
	totalSupply := num.Zero()
	if len(s.TotalSupply) > 0 {
		var overflowed bool
		totalSupply, overflowed = num.UintFromString(s.TotalSupply, 10)
		if overflowed {
			return nil, errors.New("invalid amount (not a base 10 uint)")
		}
	}
	return &StakeTotalSupply{
		TokenAddress: crypto.EthereumChecksumAddress(s.TokenAddress),
		TotalSupply:  totalSupply,
	}, nil
}

type StakeLinking struct {
	ID              string
	Type            StakeLinkingType
	TS              int64
	Party           string
	Amount          *num.Uint
	Status          StakeLinkingStatus
	FinalizedAt     int64
	TxHash          string
	BlockHeight     uint64
	BlockTime       int64
	LogIndex        uint64
	EthereumAddress string
}

func (s StakeLinking) Hash() string {
	bn, li := strconv.FormatUint(s.BlockHeight, 10), strconv.FormatUint(s.LogIndex, 10)
	bt := strconv.FormatInt(s.BlockTime, 10)
	return hex.EncodeToString(
		crypto.Hash(
			[]byte(bn + li + bt + s.TxHash + s.Party + s.EthereumAddress + s.Amount.String() + s.Type.String()),
		),
	)
}

func (s *StakeLinking) String() string {
	return fmt.Sprintf(
		"ID(%s) type(%s) ts(%v) party(%s) amount(%s) status(%s) finalizedAt(%v) txHash(%s) blockHeight(%v) blockTime(%v) logIndex(%v) ethereumAddress(%s)",
		s.ID,
		s.Type.String(),
		s.TS,
		s.Party,
		uintPointerToString(s.Amount),
		s.Status.String(),
		s.FinalizedAt,
		s.TxHash,
		s.BlockHeight,
		s.BlockTime,
		s.LogIndex,
		s.EthereumAddress,
	)
}

func (s *StakeLinking) IntoProto() *eventspb.StakeLinking {
	return &eventspb.StakeLinking{
		Id:              s.ID,
		Type:            s.Type,
		Ts:              s.TS,
		Party:           s.Party,
		Amount:          num.UintToString(s.Amount),
		Status:          s.Status,
		FinalizedAt:     s.FinalizedAt,
		TxHash:          s.TxHash,
		BlockHeight:     s.BlockHeight,
		BlockTime:       s.BlockTime,
		LogIndex:        s.LogIndex,
		EthereumAddress: s.EthereumAddress,
	}
}

func StakeLinkingFromProto(sl *eventspb.StakeLinking) *StakeLinking {
	amt, _ := num.UintFromString(sl.Amount, 10)
	return &StakeLinking{
		ID:              sl.Id,
		Type:            sl.Type,
		TS:              sl.Ts,
		Party:           sl.Party,
		Amount:          amt,
		Status:          sl.Status,
		FinalizedAt:     sl.FinalizedAt,
		TxHash:          sl.TxHash,
		BlockHeight:     sl.BlockHeight,
		BlockTime:       sl.BlockTime,
		LogIndex:        sl.LogIndex,
		EthereumAddress: crypto.EthereumChecksumAddress(sl.EthereumAddress),
	}
}

type StakeDeposited struct {
	BlockNumber, LogIndex uint64
	TxID                  string // hash

	ID              string
	VegaPubKey      string
	EthereumAddress string
	Amount          *num.Uint
	BlockTime       int64
}

func StakeDepositedFromProto(
	s *vgproto.StakeDeposited,
	blockNumber, logIndex uint64,
	txID, id string,
) (*StakeDeposited, error) {
	amount := num.Zero()
	if len(s.Amount) > 0 {
		var overflowed bool
		amount, overflowed = num.UintFromString(s.Amount, 10)
		if overflowed {
			return nil, errors.New("invalid amount (not a base 10 uint)")
		}
	}

	return &StakeDeposited{
		ID:              id,
		BlockNumber:     blockNumber,
		LogIndex:        logIndex,
		TxID:            txID,
		VegaPubKey:      s.VegaPublicKey,
		EthereumAddress: crypto.EthereumChecksumAddress(s.EthereumAddress),
		Amount:          amount,
		BlockTime:       s.BlockTime,
	}, nil
}

func (s *StakeDeposited) IntoStakeLinking() *StakeLinking {
	return &StakeLinking{
		ID:              s.ID,
		Type:            StakeLinkingTypeDeposited,
		TS:              s.BlockTime,
		Party:           s.VegaPubKey,
		Amount:          s.Amount.Clone(),
		TxHash:          s.TxID,
		BlockHeight:     s.BlockNumber,
		BlockTime:       s.BlockTime,
		LogIndex:        s.LogIndex,
		EthereumAddress: s.EthereumAddress,
	}
}

func (s StakeDeposited) String() string {
	return fmt.Sprintf(
		"ID(%s) txID(%s) blockNumber(%v) logIndex(%v) vegaPubKey(%s) ethereumAddress(%s) amount(%s) blockTime(%v)",
		s.ID,
		s.TxID,
		s.BlockNumber,
		s.LogIndex,
		s.VegaPubKey,
		s.EthereumAddress,
		uintPointerToString(s.Amount),
		s.BlockTime,
	)
}

type StakeRemoved struct {
	BlockNumber, LogIndex uint64
	TxID                  string // hash

	ID              string
	VegaPubKey      string
	EthereumAddress string
	Amount          *num.Uint
	BlockTime       int64
}

func StakeRemovedFromProto(
	s *vgproto.StakeRemoved,
	blockNumber, logIndex uint64,
	txID, id string,
) (*StakeRemoved, error) {
	amount := num.Zero()
	if len(s.Amount) > 0 {
		var overflowed bool
		amount, overflowed = num.UintFromString(s.Amount, 10)
		if overflowed {
			return nil, errors.New("invalid amount (not a base 10 uint)")
		}
	}

	return &StakeRemoved{
		ID:              id,
		BlockNumber:     blockNumber,
		LogIndex:        logIndex,
		TxID:            txID,
		VegaPubKey:      s.VegaPublicKey,
		EthereumAddress: crypto.EthereumChecksumAddress(s.EthereumAddress),
		Amount:          amount,
		BlockTime:       s.BlockTime,
	}, nil
}

func (s StakeRemoved) String() string {
	return fmt.Sprintf(
		"ID(%s) txID(%s) blockNumber(%v) logIndex(%v) vegaPubKey(%s) ethereumAddress(%s) amount(%s) blockTime(%v)",
		s.ID,
		s.TxID,
		s.BlockNumber,
		s.LogIndex,
		s.VegaPubKey,
		s.EthereumAddress,
		uintPointerToString(s.Amount),
		s.BlockTime,
	)
}

func (s *StakeRemoved) IntoStakeLinking() *StakeLinking {
	return &StakeLinking{
		ID:              s.ID,
		Type:            StakeLinkingTypeRemoved,
		TS:              s.BlockTime,
		Party:           s.VegaPubKey,
		Amount:          s.Amount.Clone(),
		TxHash:          s.TxID,
		BlockHeight:     s.BlockNumber,
		BlockTime:       s.BlockTime,
		LogIndex:        s.LogIndex,
		EthereumAddress: crypto.EthereumChecksumAddress(s.EthereumAddress),
	}
}
