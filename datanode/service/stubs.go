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
	"code.vegaprotocol.io/vega/logging"
)

type (
	Asset               struct{ *sqlstore.Assets }
	Block               struct{ *sqlstore.Blocks }
	Party               struct{ *sqlstore.Parties }
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
	TransferInstruction struct{ *sqlstore.TransferInstructions }
	StakeLinking        struct{ *sqlstore.StakeLinking }
	Notary              struct{ *sqlstore.Notary }
	MultiSig            struct {
		*sqlstore.ERC20MultiSigSignerEvent
	}
)

type (
	KeyRotations struct{ *sqlstore.KeyRotations }
	Node         struct{ *sqlstore.Node }
)

func NewAsset(store *sqlstore.Assets, log *logging.Logger) *Asset {
	return &Asset{Assets: store}
}

func NewBlock(store *sqlstore.Blocks, log *logging.Logger) *Block {
	return &Block{Blocks: store}
}

func NewParty(store *sqlstore.Parties, log *logging.Logger) *Party {
	return &Party{Parties: store}
}

func NewNetworkLimits(store *sqlstore.NetworkLimits, log *logging.Logger) *NetworkLimits {
	return &NetworkLimits{NetworkLimits: store}
}

func NewEpoch(store *sqlstore.Epochs, log *logging.Logger) *Epoch {
	return &Epoch{Epochs: store}
}

func NewDeposit(store *sqlstore.Deposits, log *logging.Logger) *Deposit {
	return &Deposit{Deposits: store}
}

func NewWithdrawal(store *sqlstore.Withdrawals, log *logging.Logger) *Withdrawal {
	return &Withdrawal{Withdrawals: store}
}

func NewRiskFactor(store *sqlstore.RiskFactors, log *logging.Logger) *RiskFactor {
	return &RiskFactor{RiskFactors: store}
}

func NewNetworkParameter(store *sqlstore.NetworkParameters, log *logging.Logger) *NetworkParameter {
	return &NetworkParameter{NetworkParameters: store}
}

func NewCheckpoint(store *sqlstore.Checkpoints, log *logging.Logger) *Checkpoint {
	return &Checkpoint{Checkpoints: store}
}

func NewOracleSpec(store *sqlstore.OracleSpec, log *logging.Logger) *OracleSpec {
	return &OracleSpec{OracleSpec: store}
}

func NewOracleData(store *sqlstore.OracleData, log *logging.Logger) *OracleData {
	return &OracleData{OracleData: store}
}

func NewLiquidityProvision(store *sqlstore.LiquidityProvision, log *logging.Logger) *LiquidityProvision {
	return &LiquidityProvision{LiquidityProvision: store}
}

func NewTransferInstruction(store *sqlstore.TransferInstructions, log *logging.Logger) *TransferInstruction {
	return &TransferInstruction{TransferInstructions: store}
}

func NewStakeLinking(store *sqlstore.StakeLinking, log *logging.Logger) *StakeLinking {
	return &StakeLinking{StakeLinking: store}
}

func NewNotary(store *sqlstore.Notary, log *logging.Logger) *Notary {
	return &Notary{Notary: store}
}

func NewMultiSig(store *sqlstore.ERC20MultiSigSignerEvent, log *logging.Logger) *MultiSig {
	return &MultiSig{ERC20MultiSigSignerEvent: store}
}

func NewKeyRotations(store *sqlstore.KeyRotations, log *logging.Logger) *KeyRotations {
	return &KeyRotations{KeyRotations: store}
}

func NewNode(store *sqlstore.Node, log *logging.Logger) *Node {
	return &Node{Node: store}
}
