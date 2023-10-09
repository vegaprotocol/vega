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

package wallet

// nolint: interfacebloat
type Wallet interface {
	KeyDerivationVersion() uint32
	Name() string
	SetName(newName string)
	ID() string
	Type() string
	HasPublicKey(pubKey string) bool
	DescribePublicKey(pubKey string) (PublicKey, error)
	DescribeKeyPair(pubKey string) (KeyPair, error)
	ListPublicKeys() []PublicKey
	ListKeyPairs() []KeyPair
	MasterKey() (MasterKeyPair, error)
	GenerateKeyPair(meta []Metadata) (KeyPair, error)
	TaintKey(pubKey string) error
	UntaintKey(pubKey string) error
	AnnotateKey(pubKey string, meta []Metadata) ([]Metadata, error)
	SignAny(pubKey string, data []byte) ([]byte, error)
	VerifyAny(pubKey string, data, sig []byte) (bool, error)
	SignTx(pubKey string, data []byte) (*Signature, error)
	IsIsolated() bool
	IsolateWithKey(pubKey string) (Wallet, error)
	Permissions(hostname string) Permissions
	PermittedHostnames() []string
	RevokePermissions(hostname string)
	PurgePermissions()
	UpdatePermissions(hostname string, perms Permissions) error
	Clone() Wallet
}

// nolint: interfacebloat
type KeyPair interface {
	PublicKey() string
	PrivateKey() string
	Name() string
	IsTainted() bool
	Metadata() []Metadata
	UpdateMetadata([]Metadata) []Metadata
	Index() uint32
	AlgorithmVersion() uint32
	AlgorithmName() string
	SignAny(data []byte) ([]byte, error)
	VerifyAny(data, sig []byte) (bool, error)
	Sign(data []byte) (*Signature, error)
}

type PublicKey interface {
	Key() string
	Name() string
	IsTainted() bool
	Metadata() []Metadata
	Index() uint32
	AlgorithmVersion() uint32
	AlgorithmName() string
	Hash() (string, error)
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}

type MasterKeyPair interface {
	PublicKey() string
	PrivateKey() string
	AlgorithmVersion() uint32
	AlgorithmName() string
	SignAny(data []byte) ([]byte, error)
	Sign(data []byte) (*Signature, error)
}
