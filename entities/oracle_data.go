// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
)

type Property struct {
	Name  string
	Value string
}

type OracleData struct {
	PublicKeys     PublicKeys
	Data           []Property
	MatchedSpecIds [][]byte // pgx automatically handles [][]byte to Postgres ByteaArray mappings
	BroadcastAt    time.Time
	VegaTime       time.Time
}

func OracleDataFromProto(data *oraclespb.OracleData, vegaTime time.Time) (*OracleData, error) {
	properties := make([]Property, 0, len(data.Data))
	specIDs := make([][]byte, 0, len(data.MatchedSpecIds))

	pubKeys, err := decodePublicKeys(data.PubKeys)
	if err != nil {
		return nil, err
	}

	for _, property := range data.Data {
		properties = append(properties, Property{
			Name:  property.Name,
			Value: property.Value,
		})
	}

	for _, specID := range data.MatchedSpecIds {
		id := NewSpecID(specID)
		idBytes, err := id.Bytes()
		if err != nil {
			return nil, fmt.Errorf("cannot decode spec ID: %w", err)
		}
		specIDs = append(specIDs, idBytes)
	}

	return &OracleData{
		PublicKeys:     pubKeys,
		Data:           properties,
		MatchedSpecIds: specIDs,
		BroadcastAt:    NanosToPostgresTimestamp(data.BroadcastAt),
		VegaTime:       vegaTime,
	}, nil
}

func (od *OracleData) ToProto() *oraclespb.OracleData {
	pubKeys := make([]string, 0, len(od.PublicKeys))
	data := make([]*oraclespb.Property, 0, len(od.Data))
	specIDs := make([]string, 0, len(od.MatchedSpecIds))

	for _, pk := range od.PublicKeys {
		pubKeys = append(pubKeys, hex.EncodeToString(pk))
	}

	for _, prop := range od.Data {
		data = append(data, &oraclespb.Property{
			Name:  prop.Name,
			Value: prop.Value,
		})
	}

	for _, id := range od.MatchedSpecIds {
		hexID := hex.EncodeToString(id)
		specIDs = append(specIDs, hexID)
	}

	return &oraclespb.OracleData{
		PubKeys:        pubKeys,
		Data:           data,
		MatchedSpecIds: specIDs,
		BroadcastAt:    od.BroadcastAt.UnixNano(),
	}
}

func (od OracleData) Cursor() *Cursor {
	pks := make([]string, 0, len(od.PublicKeys))

	for _, pk := range od.PublicKeys {
		pks = append(pks, hex.EncodeToString(pk))
	}

	return NewCursor(OracleDataCursor{
		VegaTime:   od.VegaTime,
		PublicKeys: pks,
	}.String())
}

func (od OracleData) ToProtoEdge(_ ...any) *v2.OracleDataEdge {
	return &v2.OracleDataEdge{
		Node:   od.ToProto(),
		Cursor: od.Cursor().Encode(),
	}
}

type OracleDataCursor struct {
	VegaTime   time.Time `json:"vegaTime"`
	PublicKeys []string  `json:"publicKeys"`
}

func (c OracleDataCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		// This really shouldn't happen.
		panic(fmt.Errorf("couldn't marshal oracle data cursor: %w", err))
	}

	return string(bs)
}

func (c *OracleDataCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), c)
}
