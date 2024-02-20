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

package service

import (
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers"
)

type (
	Asset               struct{ *sqlstore.Assets }
	Block               struct{ *sqlstore.Blocks }
	Party               struct{ *sqlstore.Parties }
	PartyActivityStreak struct{ *sqlstore.PartyActivityStreaks }
	FundingPayment      struct{ *sqlstore.FundingPayments }
	NetworkLimits       struct{ *sqlstore.NetworkLimits }
	Epoch               struct{ *sqlstore.Epochs }
	Deposit             struct{ *sqlstore.Deposits }
	Withdrawal          struct{ *sqlstore.Withdrawals }
	RiskFactor          struct{ *sqlstore.RiskFactors }
	NetworkParameter    struct{ *sqlstore.NetworkParameters }
	Checkpoint          struct{ *sqlstore.Checkpoints }
	OracleSpec          struct{ *sqlstore.OracleSpec }
	OracleData          struct{ *sqlstore.OracleData }
	LiquidityProvision  struct{ *sqlstore.LiquidityProvision }
	Transfer            struct{ *sqlstore.Transfers }
	StakeLinking        struct{ *sqlstore.StakeLinking }
	Notary              struct{ *sqlstore.Notary }
	MultiSig            struct {
		*sqlstore.ERC20MultiSigSignerEvent
	}
	FundingPeriods         struct{ *sqlstore.FundingPeriods }
	ReferralPrograms       struct{ *sqlstore.ReferralPrograms }
	ReferralSets           struct{ *sqlstore.ReferralSets }
	Teams                  struct{ *sqlstore.Teams }
	VestingStats           struct{ *sqlstore.VestingStats }
	VolumeDiscountStats    struct{ *sqlstore.VolumeDiscountStats }
	FeesStats              struct{ *sqlstore.FeesStats }
	VolumeDiscountPrograms struct {
		*sqlstore.VolumeDiscountPrograms
	}
	PaidLiquidityFeesStats struct {
		*sqlstore.PaidLiquidityFeesStats
	}
	PartyLockedBalances struct {
		*sqlstore.PartyLockedBalance
	}
	PartyVestingBalances struct {
		*sqlstore.PartyVestingBalance
	}
	TransactionResults struct {
		*sqlsubscribers.TransactionResults
	}
	Games       struct{ *sqlstore.Games }
	MarginModes struct{ *sqlstore.MarginModes }
	AMMPools    struct{ *sqlstore.AMMPools }
)

type (
	KeyRotations struct{ *sqlstore.KeyRotations }
	Node         struct{ *sqlstore.Node }
)

func NewAsset(store *sqlstore.Assets) *Asset {
	return &Asset{Assets: store}
}

func NewBlock(store *sqlstore.Blocks) *Block {
	return &Block{Blocks: store}
}

func NewParty(store *sqlstore.Parties) *Party {
	return &Party{Parties: store}
}

func NewPartyActivityStreak(store *sqlstore.PartyActivityStreaks) *PartyActivityStreak {
	return &PartyActivityStreak{PartyActivityStreaks: store}
}

func NewFundingPayment(store *sqlstore.FundingPayments) *FundingPayment {
	return &FundingPayment{FundingPayments: store}
}

func NewNetworkLimits(store *sqlstore.NetworkLimits) *NetworkLimits {
	return &NetworkLimits{NetworkLimits: store}
}

func NewEpoch(store *sqlstore.Epochs) *Epoch {
	return &Epoch{Epochs: store}
}

func NewDeposit(store *sqlstore.Deposits) *Deposit {
	return &Deposit{Deposits: store}
}

func NewWithdrawal(store *sqlstore.Withdrawals) *Withdrawal {
	return &Withdrawal{Withdrawals: store}
}

func NewRiskFactor(store *sqlstore.RiskFactors) *RiskFactor {
	return &RiskFactor{RiskFactors: store}
}

func NewNetworkParameter(store *sqlstore.NetworkParameters) *NetworkParameter {
	return &NetworkParameter{NetworkParameters: store}
}

func NewCheckpoint(store *sqlstore.Checkpoints) *Checkpoint {
	return &Checkpoint{Checkpoints: store}
}

func NewOracleSpec(store *sqlstore.OracleSpec) *OracleSpec {
	return &OracleSpec{OracleSpec: store}
}

func NewOracleData(store *sqlstore.OracleData) *OracleData {
	return &OracleData{OracleData: store}
}

func NewLiquidityProvision(store *sqlstore.LiquidityProvision) *LiquidityProvision {
	return &LiquidityProvision{LiquidityProvision: store}
}

func NewTransfer(store *sqlstore.Transfers) *Transfer {
	return &Transfer{Transfers: store}
}

func NewStakeLinking(store *sqlstore.StakeLinking) *StakeLinking {
	return &StakeLinking{StakeLinking: store}
}

func NewNotary(store *sqlstore.Notary) *Notary {
	return &Notary{Notary: store}
}

func NewMultiSig(store *sqlstore.ERC20MultiSigSignerEvent) *MultiSig {
	return &MultiSig{ERC20MultiSigSignerEvent: store}
}

func NewKeyRotations(store *sqlstore.KeyRotations) *KeyRotations {
	return &KeyRotations{KeyRotations: store}
}

func NewNode(store *sqlstore.Node) *Node {
	return &Node{Node: store}
}

func NewFundingPeriods(store *sqlstore.FundingPeriods) *FundingPeriods {
	return &FundingPeriods{FundingPeriods: store}
}

func NewReferralPrograms(store *sqlstore.ReferralPrograms) *ReferralPrograms {
	return &ReferralPrograms{ReferralPrograms: store}
}

func NewReferralSets(store *sqlstore.ReferralSets) *ReferralSets {
	return &ReferralSets{ReferralSets: store}
}

func NewTeams(store *sqlstore.Teams) *Teams {
	return &Teams{Teams: store}
}

func NewVestingStats(store *sqlstore.VestingStats) *VestingStats {
	return &VestingStats{VestingStats: store}
}

func NewVolumeDiscountStats(store *sqlstore.VolumeDiscountStats) *VolumeDiscountStats {
	return &VolumeDiscountStats{VolumeDiscountStats: store}
}

func NewFeesStats(store *sqlstore.FeesStats) *FeesStats {
	return &FeesStats{FeesStats: store}
}

func NewVolumeDiscountPrograms(store *sqlstore.VolumeDiscountPrograms) *VolumeDiscountPrograms {
	return &VolumeDiscountPrograms{VolumeDiscountPrograms: store}
}

func NewPaidLiquidityFeesStats(store *sqlstore.PaidLiquidityFeesStats) *PaidLiquidityFeesStats {
	return &PaidLiquidityFeesStats{PaidLiquidityFeesStats: store}
}

func NewPartyLockedBalances(store *sqlstore.PartyLockedBalance) *PartyLockedBalances {
	return &PartyLockedBalances{PartyLockedBalance: store}
}

func NewPartyVestingBalances(store *sqlstore.PartyVestingBalance) *PartyVestingBalances {
	return &PartyVestingBalances{PartyVestingBalance: store}
}

func NewTransactionResults(subscriber *sqlsubscribers.TransactionResults) *TransactionResults {
	return &TransactionResults{TransactionResults: subscriber}
}

func NewGames(store *sqlstore.Games) *Games {
	return &Games{Games: store}
}

func NewMarginModes(store *sqlstore.MarginModes) *MarginModes {
	return &MarginModes{MarginModes: store}
}

func NewAMMPools(store *sqlstore.AMMPools) *AMMPools {
	return &AMMPools{AMMPools: store}
}
