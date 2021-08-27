package query

import (
	"fmt"

	coreapipb "code.vegaprotocol.io/protos/vega/coreapi/v1"

	"github.com/golang/protobuf/jsonpb"
)

type ValidatorsCmd struct {
	NodeAddress string `long:"node-address" description:"The address of the vega node to use" default:"0.0.0.0:3002"`
}

func (opts *ValidatorsCmd) Execute(params []string) error {
	req := coreapipb.ListValidatorsRequest{}
	return getPrintValidators(opts.NodeAddress, &req)
}

func getPrintValidators(nodeAddress string, req *coreapipb.ListValidatorsRequest) error {
	clt, err := getClient(nodeAddress)
	if err != nil {
		return fmt.Errorf("could not connect to the vega node: %w", err)
	}

	ctx, cancel := timeoutContext()
	defer cancel()
	res, err := clt.ListValidators(ctx, req)
	if err != nil {
		return fmt.Errorf("error querying the vega node: %w", err)
	}

	m := jsonpb.Marshaler{
		Indent: "  ",
	}
	buf, err := m.MarshalToString(res)
	if err != nil {
		return fmt.Errorf("invalid response from vega node: %w", err)
	}

	fmt.Printf("%v", string(buf))

	return nil
}
