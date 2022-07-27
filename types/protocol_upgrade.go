package types

type UpgradeStatus struct {
	AcceptedReleaseInfo *ReleaseInfo
	ReadyToUpgrade      bool
}

type ReleaseInfo struct {
	VegaReleaseTag     string
	DatanodeReleaseTag string
	UpgradeBlockHeight uint64
}
