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

package config

// Empty is used when a command or sub-command receives no argument and has no execution.
type Empty struct{}

type VegaHomeFlag struct {
	VegaHome string `long:"home" description:"Path to the custom home for vega"`
}
