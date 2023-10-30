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

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
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
