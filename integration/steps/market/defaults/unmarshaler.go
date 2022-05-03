package defaults

import (
	"io"

	"github.com/golang/protobuf/jsonpb"

	vegapb "code.vegaprotocol.io/protos/vega"
)

type Unmarshaler struct {
	unmarshaler jsonpb.Unmarshaler
}

func NewUnmarshaler() *Unmarshaler {
	return &Unmarshaler{}
}

// UnmarshalRiskModel unmarshal a tradable instrument instead of a risk model since
// gRPC implementation of risk models can't be used with jsonpb.Unmarshaler.
func (u *Unmarshaler) UnmarshalRiskModel(r io.Reader) (*vegapb.TradableInstrument, error) {
	proto := &vegapb.TradableInstrument{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

func (u *Unmarshaler) UnmarshalPriceMonitoring(r io.Reader) (*vegapb.PriceMonitoringSettings, error) {
	proto := &vegapb.PriceMonitoringSettings{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

// UnmarshalOracleConfig unmarshal a future as this is a common parent.
func (u *Unmarshaler) UnmarshalOracleConfig(r io.Reader) (*vegapb.Future, error) {
	proto := &vegapb.Future{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

func (u *Unmarshaler) UnmarshalMarginCalculator(r io.Reader) (*vegapb.MarginCalculator, error) {
	proto := &vegapb.MarginCalculator{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

func (u *Unmarshaler) UnmarshalFeesConfig(r io.Reader) (*vegapb.Fees, error) {
	proto := &vegapb.Fees{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}
