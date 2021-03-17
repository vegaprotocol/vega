//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import "code.vegaprotocol.io/vega/proto"

type OrderSubmission = proto.OrderSubmission
type OrderCancellation = proto.OrderCancellation
type OrderAmendment = proto.OrderAmendment
type WithdrawSubmission = proto.WithdrawSubmission
type OracleDataSubmission = proto.OracleDataSubmission
type NodeRegistration = proto.NodeRegistration
type NodeVote = proto.NodeVote
type Transaction = proto.Transaction
type ChainEvent = proto.ChainEvent
type SignedBundle = proto.SignedBundle
type NetworkParameter = proto.NetworkParameter
type Signature = proto.Signature
type Transaction_PubKey = proto.Transaction_PubKey
