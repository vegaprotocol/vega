package v1

const Version = 1

func (AuctionEvent) IsEvent() {}

func (OneOffTransferInstruction) IsTransferInstructionKind()    {}
func (RecurringTransferInstruction) IsTransferInstructionKind() {}
