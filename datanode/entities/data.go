// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type Property struct {
	NumberDecimalPlaces *uint64
	Name                string
	Value               string
}

type Data struct {
	BroadcastAt    time.Time
	VegaTime       time.Time
	TxHash         TxHash
	Signers        Signers
	Data           []Property
	MatchedSpecIds [][]byte // pgx automatically handles [][]byte to Postgres ByteaArray mappings
	SeqNum         uint64
}

type ExternalData struct {
	Data *Data
}

func ExternalDataFromProto(data *datapb.ExternalData, txHash TxHash, vegaTime time.Time, seqNum uint64) (*ExternalData, error) {
	properties := []Property{}
	specIDs := [][]byte{}
	signers := Signers{}

	if data.Data != nil {
		properties = make([]Property, 0, len(data.Data.Data))
		specIDs = make([][]byte, 0, len(data.Data.MatchedSpecIds))

		for _, property := range data.Data.Data {
			properties = append(properties, Property{
				Name:                property.Name,
				Value:               property.Value,
				NumberDecimalPlaces: property.NumberDecimalPlaces,
			})
		}

		for _, specID := range data.Data.MatchedSpecIds {
			id := SpecID(specID)
			idBytes, err := id.Bytes()
			if err != nil {
				return nil, fmt.Errorf("cannot decode spec ID: %w", err)
			}
			specIDs = append(specIDs, idBytes)
		}

		var err error
		signers, err = SerializeSigners(types.SignersFromProto(data.Data.Signers))
		if err != nil {
			return nil, err
		}
	}

	return &ExternalData{
		Data: &Data{
			Signers:        signers,
			Data:           properties,
			MatchedSpecIds: specIDs,
			BroadcastAt:    NanosToPostgresTimestamp(data.Data.BroadcastAt),
			TxHash:         txHash,
			VegaTime:       vegaTime,
			SeqNum:         seqNum,
		},
	}, nil
}

func (od *ExternalData) ToProto() *datapb.ExternalData {
	properties := []*datapb.Property{}
	specIDs := []string{}
	signersAsProto := []*datapb.Signer{}

	if od.Data != nil {
		if od.Data.Data != nil {
			properties = make([]*datapb.Property, 0, len(od.Data.Data))
			specIDs = make([]string, 0, len(od.Data.MatchedSpecIds))

			for _, prop := range od.Data.Data {
				properties = append(properties, &datapb.Property{
					Name:                prop.Name,
					Value:               prop.Value,
					NumberDecimalPlaces: prop.NumberDecimalPlaces,
				})
			}

			for _, id := range od.Data.MatchedSpecIds {
				hexID := hex.EncodeToString(id)
				specIDs = append(specIDs, hexID)
			}
		}

		signers := DeserializeSigners(od.Data.Signers)
		signersAsProto = types.SignersIntoProto(signers)
	}

	return &datapb.ExternalData{
		Data: &datapb.Data{
			Signers:        signersAsProto,
			Data:           properties,
			MatchedSpecIds: specIDs,
			BroadcastAt:    od.Data.BroadcastAt.UnixNano(),
		},
	}
}

func (od ExternalData) ToOracleProto() *vegapb.OracleData {
	return &vegapb.OracleData{
		ExternalData: od.ToProto(),
	}
}

func (od ExternalData) Cursor() *Cursor {
	return NewCursor(OracleDataCursor{
		VegaTime: od.Data.VegaTime,
		Signers:  od.Data.Signers,
	}.String())
}

func (od ExternalData) ToOracleProtoEdge(_ ...any) (*v2.OracleDataEdge, error) {
	return &v2.OracleDataEdge{
		Node:   od.ToOracleProto(),
		Cursor: od.Cursor().Encode(),
	}, nil
}

type ExternalDataCursor struct {
	VegaTime time.Time `json:"vegaTime"`
	Signers  Signers   `json:"signers"`
}

func (c ExternalDataCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		// This really shouldn't happen.
		panic(fmt.Errorf("couldn't marshal oracle data cursor: %w", err))
	}

	return string(bs)
}

func (c *ExternalDataCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), c)
}
