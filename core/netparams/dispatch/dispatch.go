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

package dispatch

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/logging"
)

type Assets interface {
	IsEnabled(string) bool
}

func RewardAssetUpdate(
	log *logging.Logger,
	assets Assets,
) func(context.Context, string) error {
	return func(ctx context.Context, value string) error {
		if !assets.IsEnabled(value) {
			log.Debug("tried to push a reward update with an non-enabled asset",
				logging.String("asset-id", value))
			return fmt.Errorf("invalid asset %v", value)
		}

		return nil
	}
}
