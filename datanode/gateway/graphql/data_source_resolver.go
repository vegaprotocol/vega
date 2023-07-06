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
			}
		}
	}

	return &vegapb.DataSourceSpecConfiguration{}, nil
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
			}
		}
	}

	return &vegapb.DataSourceSpecConfigurationTime{}, nil
}
