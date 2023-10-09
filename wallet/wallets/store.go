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

package wallets

import (
	"fmt"

	"code.vegaprotocol.io/vega/paths"
	wstorev1 "code.vegaprotocol.io/vega/wallet/wallet/store/v1"
)

// InitialiseStore builds a wallet Store specifically for users wallets.
func InitialiseStore(vegaHome string, withFileWatcher bool) (*wstorev1.FileStore, error) {
	p := paths.New(vegaHome)
	return InitialiseStoreFromPaths(p, withFileWatcher)
}

// InitialiseStoreFromPaths builds a wallet Store specifically for users wallets.
func InitialiseStoreFromPaths(vegaPaths paths.Paths, withFileWatcher bool) (*wstorev1.FileStore, error) {
	walletsHome, err := vegaPaths.CreateDataPathFor(paths.WalletsDataHome)
	if err != nil {
		return nil, fmt.Errorf("couldn't get wallets data home path: %w", err)
	}
	return wstorev1.InitialiseStore(walletsHome, withFileWatcher)
}
