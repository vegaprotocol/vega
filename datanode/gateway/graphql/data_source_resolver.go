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

package gql

import (
	"context"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type myDataSourceDefinitionResolver VegaResolverRoot

func (d myDataSourceDefinitionResolver) SourceType(_ context.Context, obj *vegapb.DataSourceDefinition) (DataSourceKind, error) {
	if obj != nil {
		if obj.SourceType != nil {
			switch dp := obj.SourceType.(type) {
			case *vegapb.DataSourceDefinition_External:
				if dp.External.SourceType != nil {
					return dp.External, nil
				}
			case *vegapb.DataSourceDefinition_Internal:
				if dp.Internal.SourceType != nil {
					return dp.Internal, nil
				}
			}
		}
	}

	return nil, nil
}

type myDataSourceDefinitionExternalResolver VegaResolverRoot

func (de myDataSourceDefinitionExternalResolver) SourceType(_ context.Context, obj *vegapb.DataSourceDefinitionExternal) (ExternalDataSourceKind, error) {
	if obj != nil {
		if obj.SourceType != nil {
			switch tp := obj.SourceType.(type) {
			case *vegapb.DataSourceDefinitionExternal_Oracle:
				if tp.Oracle != nil {
					return tp.Oracle, nil
				}
			case *vegapb.DataSourceDefinitionExternal_EthOracle:
				if tp.EthOracle != nil {
					return tp.EthOracle, nil
				}
			}
		}
	}

	return nil, nil
}

type myDataSourceDefinitionInternalResolver VegaResolverRoot

func (di myDataSourceDefinitionInternalResolver) SourceType(_ context.Context, obj *vegapb.DataSourceDefinitionInternal) (InternalDataSourceKind, error) {
	if obj != nil {
		if obj.SourceType != nil {
			switch tp := obj.SourceType.(type) {
			case *vegapb.DataSourceDefinitionInternal_Time:
				if tp.Time != nil {
					return tp.Time, nil
				}
			case *vegapb.DataSourceDefinitionInternal_TimeTrigger:
				if tp.TimeTrigger != nil {
					return tp.TimeTrigger, nil
				}
			}
		}
	}

	return &vegapb.DataSourceSpecConfigurationTime{}, nil
}
