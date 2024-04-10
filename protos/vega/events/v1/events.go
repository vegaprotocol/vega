package v1

const Version = 1

func (AuctionEvent) IsEvent()      {}
func (TransactionResult) IsEvent() {}

func (OneOffTransfer) IsTransferKind()              {}
func (RecurringTransfer) IsTransferKind()           {}
func (OneOffGovernanceTransfer) IsTransferKind()    {}
func (RecurringGovernanceTransfer) IsTransferKind() {}

func (OneOffGovernanceTransfer) IsGovernanceTransferKind()    {}
func (RecurringGovernanceTransfer) IsGovernanceTransferKind() {}

func (FundingPeriodDataPoint_Source) GetEnums() map[int32]string {
	return FundingPeriodDataPoint_Source_name
}

func (ProtocolUpgradeProposalStatus) GetEnums() map[int32]string {
	return ProtocolUpgradeProposalStatus_name
}

func (StakeLinking_Status) GetEnums() map[int32]string {
	return StakeLinking_Status_name
}

func (StakeLinking_Type) GetEnums() map[int32]string {
	return StakeLinking_Type_name
}

func (Transfer_Status) GetEnums() map[int32]string {
	return Transfer_Status_name
}
