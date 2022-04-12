package entities

import (
	"encoding/hex"
	"fmt"
	"time"

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
		BroadcastAt:    time.Unix(0, data.BroadcastAt),
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
