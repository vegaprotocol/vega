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

import (
	"sort"
)

type HDKeyRing struct {
	keys      map[string]HDKeyPair
	nextIndex uint32
}

func NewHDKeyRing() *HDKeyRing {
	return &HDKeyRing{
		keys:      map[string]HDKeyPair{},
		nextIndex: 1,
	}
}

func LoadHDKeyRing(keyPairs []HDKeyPair) *HDKeyRing {
	keyRing := NewHDKeyRing()
	for _, keyPair := range keyPairs {
		keyRing.Upsert(keyPair)
	}
	return keyRing
}

func (r *HDKeyRing) FindPair(pubKey string) (HDKeyPair, bool) {
	keyPair, ok := r.keys[pubKey]
	return keyPair, ok
}

func (r *HDKeyRing) Upsert(keyPair HDKeyPair) {
	r.keys[keyPair.PublicKey()] = keyPair
	if r.nextIndex <= keyPair.Index() {
		r.nextIndex = keyPair.Index() + 1
	}
}

// ListPublicKeys returns the list of public keys sorted by key index.
func (r *HDKeyRing) ListPublicKeys() []HDPublicKey {
	sortedKeyPairs := r.ListKeyPairs()
	pubKeys := make([]HDPublicKey, len(r.keys))
	for i, keyPair := range sortedKeyPairs {
		pubKeys[i] = keyPair.ToPublicKey()
	}
	return pubKeys
}

func (r *HDKeyRing) NextIndex() uint32 {
	return r.nextIndex
}

// ListKeyPairs returns the list of key pairs sorted by key index.
func (r *HDKeyRing) ListKeyPairs() []HDKeyPair {
	keysList := make([]HDKeyPair, len(r.keys))
	i := 0
	for _, key := range r.keys {
		keysList[i] = key
		i++
	}
	sort.SliceStable(keysList, func(i, j int) bool {
		return keysList[i].Index() < keysList[j].Index()
	})
	return keysList
}
