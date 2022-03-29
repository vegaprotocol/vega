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
}

var wellKnownIdsReversed = map[string]string{
	"00": "VOTE",
	"01": systemOwnerStr,
	"02": noMarketStr,
	"03": "network",
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
