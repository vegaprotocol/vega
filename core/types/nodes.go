// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
)

type NodeSignature = commandspb.NodeSignature

type NodeSignatureKind = commandspb.NodeSignatureKind

const (
	// NodeSignatureKindUnspecified represents an unspecified or missing value from the input.
	NodeSignatureKindUnspecified NodeSignatureKind = commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_UNSPECIFIED
	// NodeSignatureKindAssetNew represents a signature for a new asset allow-listing.
	NodeSignatureKindAssetNew NodeSignatureKind = commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW
	// NodeSignatureKindAssetUpdate represents a signature for an asset update allow-listing.
	NodeSignatureKindAssetUpdate NodeSignatureKind = commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_UPDATE
	// Represents a signature for an asset withdrawal.
	NodeSignatureKindAssetWithdrawal            NodeSignatureKind = commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL
	NodeSignatureKindERC20MultiSigSignerAdded   NodeSignatureKind = commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ERC20_MULTISIG_SIGNER_ADDED
	NodeSignatureKindERC20MultiSigSignerRemoved NodeSignatureKind = commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ERC20_MULTISIG_SIGNER_REMOVED
)
