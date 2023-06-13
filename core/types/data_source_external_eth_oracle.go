package types

import (
	"fmt"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	"google.golang.org/protobuf/types/known/structpb"
)

type EthCallSpec struct {
	Address               string
	AbiJson               []byte
	Method                string
	ArgsJson              []string
	Trigger               EthCallTrigger
	RequiredConfirmations uint64
	Normalisers           map[string]string
	Filters               DataSourceSpecFilters
}

func EthCallSpecFromProto(proto *vegapb.EthCallSpec) (EthCallSpec, error) {
	if proto == nil {
		return EthCallSpec{}, fmt.Errorf("ethereum call spec proto is nil")
	}

	trigger, err := EthCallTriggerFromProto(proto.Trigger)
	if err != nil {
		return EthCallSpec{}, fmt.Errorf("error unmarshalling trigger: %w", err)
	}

	filters := DataSourceSpecFiltersFromProto(proto.Filters)

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

	return EthCallSpec{
		Address:               proto.Address,
		AbiJson:               abiBytes,
		Method:                proto.Method,
		ArgsJson:              jsonArgs,
		Trigger:               trigger,
		RequiredConfirmations: proto.RequiredConfirmations,
		Filters:               filters,
		Normalisers:           proto.Normalisers,
	}, nil
}

func (s EthCallSpec) IntoProto() (*vegapb.EthCallSpec, error) {
	argsPBValue := []*structpb.Value{}
	for _, arg := range s.ArgsJson {
		v := structpb.Value{}
		err := v.UnmarshalJSON([]byte(arg))
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal arg json '%s': %w", arg, err)
		}
		argsPBValue = append(argsPBValue, &v)
	}

	abiPBList := structpb.ListValue{}
	err := abiPBList.UnmarshalJSON(s.AbiJson)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal abi json: %w", err)
	}

	return &vegapb.EthCallSpec{
		Address:               s.Address,
		Abi:                   &abiPBList,
		Method:                s.Method,
		Args:                  argsPBValue,
		Trigger:               s.Trigger.IntoEthCallTriggerProto(),
		RequiredConfirmations: s.RequiredConfirmations,
		Filters:               s.Filters.IntoProto(),
		Normalisers:           s.Normalisers,
	}, nil
}

func (s EthCallSpec) ToDataSourceDefinitionProto() (*vegapb.DataSourceDefinition, error) {
	eth, err := s.IntoProto()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to eth oracle proto: %w", err)
	}

	return &vegapb.DataSourceDefinition{
		SourceType: &vegapb.DataSourceDefinition_External{
			External: &vegapb.DataSourceDefinitionExternal{
				SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
					EthOracle: eth,
				},
			},
		},
	}, nil
}

func (s EthCallSpec) String() string {
	return fmt.Sprintf("ethcallspec(%v, %v, %v, %v, %v, %v, %v)",
		s.Address, s.AbiJson, s.Method, s.ArgsJson, s.Trigger, s.RequiredConfirmations, s.Filters)
}

// Whats the need for this deep clone?
func (s EthCallSpec) DeepClone() dataSourceType {
	clonedNormalisers := make(map[string]string)
	for key, value := range s.Normalisers {
		clonedNormalisers[key] = value
	}

	return EthCallSpec{
		Address:               s.Address,
		AbiJson:               s.AbiJson,
		Method:                s.Method,
		ArgsJson:              append([]string(nil), s.ArgsJson...),
		Trigger:               s.Trigger,
		RequiredConfirmations: s.RequiredConfirmations,
		Filters:               append(DataSourceSpecFilters(nil), s.Filters...),
		Normalisers:           clonedNormalisers,
	}
}
