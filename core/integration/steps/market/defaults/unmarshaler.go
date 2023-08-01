// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package defaults

import (
	"io"

	"github.com/golang/protobuf/jsonpb"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
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

func (u *Unmarshaler) UnmarshalLliquiditySLAParams(r io.Reader) (*vegapb.LiquiditySLAParameters, error) {
	proto := &vegapb.LiquiditySLAParameters{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

func (u *Unmarshaler) UnmarshalLiquidityMonitoring(r io.Reader) (*vegapb.LiquidityMonitoringParameters, error) {
	proto := &vegapb.LiquidityMonitoringParameters{}
	if err := u.unmarshaler.Unmarshal(r, proto); err != nil {
		return nil, err
	}
	return proto, nil
}

func (u *Unmarshaler) UnmarshalPerpsDataSourceConfig(r io.Reader) (*vegapb.Perps, error) {
	proto := &vegapb.Perps{}
	err := u.unmarshaler.Unmarshal(r, proto)
	if err != nil {
		return nil, err
	}
	return proto, nil
}

// UnmarshalDataSourceConfig unmarshal a future as this is a common parent.
func (u *Unmarshaler) UnmarshalDataSourceConfig(r io.Reader) (*vegapb.Future, error) {
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
