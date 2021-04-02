package defaults

import (
	"io"

	"github.com/golang/protobuf/jsonpb"

	types "code.vegaprotocol.io/vega/proto"
)

type Unmarshaler struct {
	unmarshaler jsonpb.Unmarshaler
}

func NewUnmarshaler() *Unmarshaler {
	return &Unmarshaler{}
}

// UnmarshalRiskModel unmarshal a tradable instrument instead of a risk model since
// gRPC implementation of risk models can't be used with jsonpb.Unmarshaler.
func (u *Unmarshaler) UnmarshalRiskModel(r io.Reader) (*types.TradableInstrument, error) {
	proto := &types.TradableInstrument{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

func (u *Unmarshaler) UnmarshalPriceMonitoring(r io.Reader) (*types.PriceMonitoringSettings, error) {
	proto := &types.PriceMonitoringSettings{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

// UnmarshalOracleConfig unmarshal a future as this is a common parent
func (u *Unmarshaler) UnmarshalOracleConfig(r io.Reader) (*types.Future, error) {
	proto := &types.Future{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

func (u *Unmarshaler) UnmarshalMarginCalculator(r io.Reader) (*types.MarginCalculator, error) {
	proto := &types.MarginCalculator{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

func (u *Unmarshaler) UnmarshalFeesConfig(r io.Reader) (*types.Fees, error) {
	proto := &types.Fees{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}
