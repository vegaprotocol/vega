package governance

import (
	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	ErrEmptyNetParamKey   = errors.New("empty network parameter key")
	ErrEmptyNetParamValue = errors.New("empty network parmater value")
)

func validateNetworkParameterUpdate(
	netp NetParams, np *types.NetworkParameter) error {
	if len(np.Key) <= 0 {
		return ErrEmptyNetParamKey
	}

	if len(np.Value) <= 0 {
		return ErrEmptyNetParamValue
	}

	// so we seems to just need to call on validate in here.
	// no need to know what's the parameter really or anything else
	return netp.Validate(np.Key, np.Value)
}
