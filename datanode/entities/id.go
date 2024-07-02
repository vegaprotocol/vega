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
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgtype"
)

var (
	ErrInvalidID = errors.New("not a valid hex id (or well known exception)")
	ErrNotFound  = errors.New("no resource corresponding to this id")
)

type ID[T any] string

var wellKnownIds = map[string]string{
	"VOTE":       "00",
	"network":    "03",
	"XYZalpha":   "04",
	"XYZbeta":    "05",
	"XYZdelta":   "06",
	"XYZepsilon": "07",
	"XYZgamma":   "08",
	"fBTC":       "09",
	"fDAI":       "0a",
	"fEURO":      "0b",
	"fUSDC":      "0c",
}

var wellKnownIdsReversed = map[string]string{
	"00": "VOTE",
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

func (id *ID[T]) Bytes() ([]byte, error) {
	strID := id.String()
	sub, ok := wellKnownIds[strID]
	if ok {
		strID = sub
	}

	bytes, err := hex.DecodeString(strID)
	if err != nil {
		return nil, fmt.Errorf("decoding '%v': %w", id.String(), ErrInvalidID)
	}
	return bytes, nil
}

func (id *ID[T]) SetBytes(src []byte) error {
	strID := hex.EncodeToString(src)

	sub, ok := wellKnownIdsReversed[strID]
	if ok {
		strID = sub
	}
	*id = ID[T](strID)
	return nil
}

func (id *ID[T]) Error() error {
	_, err := id.Bytes()
	return err
}

func (id *ID[T]) String() string {
	if id == nil {
		return ""
	}

	return string(*id)
}

func (id ID[T]) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	bytes, err := id.Bytes()
	if err != nil {
		return buf, err
	}
	return append(buf, bytes...), nil
}

func (id *ID[T]) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	return id.SetBytes(src)
}

func (id *ID[T]) Where(fieldName *string, nextBindVar func(args *[]any, arg any) string, args ...any) (string, []any) {
	if fieldName == nil {
		return fmt.Sprintf("id = %s", nextBindVar(&args, id)), args
	}
	return fmt.Sprintf("%s = %s", *fieldName, nextBindVar(&args, id)), args
}
