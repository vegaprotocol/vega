//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
)

type OrderSubmission = commandspb.OrderSubmission
type OrderCancellation = commandspb.OrderCancellation
type OrderAmendment = commandspb.OrderAmendment
type WithdrawSubmission = commandspb.WithdrawSubmission
type OracleDataSubmission = commandspb.OracleDataSubmission
type NodeRegistration = commandspb.NodeRegistration
type NodeVote = commandspb.NodeVote
type Transaction = proto.Transaction
type ChainEvent = commandspb.ChainEvent
type SignedBundle = proto.SignedBundle
type NetworkParameter = proto.NetworkParameter
type Signature = proto.Signature
type Transaction_PubKey = proto.Transaction_PubKey
