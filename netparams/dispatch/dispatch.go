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

package dispatch

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/logging"
)

type Assets interface {
	IsEnabled(string) bool
}

func GovernanceAssetUpdate(
	log *logging.Logger,
	assets Assets,
) func(context.Context, string) error {
	return func(ctx context.Context, value string) error {
		if !assets.IsEnabled(value) {
			log.Debug("tried to push a governance update with an non-enabled asset",
				logging.String("asset-id", value))
			return fmt.Errorf("invalid asset %v", value)
		}

		return nil
	}
}
