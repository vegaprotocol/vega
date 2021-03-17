package types

import "code.vegaprotocol.io/vega/proto"

type Proposal = proto.Proposal
type Vote = proto.Vote
type ProposalTerms = proto.ProposalTerms

type ProposalError = proto.ProposalError

const (
	// Default value
	ProposalError_PROPOSAL_ERROR_UNSPECIFIED ProposalError = 0
	// The specified close time is too early base on network parameters
	ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON ProposalError = 1
	// The specified close time is too late based on network parameters
	ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE ProposalError = 2
	// The specified enact time is too early based on network parameters
	ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_SOON ProposalError = 3
	// The specified enact time is too late based on network parameters
	ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_LATE ProposalError = 4
	// The proposer for this proposal as insufficient tokens
	ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS ProposalError = 5
	// The instrument quote name and base name were the same
	ProposalError_PROPOSAL_ERROR_INVALID_INSTRUMENT_SECURITY ProposalError = 6
	// The proposal has no product
	ProposalError_PROPOSAL_ERROR_NO_PRODUCT ProposalError = 7
	// The specified product is not supported
	ProposalError_PROPOSAL_ERROR_UNSUPPORTED_PRODUCT ProposalError = 8
	// Invalid future maturity timestamp (expect RFC3339)
	ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT_TIMESTAMP ProposalError = 9
	// The product maturity is past
	ProposalError_PROPOSAL_ERROR_PRODUCT_MATURITY_IS_PASSED ProposalError = 10
	// The proposal has no trading mode
	ProposalError_PROPOSAL_ERROR_NO_TRADING_MODE ProposalError = 11
	// The proposal has an unsupported trading mode
	ProposalError_PROPOSAL_ERROR_UNSUPPORTED_TRADING_MODE ProposalError = 12
	// The proposal failed node validation
	ProposalError_PROPOSAL_ERROR_NODE_VALIDATION_FAILED ProposalError = 13
	// A field is missing in a builtin asset source
	ProposalError_PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD ProposalError = 14
	// The contract address is missing in the ERC20 asset source
	ProposalError_PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS ProposalError = 15
	// The asset identifier is invalid or does not exist on the Vega network
	ProposalError_PROPOSAL_ERROR_INVALID_ASSET ProposalError = 16
	// Proposal terms timestamps are not compatible (Validation < Closing < Enactment)
	ProposalError_PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS ProposalError = 17
	// No risk parameters were specified
	ProposalError_PROPOSAL_ERROR_NO_RISK_PARAMETERS ProposalError = 18
	// Invalid key in update network parameter proposal
	ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY ProposalError = 19
	// Invalid valid in update network parameter proposal
	ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE ProposalError = 20
	// Validation failed for network parameter proposal
	ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED ProposalError = 21
	// Opening auction duration is less than the network minimum opening auction time
	ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_SMALL ProposalError = 22
	// Opening auction duration is more than the network minimum opening auction time
	ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_LARGE ProposalError = 23
	// Market proposal is missing a liquidity commitment
	ProposalError_PROPOSAL_ERROR_MARKET_MISSING_LIQUIDITY_COMMITMENT ProposalError = 24
	// Market proposal market could not be instantiate in execution
	ProposalError_PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET ProposalError = 25
	// Market proposal market contained invalid product definition
	ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT ProposalError = 26
)

type Proposal_State = proto.Proposal_State

const (
	// Default value, always invalid
	Proposal_STATE_UNSPECIFIED Proposal_State = 0
	// Proposal enactment has failed - even though proposal has passed, its execution could not be performed
	Proposal_STATE_FAILED Proposal_State = 1
	// Proposal is open for voting
	Proposal_STATE_OPEN Proposal_State = 2
	// Proposal has gained enough support to be executed
	Proposal_STATE_PASSED Proposal_State = 3
	// Proposal wasn't accepted (proposal terms failed validation due to wrong configuration or failing to meet network requirements)
	Proposal_STATE_REJECTED Proposal_State = 4
	// Proposal didn't get enough votes (either failing to gain required participation or majority level)
	Proposal_STATE_DECLINED Proposal_State = 5
	// Proposal enacted
	Proposal_STATE_ENACTED Proposal_State = 6
	// Waiting for node validation of the proposal
	Proposal_STATE_WAITING_FOR_NODE_VOTE Proposal_State = 7
)
