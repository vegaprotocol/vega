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

var PublicKeysPermissionLabel = "public_keys"

// Permissions describes the permissions set on a given hostname.
type Permissions struct {
	PublicKeys PublicKeysPermission `json:"publicKeys"`
}

func (p Permissions) Summary() PermissionsSummary {
	summary := map[string]string{}
	summary[PublicKeysPermissionLabel] = AccessModeToString(p.PublicKeys.Access)
	return summary
}

func (p Permissions) CanListKeys() bool {
	return p.PublicKeys.Access == ReadAccess
}

func (p Permissions) CanUseKey(pubKey string) bool {
	if !p.CanListKeys() {
		return false
	}

	// No allowed keys specified. All keys can be listed.
	if len(p.PublicKeys.AllowedKeys) == 0 {
		return true
	}

	for _, k := range p.PublicKeys.AllowedKeys {
		if k == pubKey {
			return true
		}
	}
	return false
}

func DefaultPermissions() Permissions {
	return Permissions{
		PublicKeys: NoPublicKeysPermission(),
	}
}

type PermissionsSummary map[string]string

type AccessMode string

var (
	NoAccess   AccessMode = "none"
	ReadAccess AccessMode = "read"
)

func AccessModeToString(m AccessMode) string {
	switch m {
	case ReadAccess, NoAccess:
		return string(m)
	default:
		return string(NoAccess)
	}
}

// PublicKeysPermission defines what the third-party application can do with
// the public keys of the wallet.
//
// Methods requiring read access:
//   - list_keys
type PublicKeysPermission struct {
	Access AccessMode `json:"access"`
	// AllowedKeys lists all the keys a third-party application has access to.
	// All keys are valid and usable (no tainted key).
	AllowedKeys []string `json:"allowedKeys"`
}

func (p PublicKeysPermission) Enabled() bool {
	return p.Access != NoAccess
}

func (p PublicKeysPermission) HasAllowedKeys() bool {
	return len(p.AllowedKeys) != 0
}

// NoPublicKeysPermission returns a revoked access for public keys.
func NoPublicKeysPermission() PublicKeysPermission {
	return PublicKeysPermission{
		Access: NoAccess,
	}
}
