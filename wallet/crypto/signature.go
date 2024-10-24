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

package crypto

import (
	"crypto"
	"encoding/json"
	"errors"
)

const (
	Ed25519 string = "vega/ed25519"
)

var ErrUnsupportedSignatureAlgorithm = errors.New("unsupported signature algorithm")

type SignatureAlgorithm struct {
	impl signatureAlgorithmImpl
}

type signatureAlgorithmImpl interface {
	Sign(priv crypto.PrivateKey, buf []byte) ([]byte, error)
	Verify(pub crypto.PublicKey, message, sig []byte) (bool, error)
	Name() string
	Version() uint32
}

func NewEd25519() SignatureAlgorithm {
	return SignatureAlgorithm{
		impl: newEd25519(),
	}
}

func NewSignatureAlgorithm(name string, version uint32) (SignatureAlgorithm, error) {
	if name == Ed25519 && version == 1 {
		return NewEd25519(), nil
	}
	return SignatureAlgorithm{}, ErrUnsupportedSignatureAlgorithm
}

func (a *SignatureAlgorithm) Sign(priv crypto.PrivateKey, buf []byte) ([]byte, error) {
	return a.impl.Sign(priv, buf)
}

func (a *SignatureAlgorithm) Verify(pub crypto.PublicKey, message, sig []byte) (bool, error) {
	return a.impl.Verify(pub, message, sig)
}

func (a *SignatureAlgorithm) Name() string {
	return a.impl.Name()
}

func (a *SignatureAlgorithm) Version() uint32 {
	return a.impl.Version()
}

func (a *SignatureAlgorithm) MarshalJSON() ([]byte, error) {
	if a == nil {
		return nil, ErrSignatureIsNil
	}
	return json.Marshal(&jsonAlgorithm{
		Name:    a.Name(),
		Version: a.Version(),
	})
}

func (a *SignatureAlgorithm) UnmarshalJSON(data []byte) error {
	jsonAlgo := &jsonAlgorithm{}
	if err := json.Unmarshal(data, &jsonAlgo); err != nil {
		return err
	}

	algo, err := NewSignatureAlgorithm(jsonAlgo.Name, jsonAlgo.Version)
	if err != nil {
		return err
	}

	*a = algo
	return nil
}

type jsonAlgorithm struct {
	Name    string `json:"name"`
	Version uint32 `json:"version"`
}
