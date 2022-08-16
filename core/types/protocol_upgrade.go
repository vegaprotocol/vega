package types

type UpgradeStatus struct {
	AcceptedReleaseInfo *ReleaseInfo
	ReadyToUpgrade      bool
}

type ReleaseInfo struct {
	VegaReleaseTag     string
	UpgradeBlockHeight uint64
}
