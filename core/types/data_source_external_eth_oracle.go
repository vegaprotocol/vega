package types

import (
	"fmt"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type EthCallSpec struct {
	Address               string
	AbiJson               []byte
	Method                string
	ArgsJson              []string
	Trigger               EthCallTrigger
	RequiredConfirmations uint64
	// Normaliser            normaliser
	Filters DataSourceSpecFilters
}

func EthCallSpecFromProto(proto *vegapb.EthCallSpec) (EthCallSpec, error) {
	if proto == nil {
		return EthCallSpec{}, fmt.Errorf("ethereum call spec proto is nil")
	}

	trigger, err := EthCallTriggerFromProto(proto.Trigger)
	if err != nil {
		return EthCallSpec{}, fmt.Errorf("error unmarshalling trigger: %w", err)
	}

	filters, err := DataSourceSpecFiltersFromProto(proto.Filters)
	if err != nil {
		return EthCallSpec{}, fmt.Errorf("error unmarshalling filters: %w", err)
	}

	abiBytes, err := proto.Abi.MarshalJSON()
	if err != nil {
		return EthCallSpec{}, fmt.Errorf("error marshalling abi: %w", err)
	}

	jsonArgs := []string{}
	for _, protoArg := range proto.Args {
		jsonArg, err := protoArg.MarshalJSON()
		if err != nil {
			return EthCallSpec{}, fmt.Errorf("error marshalling arg: %w", err)
		}
		jsonArgs = append(jsonArgs, string(jsonArg))
	}

	// norm := NormaliserFromProto(proto.EthOracle.Normaliser)

	return EthCallSpec{
		Address:               proto.Address,
		AbiJson:               abiBytes,
		Method:                proto.Method,
		ArgsJson:              jsonArgs,
		Trigger:               trigger,
		RequiredConfirmations: proto.RequiredConfirmations,
		Filters:               filters,
	}, nil
}

func (s EthCallSpec) IntoProto() *vegapb.EthCallSpec {
	ecs := &vegapb.EthCallSpec{}
	/*
		if s != nil {
			ecs.Address = s.Address
			if s.AbiJson != "" {
			}
			ecs.Method = s.Method

			if s.ArgsJson != nil {
			}

			if s.Trigger != nil {
			}

			ecs.RequiredConfirmations = s.RequiredConfirmations
			if s.Filter != nil {
			}

			if s.Normaliser != nil {
			}
		}*/
	return ecs
}

func (s EthCallSpec) ToDataSourceDefinitionProto() *vegapb.DataSourceDefinition {
	return &vegapb.DataSourceDefinition{
		SourceType: &vegapb.DataSourceDefinition_External{
			External: &vegapb.DataSourceDefinitionExternal{
				SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
					EthOracle: s.IntoProto(),
				},
			},
		},
	}
}

func (s EthCallSpec) String() string {
	return "todo"
}

func (s EthCallSpec) DeepClone() dataSourceType {
	return EthCallSpec{
		Address:               s.Address,
		AbiJson:               s.AbiJson,
		Method:                s.Method,
		ArgsJson:              append([]string(nil), s.ArgsJson...),
		Trigger:               s.Trigger,
		RequiredConfirmations: s.RequiredConfirmations,
		Filters:               append(DataSourceSpecFilters(nil), s.Filters...),
	}
}
