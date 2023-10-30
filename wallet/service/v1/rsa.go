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

package v1

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

type RSAKeys struct {
	Pub  []byte
	Priv []byte
}

func GenerateRSAKeys() (*RSAKeys, error) {
	bitSize := 4096

	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate RSA keys: %w", err)
	}

	privateKey, err := toPrivatePKCS1Key(key)
	if err != nil {
		return nil, fmt.Errorf("couldn't extract private RSA key: %w", err)
	}

	publicKey, err := toPublicPKCS1Key(key)
	if err != nil {
		return nil, fmt.Errorf("couldn't extract public RSA key: %w", err)
	}

	return &RSAKeys{
		Pub:  publicKey,
		Priv: privateKey,
	}, nil
}

func toPrivatePKCS1Key(key *rsa.PrivateKey) ([]byte, error) {
	privateKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	privateKeyBuffer := bytes.NewBuffer([]byte{})
	err := pem.Encode(privateKeyBuffer, privateKey)
	if err != nil {
		return nil, fmt.Errorf("couldn't encode private RSA key: %w", err)
	}
	return privateKeyBuffer.Bytes(), nil
}

func toPublicPKCS1Key(key *rsa.PrivateKey) ([]byte, error) {
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal public RSA key: %w", err)
	}
	publicKey := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	publicKeyBuffer := bytes.NewBuffer([]byte{})
	err = pem.Encode(publicKeyBuffer, publicKey)
	if err != nil {
		return nil, fmt.Errorf("couldn't encode public RSA key: %w", err)
	}
	return publicKeyBuffer.Bytes(), nil
}
