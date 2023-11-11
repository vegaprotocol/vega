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

package entities

import (
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type OracleData struct {
	ExternalData *ExternalData
}

func OracleDataFromProto(data *vegapb.OracleData, txHash TxHash, vegaTime time.Time, seqNum uint64) (*OracleData, error) {
	extData, err := ExternalDataFromProto(data.ExternalData, txHash, vegaTime, seqNum)
	if err != nil {
		return nil, err
	}

	return &OracleData{
		ExternalData: extData,
	}, nil
}

func (od OracleData) ToProto() *vegapb.OracleData {
	return &vegapb.OracleData{
		ExternalData: od.ExternalData.ToProto(),
	}
}

func (od OracleData) Cursor() *Cursor {
	return od.ExternalData.Cursor()
}

func (od OracleData) ToProtoEdge(_ ...any) (*v2.OracleDataEdge, error) {
	tp, err := od.ExternalData.ToOracleProtoEdge()
	if err != nil {
		return nil, err
	}

	return tp, nil
}

type OracleDataCursor = ExternalDataCursor
