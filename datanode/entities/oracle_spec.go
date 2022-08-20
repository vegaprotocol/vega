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
	"strings"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	oraclespb "code.vegaprotocol.io/vega/protos/vega/oracles/v1"
)

type _Spec struct{}

type SpecID = ID[_Spec]

type (
	PublicKey  = []byte
	PublicKeys = []PublicKey
)

type OracleSpec struct {
	ID         SpecID
	CreatedAt  time.Time
	UpdatedAt  time.Time
	PublicKeys PublicKeys
	Filters    []Filter
	Status     OracleSpecStatus
	TxHash     TxHash
	VegaTime   time.Time
}

func OracleSpecFromProto(spec *oraclespb.OracleSpec, txHash TxHash, vegaTime time.Time) (*OracleSpec, error) {
	id := SpecID(spec.Id)
	pubKeys, err := decodePublicKeys(spec.PubKeys)
	if err != nil {
		return nil, err
	}

	filters := filtersFromProto(spec.Filters)

	return &OracleSpec{
		ID:         id,
		CreatedAt:  time.Unix(0, spec.CreatedAt),
		UpdatedAt:  time.Unix(0, spec.UpdatedAt),
		PublicKeys: pubKeys,
		Filters:    filters,
		Status:     OracleSpecStatus(spec.Status),
		TxHash:     txHash,
		VegaTime:   vegaTime,
	}, nil
}

func (os *OracleSpec) ToProto() *oraclespb.OracleSpec {
	pubKeys := make([]string, 0, len(os.PublicKeys))

	for _, pk := range os.PublicKeys {
		pubKey := hex.EncodeToString(pk)
		pubKeys = append(pubKeys, pubKey)
	}

	filters := filtersToProto(os.Filters)

	return &oraclespb.OracleSpec{
		Id:        os.ID.String(),
		CreatedAt: os.CreatedAt.UnixNano(),
		UpdatedAt: os.UpdatedAt.UnixNano(),
		PubKeys:   pubKeys,
		Filters:   filters,
		Status:    oraclespb.OracleSpec_Status(os.Status),
	}
}

func (os OracleSpec) Cursor() *Cursor {
	return NewCursor(OracleSpecCursor{os.VegaTime, os.ID}.String())
}

func (os OracleSpec) ToProtoEdge(_ ...any) (*v2.OracleSpecEdge, error) {
	return &v2.OracleSpecEdge{
		Node:   os.ToProto(),
		Cursor: os.Cursor().Encode(),
	}, nil
}

func decodePublicKeys(publicKeys []string) (PublicKeys, error) {
	pkList := make(PublicKeys, 0, len(publicKeys))

	for _, publicKey := range publicKeys {
		publicKey := strings.TrimPrefix(publicKey, "0x")
		pk, err := hex.DecodeString(publicKey)
		if err != nil {
			return nil, fmt.Errorf("cannot decode public key: %s", publicKey)
		}

		pkList = append(pkList, pk)
	}

	return pkList, nil
}

type OracleSpecCursor struct {
	VegaTime time.Time `json:"vegaTime"`
	ID       SpecID    `json:"id"`
}

func (os OracleSpecCursor) String() string {
	bs, err := json.Marshal(os)
	if err != nil {
		panic(fmt.Errorf("could not marshal oracle spec cursor: %w", err))
	}
	return string(bs)
}

func (os *OracleSpecCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), os)
}
