//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
)

type NodeSignature = commandspb.NodeSignature

type NodeSignatureKind = commandspb.NodeSignatureKind

const (
	// Represents an unspecified or missing value from the input
	NodeSignatureKindUnspecified NodeSignatureKind = commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_UNSPECIFIED
	// Represents a signature for a new asset allow-listing
	NodeSignatureKindAssetNew NodeSignatureKind = commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW
	// Represents a signature for an asset withdrawal
	NodeSignatureKindAssetWithdrawal NodeSignatureKind = commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL
)
