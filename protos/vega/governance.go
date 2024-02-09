package vega

type (
	ProposalOneOffTermChangeType      = isProposalTerms_Change
	ProposalOneOffTermBatchChangeType = isBatchProposalTermsChange_Change
)

func (gd *GovernanceData) IsProposalNode() {}
