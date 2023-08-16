// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package service

import (
	"code.vegaprotocol.io/vega/datanode/sqlstore"
)

type (
	Asset               struct{ *sqlstore.Assets }
	Block               struct{ *sqlstore.Blocks }
	Party               struct{ *sqlstore.Parties }
	PartyActivityStreak struct{ *sqlstore.PartyActivityStreaks }
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
	FundingPeriods   struct{ *sqlstore.FundingPeriods }
	ReferralPrograms struct{ *sqlstore.ReferralPrograms }
	ReferralSets     struct{ *sqlstore.ReferralSets }
	Teams            struct{ *sqlstore.Teams }
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
