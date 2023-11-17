// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package defaults

import (
	"io"

	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/protobuf/jsonpb"
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

func (u *Unmarshaler) UnmarshalLiquiditySLAParams(r io.Reader) (*vegapb.LiquiditySLAParameters, error) {
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

func (u *Unmarshaler) UnmarshalPerpsDataSourceConfig(r io.Reader) (*vegapb.Perpetual, error) {
	proto := &vegapb.Perpetual{}
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

func (u *Unmarshaler) UnmarshalLiquidationConfig(r io.Reader) (*vegapb.LiquidationStrategy, error) {
	proto := &vegapb.LiquidationStrategy{}
	if err := u.unmarshaler.Unmarshal(r, proto); err != nil {
		return nil, err
	}
	return proto, nil
}
