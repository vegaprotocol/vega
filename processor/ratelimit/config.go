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

package ratelimit

type Config struct {
	// How many requests
	Requests int `long:"requests" description:" "`

	// In the last `PerNBlocks` blocks
	PerNBlocks int `long:"per-n-blocks" description:" "`
}

// NewDefaultConfig allows 500 requests in the last 10 blocks.
func NewDefaultConfig() Config {
	return Config{
		Requests:   500,
		PerNBlocks: 10,
	}
}
