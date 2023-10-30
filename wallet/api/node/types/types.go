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

package types

import "fmt"

type TransactionError struct {
	ABCICode uint32
	Message  string
}

func (e TransactionError) Error() string {
	return fmt.Sprintf("%s (ABCI code %d)", e.Message, e.ABCICode)
}

func (e TransactionError) Is(target error) bool {
	_, ok := target.(TransactionError)
	return ok
}

type Statistics struct {
	BlockHash   string
	BlockHeight uint64
	ChainID     string
	VegaTime    string
}

type SpamStatistics struct {
	ChainID           string
	EpochSeq          uint64
	LastBlockHeight   uint64
	Proposals         *SpamStatistic
	Delegations       *SpamStatistic
	Transfers         *SpamStatistic
	NodeAnnouncements *SpamStatistic
	IssuesSignatures  *SpamStatistic
	CreateReferralSet *SpamStatistic
	UpdateReferralSet *SpamStatistic
	ApplyReferralCode *SpamStatistic
	Votes             *VoteSpamStatistics
	PoW               *PoWStatistics
}

type SpamStatistic struct {
	CountForEpoch uint64
	MaxForEpoch   uint64
	BannedUntil   *string
}

type VoteSpamStatistics struct {
	Proposals   map[string]uint64
	MaxForEpoch uint64
	BannedUntil *string
}

type PoWStatistics struct {
	PowBlockStates []PoWBlockState
	PastBlocks     uint64
	BannedUntil    *string
}

type PoWBlockState struct {
	BlockHeight          uint64
	BlockHash            string
	TransactionsSeen     uint64
	ExpectedDifficulty   *uint64
	HashFunction         string
	Difficulty           uint64
	TxPerBlock           uint64
	IncreasingDifficulty bool
}

type LastBlock struct {
	ChainID                         string
	BlockHeight                     uint64
	BlockHash                       string
	ProofOfWorkHashFunction         string
	ProofOfWorkDifficulty           uint32
	ProofOfWorkPastBlocks           uint32
	ProofOfWorkTxPerBlock           uint32
	ProofOfWorkIncreasingDifficulty bool
}
