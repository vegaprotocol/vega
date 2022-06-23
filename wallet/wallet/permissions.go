package wallet

import (
	"fmt"
)

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
	return p.PublicKeys.Access == ReadAccess || p.PublicKeys.Access == WriteAccess
}

func (p Permissions) CanUseKey(pubKey string) bool {
	if !p.CanListKeys() {
		return false
	}

	// No restricted keys specified. All keys can be listed.
	if len(p.PublicKeys.RestrictedKeys) == 0 {
		return true
	}

	for _, k := range p.PublicKeys.RestrictedKeys {
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
	NoAccess    AccessMode = "none"
	ReadAccess  AccessMode = "read"
	WriteAccess AccessMode = "write"
)

func ToAccessMode(mode string) (AccessMode, error) {
	switch mode {
	case "read":
		return ReadAccess, nil
	case "write":
		return WriteAccess, nil
	case "none":
		return NoAccess, nil
	default:
		return NoAccess, fmt.Errorf("access mode %q is not supported", mode)
	}
}

func AccessModeToString(m AccessMode) string {
	switch m {
	case ReadAccess, WriteAccess, NoAccess:
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
//
// Methods requiring write access:
//   Nothing requires this type of access for now.
type PublicKeysPermission struct {
	Access         AccessMode `json:"access"`
	RestrictedKeys []string   `json:"restrictedKeys"`
}

func (p PublicKeysPermission) Enabled() bool {
	return p.Access != NoAccess
}

func (p PublicKeysPermission) HasRestrictedKeys() bool {
	return len(p.RestrictedKeys) != 0
}

// NoPublicKeysPermission returns a revoked access for public keys.
func NoPublicKeysPermission() PublicKeysPermission {
	return PublicKeysPermission{
		Access: NoAccess,
	}
}
