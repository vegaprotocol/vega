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
	"errors"
	"fmt"

	"github.com/jackc/pgtype"
)

var ErrInvalidID = errors.New("Not a valid hex ID (or well known exception)")

type ID string

var wellKnownIds = map[string]string{
	"VOTE":         "00",
	systemOwnerStr: "01",
	noMarketStr:    "02",
	"network":      "03",
	"XYZalpha":     "04",
	"XYZbeta":      "05",
	"XYZdelta":     "06",
	"XYZepsilon":   "07",
	"XYZgamma":     "08",
	"fBTC":         "09",
	"fDAI":         "0a",
	"fEURO":        "0b",
	"fUSDC":        "0c",
}

var wellKnownIdsReversed = map[string]string{
	"00": "VOTE",
	"01": systemOwnerStr,
	"02": noMarketStr,
	"03": "network",
	"04": "XYZalpha",
	"05": "XYZbeta",
	"06": "XYZdelta",
	"07": "XYZepsilon",
	"08": "XYZgamma",
	"09": "fBTC",
	"0a": "fDAI",
	"0b": "fEURO",
	"0c": "fUSDC",
}

func (id *ID) Bytes() ([]byte, error) {
	strID := id.String()
	sub, ok := wellKnownIds[strID]
	if ok {
		strID = sub
	}

	bytes, err := hex.DecodeString(strID)
	if err != nil {
		return nil, fmt.Errorf("decoding '%v': %w", string(id.String()), ErrInvalidID)
	}
	return bytes, nil
}

func (id *ID) Error() error {
	_, err := id.Bytes()
	return err
}

func (id *ID) String() string {
	return string(*id)
}

func (id ID) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	bytes, err := id.Bytes()
	if err != nil {
		return buf, err
	}
	return append(buf, bytes...), nil
}

func (id *ID) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	strID := hex.EncodeToString(src)

	sub, ok := wellKnownIdsReversed[strID]
	if ok {
		strID = sub
	}
	*id = ID(strID)
	return nil
}

func (id *ID) UnmarshalJSON(src []byte) error {
	// Unmarshal ID from pg JSONB which is already a JSON string
	if n := len(src); n > 1 && src[0] == '"' && src[n-1] == '"' {
		*id = ID(string(src[1 : n-1]))
		return nil
	}

	return nil
}
