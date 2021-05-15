package proto

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

func NewTxFromSignedBundlePayload(payload []byte) (*Transaction, *SignedBundle, error) {
	bundle := &SignedBundle{}
	if err := proto.Unmarshal(payload, bundle); err != nil {
		return nil, nil, errors.Wrap(err, "unable to unmarshal signed bundle")
	}

	tx := &Transaction{}
	if err := proto.Unmarshal(bundle.Tx, tx); err != nil {
		return nil, nil, errors.Wrap(err, "unable to unmarshal transaction from signed bundle")
	}

	return tx, bundle, nil
}

// IsEvent methods needs to be implemented so we can used mapped types in GQL union
func (Asset) IsEvent()                           {}
func (LiquidityProvision) IsEvent()              {}
func (Vote) IsEvent()                            {}
func (Order) IsEvent()                           {}
func (Account) IsEvent()                         {}
func (Trade) IsEvent()                           {}
func (Party) IsEvent()                           {}
func (MarginLevels) IsEvent()                    {}
func (MarketData) IsEvent()                      {}
func (GovernanceData) IsEvent()                  {}
func (RiskFactor) IsEvent()                      {}
func (Deposit) IsEvent()                         {}
func (Withdrawal) IsEvent()                      {}
func (Market) IsEvent()                          {}
func (Future) IsProduct()                        {}
func (NewMarket) IsProposalChange()              {}
func (NewAsset) IsProposalChange()               {}
func (UpdateMarket) IsProposalChange()           {}
func (UpdateNetworkParameter) IsProposalChange() {}
func (LogNormalRiskModel) IsRiskModel()          {}
func (SimpleRiskModel) IsRiskModel()             {}
func (SimpleModelParams) IsRiskModel()           {}
