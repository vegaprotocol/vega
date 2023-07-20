// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package common

import (
	"time"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type DataSourceType interface {
	String() string
	DeepClone() DataSourceType
	ToDefinitionProto() (*vegapb.DataSourceDefinition, error)
	GetFilters() []*SpecFilter
}

type Timer interface {
	DataSourceType
	IsTriggered(time.Time) bool
	GetTimeTriggers() InternalTimeTriggers
}

type signer interface {
	oneOfProto() interface{}
	DeepClone() signer
	GetSignerType() SignerType
	AsHex(bool) (signer, error)
	AsString() (signer, error)
	String() string
	IsEmpty() bool
}
