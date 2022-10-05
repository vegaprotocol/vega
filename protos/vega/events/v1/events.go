package v1

const Version = 1

func (AuctionEvent) IsEvent()      {}
func (TransactionResult) IsEvent() {}

func (OneOffTransfer) IsTransferKind()    {}
func (RecurringTransfer) IsTransferKind() {}
