package wallet

const (
	// Version1 identifies HD wallet with a key derivation version 1.
	Version1 = uint32(1)
	// Version2 identifies HD wallet with a key derivation version 2.
	Version2 = uint32(2)
	// LatestVersion is the latest version of Vega's HD wallet. Created wallets
	// are always pointing to the latest version.
	LatestVersion = Version2
)

// SupportedKeyDerivationVersions list of key derivation versions supported by
// Vega's HD wallet.
var SupportedKeyDerivationVersions = []uint32{Version1, Version2}

func IsKeyDerivationVersionSupported(v uint32) bool {
	return v == Version1 || v == Version2
}
