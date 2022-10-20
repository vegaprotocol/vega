package service

type ProtocolUpgrade struct {
	upgradeStarted bool
}

func NewProtocolUpgrade() *ProtocolUpgrade {
	return &ProtocolUpgrade{}
}

func (p *ProtocolUpgrade) GetProtocolUpgradeStarted() bool {
	return p.upgradeStarted
}

func (p *ProtocolUpgrade) SetProtocolUpgradeStarted() {
	p.upgradeStarted = true
}
