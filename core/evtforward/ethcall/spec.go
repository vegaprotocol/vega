package ethcall

import (
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/types/known/structpb"
)

type Spec struct {
	Call
	types.EthCallTrigger
	types.EthFilter
	types.Normaliser
	requiredConfirmations uint64
}

func NewSpec(call Call, trigger types.EthCallTrigger, filter types.EthFilter, normaliser types.Normaliser, requiredConfirmations uint64) Spec {
	return Spec{
		Call:                  call,
		EthCallTrigger:        trigger,
		EthFilter:             filter,
		Normaliser:            normaliser,
		requiredConfirmations: requiredConfirmations,
	}
}

func (s Spec) RequiredConfirmations() uint64 {
	return s.requiredConfirmations
}

func (s Spec) Hash() []byte {
	hashFunc := sha3.New256()
	hashFunc.Write(s.Call.Hash())
	hashFunc.Write(s.EthCallTrigger.Hash())
	hashFunc.Write(s.EthFilter.Hash())
	hashFunc.Write([]byte(fmt.Sprintf("requiredconfirmations: %v", s.requiredConfirmations)))

	return hashFunc.Sum(nil)
}

func (s Spec) HashHex() string {
	return hex.EncodeToString(s.Hash())
}

func (s Spec) ToProto() (*vegapb.DataSourceDefinition, error) {
	args, err := s.Args()
	if err != nil {
		return nil, fmt.Errorf("failed to get eth call args: %w", err)
	}

	jsonArgs, err := AnyArgsToJson(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal eth call args: %w", err)
	}

	argsPBValue := []*structpb.Value{}
	for _, arg := range jsonArgs {
		v := structpb.Value{}
		err := v.UnmarshalJSON([]byte(arg))
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal arg json '%s': %w", arg, err)
		}
		argsPBValue = append(argsPBValue, &v)
	}

	abiPBList := structpb.ListValue{}
	err = abiPBList.UnmarshalJSON(s.abiJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal abi json: %w", err)
	}

	f, err := s.EthFilter.IntoProto()
	if err != nil {
		return nil, err
	}
	return &vegapb.DataSourceDefinition{
		SourceType: &vegapb.DataSourceDefinition_External{
			External: &vegapb.DataSourceDefinitionExternal{
				SourceType: &vegapb.DataSourceDefinitionExternal_EthOracle{
					EthOracle: &vegapb.EthCallSpec{
						Address:               s.address.Hex(),
						Abi:                   &abiPBList,
						Method:                s.method,
						Args:                  argsPBValue,
						Trigger:               s.EthCallTrigger.IntoProto(),
						Filter:                f,
						Normaliser:            s.Normaliser.IntoProto(),
						RequiredConfirmations: s.RequiredConfirmations(),
					},
				},
			},
		},
	}, nil
}

func NewSpecFromProto(proto *vega.DataSourceDefinition) (Spec, error) {
	if proto == nil {
		return Spec{}, fmt.Errorf("null data source definition")
	}

	externalProto := proto.GetExternal()
	if externalProto == nil {
		return Spec{}, fmt.Errorf("not an external data source")
	}

	ethCallProto := externalProto.GetEthOracle()
	if ethCallProto == nil {
		return Spec{}, fmt.Errorf("not an eth call data source")
	}

	// Get args out of proto 'struct' format into JSON
	jsonArgs := []string{}
	for _, protoArg := range ethCallProto.Args {
		jsonArg, err := protoArg.MarshalJSON()
		if err != nil {
			return Spec{}, fmt.Errorf("unable to marshal args from proto to json: %w", err)
		}
		jsonArgs = append(jsonArgs, string(jsonArg))
	}

	abiJson, err := ethCallProto.Abi.MarshalJSON()
	if err != nil {
		return Spec{}, fmt.Errorf("unable to marshal abi: %w", err)
	}

	// Convert JSON args to go types using ABI
	args, err := JsonArgsToAny(ethCallProto.Method, jsonArgs, string(abiJson))
	if err != nil {
		return Spec{}, fmt.Errorf("unable to deserialize args: %w", err)
	}

	call, err := NewCall(ethCallProto.Method, args, ethCallProto.Address, abiJson)
	if err != nil {
		return Spec{}, fmt.Errorf("unable to create call: %w", err)
	}

	trigger := types.EthCallTriggerFromProto(ethCallProto.Trigger)
	//if err != nil {
	//	return Spec{}, fmt.Errorf("unable to create trigger: %w", err)
	//}

	filter, err := types.EthFilterFromProto(ethCallProto.Filter)
	if err != nil {
		return Spec{}, fmt.Errorf("unable to create filter: %w", err)
	}

	normaliser, err := types.NormaliserFromProto(ethCallProto.Normaliser)
	if err != nil {
		return Spec{}, fmt.Errorf("unable to create filter: %w", err)
	}

	return Spec{
		Call:                  call,
		EthCallTrigger:        *trigger,
		EthFilter:             *filter,
		Normaliser:            *normaliser,
		requiredConfirmations: ethCallProto.RequiredConfirmations,
	}, nil
}
