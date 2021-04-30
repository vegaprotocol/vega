package mocks

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/oracles"
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/txn"
	"code.vegaprotocol.io/vega/validators"

	"github.com/golang/protobuf/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination node_wallet_mock.go -package mocks code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks NodeWallet
type NodeWallet interface {
	Get(chain nodewallet.Blockchain) (nodewallet.Wallet, bool)
}

//go:generate go run github.com/golang/mock/mockgen -destination broker_mock.go -package mocks code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks Broker
type Broker interface {
	Send(e events.Event)
	SendBatch(e []events.Event)
}

//go:generate go run github.com/golang/mock/mockgen -destination notary_mock.go -package mocks code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks Notary
type Notary interface {
	StartAggregate(resID string, kind commandspb.NodeSignatureKind) error
	SendSignature(ctx context.Context, id string, sig []byte, kind commandspb.NodeSignatureKind) error
	IsSigned(ctx context.Context, id string, kind commandspb.NodeSignatureKind) ([]commandspb.NodeSignature, bool)
	AddSig(ctx context.Context, pubKey []byte, ns commandspb.NodeSignature) ([]commandspb.NodeSignature, bool, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination witness_mock.go -package mocks code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks Witness
type Witness interface {
	StartCheck(validators.Resource, func(interface{}, bool), time.Time) error
	AddNodeCheck(ctx context.Context, nv *commandspb.NodeVote) error
}

//go:generate go run github.com/golang/mock/mockgen -destination evtforwarder_mock.go -package mocks code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks EvtForwarder
type EvtForwarder interface {
	Ack(*commandspb.ChainEvent) bool
}

//go:generate go run github.com/golang/mock/mockgen -destination oracle_engine_mock.go -package mocks code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks OracleEngine
type OracleEngine interface {
	BroadcastData(context.Context, oracles.OracleData) error
	Subscribe(context.Context, oracles.OracleSpec, oracles.OnMatchedOracleData) oracles.SubscriptionID
	Unsubscribe(context.Context, oracles.SubscriptionID)
}

//go:generate go run github.com/golang/mock/mockgen -destination oracle_adaptors_mock.go -package mocks code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks OracleAdaptors
type OracleAdaptors interface {
	Normalise(types.OracleDataSubmission) (*oracles.OracleData, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination commander_mock.go -package mocks code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks Commander
type Commander interface {
	Command(ctx context.Context, cmd txn.Command, payload proto.Message) error
}

//go:generate go run github.com/golang/mock/mockgen -destination governance_engine_mock.go -package mocks code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks GovernanceEngine
type GovernanceEngine interface {
	SubmitProposal(context.Context, commandspb.ProposalSubmission, string, string) (*governance.ToSubmit, error)
	AddVote(context.Context, commandspb.VoteSubmission, string) error
	OnChainTimeUpdate(context.Context, time.Time) ([]*governance.ToEnact, []*governance.VoteClosed)
	RejectProposal(context.Context, *types.Proposal, types.ProposalError) error
}
