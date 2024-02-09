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

package commands

import (
	"fmt"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckUpdatePartyProfile(cmd *commandspb.UpdatePartyProfile) error {
	return checkUpdatePartyProfile(cmd).ErrorOrNil()
}

func checkUpdatePartyProfile(cmd *commandspb.UpdatePartyProfile) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("update_party_profile", ErrIsRequired)
	}

	if len(cmd.Alias) > 32 {
		errs.AddForProperty("update_party_profile.alias", ErrIsLimitedTo32Characters)
	}

	if len(cmd.Metadata) > 10 {
		errs.AddForProperty("update_party_profile.metadata", ErrIsLimitedTo10Entries)
	} else {
		seenKeys := map[string]interface{}{}
		for i, m := range cmd.Metadata {
			if len(m.Key) > 32 {
				errs.AddForProperty(fmt.Sprintf("update_party_profile.metadata.%d.key", i), ErrIsLimitedTo32Characters)
			} else if len(m.Key) == 0 {
				errs.AddForProperty(fmt.Sprintf("update_party_profile.metadata.%d.key", i), ErrCannotBeBlank)
			}

			_, alreadySeen := seenKeys[m.Key]
			if alreadySeen {
				errs.AddForProperty(fmt.Sprintf("update_party_profile.metadata.%d.key", i), ErrIsDuplicated)
			}
			seenKeys[m.Key] = nil

			if len(m.Value) > 255 {
				errs.AddForProperty(fmt.Sprintf("update_party_profile.metadata.%d.value", i), ErrIsLimitedTo255Characters)
			}
		}
	}

	return errs
}
