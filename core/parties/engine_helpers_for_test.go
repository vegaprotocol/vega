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

package parties

import (
	"strings"

	"code.vegaprotocol.io/vega/core/types"

	"golang.org/x/exp/slices"
)

func (e *Engine) ListProfiles() []types.PartyProfile {
	profiles := make([]types.PartyProfile, 0, len(e.profiles))

	for _, profile := range e.profiles {
		profiles = append(profiles, *profile)
	}

	SortByPartyID(profiles)

	return profiles
}

func SortByPartyID(toSort []types.PartyProfile) {
	slices.SortStableFunc(toSort, func(a, b types.PartyProfile) int {
		return strings.Compare(string(a.PartyID), string(b.PartyID))
	})
}
