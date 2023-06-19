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

type (
	_Spec  struct{}
	SpecID = ID[_Spec]
)

type (
	Signer  []byte
	Signers = []Signer
)

type Property struct {
	Name  string
	Value string
}

func SerializeSigners(signers []*types.Signer) (Signers, error) {
	if len(signers) > 0 {
		sigList := Signers{}

		for _, signer := range signers {
			data, err := signer.Serialize()
			if err != nil {
				return nil, err
			}
			sigList = append(sigList, data)
		}

		return sigList, nil
	}

	return Signers{}, nil
}

func DeserializeSigners(data Signers) []*types.Signer {
	if len(data) > 0 {
		signers := []*types.Signer{}
		for _, s := range data {
			signer := types.DeserializeSigner(s)
			signers = append(signers, signer)
		}

		return signers
	}

	return nil
}

type Data struct {
	Signers        Signers
	Data           []Property
	MetaData       []Property
	MatchedSpecIds [][]byte // pgx automatically handles [][]byte to Postgres ByteaArray mappings
	BroadcastAt    time.Time
	TxHash         TxHash
	VegaTime       time.Time
	SeqNum         uint64
}

type ExternalData struct {
	Data *Data
}

func ExternalDataFromProto(data *datapb.ExternalData, txHash TxHash, vegaTime time.Time, seqNum uint64) (*ExternalData, error) {
	properties := []Property{}
	specIDs := [][]byte{}
	signers := Signers{}
	var metaDataProperties []Property

	if data.Data != nil {
		properties = make([]Property, 0, len(data.Data.Data))
		specIDs = make([][]byte, 0, len(data.Data.MatchedSpecIds))

		for _, property := range data.Data.Data {
			properties = append(properties, Property{
				Name:  property.Name,
				Value: property.Value,
			})
		}

		if len(data.Data.MetaData) > 0 {
			metaDataProperties = make([]Property, 0, len(data.Data.MetaData))

			for _, m := range data.Data.MetaData {
				metaDataProperties = append(metaDataProperties, Property{
					Name:  m.Name,
					Value: m.Value,
				})
			}
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
			MetaData:       metaDataProperties,
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
	metaDataProperties := []*datapb.Property{}

	if od.Data != nil {
		if od.Data.Data != nil {
			properties = make([]*datapb.Property, 0, len(od.Data.Data))
			specIDs = make([]string, 0, len(od.Data.MatchedSpecIds))
			metaDataProperties = make([]*datapb.Property, 0, len(od.Data.MetaData))

			for _, prop := range od.Data.Data {
				properties = append(properties, &datapb.Property{
					Name:  prop.Name,
					Value: prop.Value,
				})
			}

			for _, m := range od.Data.MetaData {
				metaDataProperties = append(metaDataProperties, &datapb.Property{
					Name:  m.Name,
					Value: m.Value,
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
			MetaData:       metaDataProperties,
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

type Condition struct {
	Operator datapb.Condition_Operator
	Value    string
}

func (c Condition) ToProto() *datapb.Condition {
	return &datapb.Condition{
		Operator: c.Operator,
		Value:    c.Value,
	}
}

func ConditionFromProto(protoCondition *datapb.Condition) Condition {
	return Condition{
		Operator: protoCondition.Operator,
		Value:    protoCondition.Value,
	}
}

type PropertyKey struct {
	Name          string `json:"name"`
	Type          datapb.PropertyKey_Type
	DecimalPlaces *uint64 `json:"number_decimal_places,omitempty"`
}

type Filter struct {
	Key        PropertyKey `json:"key"`
	Conditions []Condition `json:"conditions"`
}

func (f Filter) ToProto() *datapb.Filter {
	conditions := make([]*datapb.Condition, 0, len(f.Conditions))
	for _, condition := range f.Conditions {
		conditions = append(conditions, condition.ToProto())
	}

	var ndp *uint64
	if f.Key.DecimalPlaces != nil {
		v := *f.Key.DecimalPlaces
		ndp = &v
	}

	return &datapb.Filter{
		Key: &datapb.PropertyKey{
			Name:                f.Key.Name,
			Type:                f.Key.Type,
			NumberDecimalPlaces: ndp,
		},
		Conditions: conditions,
	}
}

func FiltersFromProto(filters []*datapb.Filter) []Filter {
	if len(filters) == 0 {
		return nil
	}

	results := make([]Filter, 0, len(filters))
	for _, filter := range filters {
		conditions := make([]Condition, 0, len(filter.Conditions))

		for _, condition := range filter.Conditions {
			conditions = append(conditions, Condition{
				Operator: condition.Operator,
				Value:    condition.Value,
			})
		}

		var ndp *uint64
		if filter.Key.NumberDecimalPlaces != nil {
			v := *filter.Key.NumberDecimalPlaces
			ndp = &v
		}
		results = append(results, Filter{
			Key: PropertyKey{
				Name:          filter.Key.Name,
				Type:          filter.Key.Type,
				DecimalPlaces: ndp,
			},
			Conditions: conditions,
		})
	}

	return results
}
