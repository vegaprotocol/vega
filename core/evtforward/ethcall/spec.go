package ethcall

import (
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/vega/protos/vega"

	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/types/known/structpb"
)

type Spec struct {
	Call
	Trigger
	Filter
	Normaliser
	requiredConfirmations uint64
}

func NewSpec(call Call, trigger Trigger, filter Filter, normaliser Normaliser) Spec {
	return Spec{
		Call:       call,
		Trigger:    trigger,
		Filter:     filter,
		Normaliser: normaliser,
	}
}

func (s Spec) RequiredConfirmations() uint64 {
	return s.requiredConfirmations
}

func (s Spec) Hash() []byte {
	hashFunc := sha3.New256()
	hashFunc.Write(s.Call.Hash())
	hashFunc.Write(s.Trigger.Hash())
	hashFunc.Write(s.Filter.Hash())
	return hashFunc.Sum(nil)
}

func (s Spec) HashHex() string {
	return hex.EncodeToString(s.Hash())
}

func (s Spec) ToProto() (*vega.DataSourceDefinition, error) {
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

	return &vega.DataSourceDefinition{
		SourceType: &vega.DataSourceDefinition_External{
			External: &vega.DataSourceDefinitionExternal{
				SourceType: &vega.DataSourceDefinitionExternal_EthCall{
					EthCall: &vega.EthCallSpec{
						Address:               s.address.Hex(),
						Abi:                   &abiPBList,
						Method:                s.method,
						Args:                  argsPBValue,
						Trigger:               s.Trigger.ToProto(),
						Filter:                s.Filter.ToProto(),
						Normaliser:            s.Normaliser.ToProto(),
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

	ethCallProto := externalProto.GetEthCall()
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

	trigger, err := TriggerFromProto(ethCallProto.Trigger)
	if err != nil {
		return Spec{}, fmt.Errorf("unable to create trigger: %w", err)
	}

	filter, err := FilterFromProto(ethCallProto.Filter)
	if err != nil {
		return Spec{}, fmt.Errorf("unable to create filter: %w", err)
	}

	normaliser, err := NormaliserFromProto(ethCallProto.Normaliser)
	if err != nil {
		return Spec{}, fmt.Errorf("unable to create filter: %w", err)
	}

	return Spec{
		Call:                  call,
		Trigger:               trigger,
		Filter:                filter,
		Normaliser:            normaliser,
		requiredConfirmations: ethCallProto.RequiredConfirmations,
	}, nil
}
