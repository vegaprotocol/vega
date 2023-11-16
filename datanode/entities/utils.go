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
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgtype"
)

type VegaPublicKey string

func (pk *VegaPublicKey) Bytes() ([]byte, error) {
	strPK := pk.String()

	bytes, err := hex.DecodeString(strPK)
	if err != nil {
		return nil, fmt.Errorf("decoding '%v': %w", pk.String(), ErrInvalidID)
	}
	return bytes, nil
}

func (pk *VegaPublicKey) Error() error {
	_, err := pk.Bytes()
	return err
}

func (pk *VegaPublicKey) String() string {
	return string(*pk)
}

func (pk VegaPublicKey) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	bytes, err := pk.Bytes()
	if err != nil {
		return buf, err
	}
	return append(buf, bytes...), nil
}

func (pk *VegaPublicKey) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	strPK := hex.EncodeToString(src)

	*pk = VegaPublicKey(strPK)
	return nil
}

type TendermintPublicKey string

func (pk *TendermintPublicKey) Bytes() ([]byte, error) {
	strPK := pk.String()

	bytes, err := base64.StdEncoding.DecodeString(strPK)
	if err != nil {
		return nil, fmt.Errorf("decoding '%v': %w", pk.String(), ErrInvalidID)
	}
	return bytes, nil
}

func (pk *TendermintPublicKey) Error() error {
	_, err := pk.Bytes()
	return err
}

func (pk *TendermintPublicKey) String() string {
	return string(*pk)
}

func (pk TendermintPublicKey) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	bytes, err := pk.Bytes()
	if err != nil {
		return buf, err
	}
	return append(buf, bytes...), nil
}

func (pk *TendermintPublicKey) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	strPK := base64.StdEncoding.EncodeToString(src)

	*pk = TendermintPublicKey(strPK)
	return nil
}

type EthereumAddress string

func (addr *EthereumAddress) Bytes() ([]byte, error) {
	strAddr := addr.String()

	if !strings.HasPrefix(strAddr, "0x") {
		return nil, fmt.Errorf("invalid '%v': %w", addr.String(), ErrInvalidID)
	}

	bytes, err := hex.DecodeString(strAddr[2:])
	if err != nil {
		return nil, fmt.Errorf("decoding '%v': %w", addr.String(), ErrInvalidID)
	}
	return bytes, nil
}

func (addr *EthereumAddress) Error() error {
	_, err := addr.Bytes()
	return err
}

func (addr *EthereumAddress) String() string {
	return string(*addr)
}

func (addr EthereumAddress) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	bytes, err := addr.Bytes()
	if err != nil {
		return buf, err
	}
	return append(buf, bytes...), nil
}

func (addr *EthereumAddress) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	strAddr := "0x" + hex.EncodeToString(src)

	*addr = EthereumAddress(strAddr)
	return nil
}

type TxHash string

func (h *TxHash) Bytes() ([]byte, error) {
	strPK := h.String()

	bytes, err := hex.DecodeString(strPK)
	if err != nil {
		return nil, fmt.Errorf("decoding '%v': %w", h.String(), ErrInvalidID)
	}
	return bytes, nil
}

func (h *TxHash) Error() error {
	_, err := h.Bytes()
	return err
}

func (h *TxHash) String() string {
	return string(*h)
}

func (h TxHash) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	bytes, err := h.Bytes()
	if err != nil {
		return buf, err
	}
	return append(buf, bytes...), nil
}

func (h *TxHash) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	*h = TxHash(hex.EncodeToString(src))
	return nil
}

// NanosToPostgresTimestamp postgres stores timestamps in microsecond resolution.
func NanosToPostgresTimestamp(nanos int64) time.Time {
	return time.Unix(0, nanos).Truncate(time.Microsecond)
}
