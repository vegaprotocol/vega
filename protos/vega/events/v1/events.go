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
