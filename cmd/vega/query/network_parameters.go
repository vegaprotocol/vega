package query

import (
	"errors"
	"fmt"

	coreapipb "code.vegaprotocol.io/protos/vega/coreapi/v1"

	"github.com/golang/protobuf/jsonpb"
)

type NetworkParametersCmd struct {
	NodeAddress string `long:"node-address" description:"The address of the vega node to use" default:"0.0.0.0:3002"`
}

func (opts *NetworkParametersCmd) Execute(params []string) error {
	if len(params) > 1 {
		return errors.New("only one network parameter key can be to be specified")
	}

	var key string
	if len(params) == 1 {
		key = params[0]
	}

	req := coreapipb.ListNetworkParametersRequest{
		NetworkParameterKey: key,
	}
	return getPrintNetworkParameters(opts.NodeAddress, &req)
}

func getPrintNetworkParameters(nodeAddress string, req *coreapipb.ListNetworkParametersRequest) error {
	clt, err := getClient(nodeAddress)
	if err != nil {
		return fmt.Errorf("could not connect to the vega node: %w", err)
	}

	ctx, cancel := timeoutContext()
	defer cancel()
	res, err := clt.ListNetworkParameters(ctx, req)
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
