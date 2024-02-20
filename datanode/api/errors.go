// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"code.vegaprotocol.io/vega/datanode/entities"
	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// API Errors and descriptions.
var (
	// ErrNoTrustedProxy indactes a forwarded request that did not pass through a trusted proxy.
	ErrNoTrustedProxy = errors.New("forwarded requests need to pass through a trusted proxy")
	// ErrChannelClosed signals that the channel streaming data is closed.
	ErrChannelClosed = errors.New("channel closed")
	// ErrNotAValidVegaID signals an invalid id.
	ErrNotAValidVegaID = newInvalidArgumentError("not a valid vega id")
	// ErrMissingResourceID signals to the caller that the request expected a
	// resource id but the field is missing or empty.
	ErrMissingResourceID = newInvalidArgumentError("missing resource ID")
	// ErrEmptyMissingMarketID signals to the caller that the request expected a
	// market id but the field is missing or empty.
	ErrEmptyMissingMarketID = newInvalidArgumentError("empty or missing market ID")
	// ErrInvalidMarketID signals to the caller that the request expected a
	// market id but the field is not in the right format.
	ErrInvalidMarketID = newInvalidArgumentError("invalid market ID")
	// ErrMissingPrice signals to the caller that the request expected a price.
	ErrMissingPrice = newInvalidArgumentError("missing price")
	// ErrInvalidOrderPrice signals to the caller that the request expected a valid price.
	ErrInvalidOrderPrice = newInvalidArgumentError("invalid order price")
	// ErrInvalidOrderSize signals to the caller that the request expected a valid size.
	ErrInvalidOrderSize = newInvalidArgumentError("invalid order size")
	// ErrServerShutdown signals to the client that the server  is shutting down.
	ErrServerShutdown = errors.New("server shutdown")
	// ErrStreamClosed signals to the users that the grpc stream is closing.
	ErrStreamClosed = errors.New("stream closed")
	// ErrStreamInternal signals to the users that the grpc stream has an internal problem.
	ErrStreamInternal = errors.New("internal stream failure")
	// ErrNotMapped is when an error cannot be found in the current error map/lookup table.
	ErrNotMapped = errors.New("error not found in error lookup table")
	// ErrMissingPartyID signals that the payload is expected to contain a party id.
	ErrMissingPartyID = newInvalidArgumentError("missing party id")
	// ErrInvalidPartyID signals that the given party id is not a valid ID.
	ErrInvalidPartyID = newInvalidArgumentError("invalid party id")
	// ErrInvalidPagination signals that the pagination is invalid.
	ErrInvalidPagination = newInvalidArgumentError("invalid pagination")
	// ErrInvalidCandleID signals an invalid candle ID submitted.
	ErrInvalidCandleID = newInvalidArgumentError("invalid candle id")
	// ErrInvalidCandleTimestampsRange signals an impossible range with the candle timestamps.
	ErrInvalidCandleTimestampsRange = newInvalidArgumentError("invalid candle timestamps range")
	// ErrInvalidFilter signals that the filter is invalid.
	ErrInvalidFilter = newInvalidArgumentError("invalid filter")
	// ErrMalformedRequest signals that the request was malformed.
	ErrMalformedRequest = newInvalidArgumentError("malformed request")
	// ErrMissingOrderID signals that an order ID was required but not specified.
	ErrMissingOrderID = newInvalidArgumentError("missing orderID parameter")
	// ErrInvalidOrderID signals that an order ID provided was not a valid ID.
	ErrInvalidOrderID = newInvalidArgumentError("invalid orderID parameter")
	// ErrMissingCandleID returned if candle with this id is missing.
	ErrMissingCandleID = newInvalidArgumentError("candle id is a required parameter")
	// ErrMissingProposalID returned if proposal with this id is missing.
	ErrMissingProposalID = newInvalidArgumentError("proposal id is a required parameter")
	// ErrMissingProposalIDAndPartyID returned if proposal id and party id is missing.
	ErrMissingProposalIDAndPartyID = newInvalidArgumentError("missing proposal id and party id")
	// ErrMissingProposalIDOrPartyID returned if proposal id and party id is missing.
	ErrMissingProposalIDOrPartyID = newInvalidArgumentError("missing proposal id or party id")
	// ErrMissingProposalIDOrReference returned if proposal id or reference is missing.
	ErrMissingProposalIDOrReference = newInvalidArgumentError("missing proposal ID or reference")
	// ErrInvalidProposalID returned if proposal id is invalid.
	ErrInvalidProposalID = newInvalidArgumentError("invalid proposal id")
	// ErrMissingWithdrawalID is returned when the withdrawal ID is missing from the request.
	ErrMissingWithdrawalID = newInvalidArgumentError("missing withdrawal ID")
	// ErrInvalidWithdrawalID is returned when the withdrawal ID is not a valid vega ID.
	ErrInvalidWithdrawalID = newInvalidArgumentError("invalid withdrawal ID")
	// ErrMissingOracleSpecID is returned when the ID is missing from the request.
	ErrMissingOracleSpecID = newInvalidArgumentError("missing oracle spec ID")
	// ErrInvalidOracleSpecID is returned when the ID is not a valid ID.
	ErrInvalidOracleSpecID = newInvalidArgumentError("invalid oracle spec ID")
	// ErrMissingDepositID is returned when the deposit ID is missing from the request.
	ErrMissingDepositID = newInvalidArgumentError("missing deposit ID")
	// ErrMissingAssetID is returned when the Asset ID is missing from the request.
	ErrMissingAssetID = newInvalidArgumentError("missing asset ID")
	// ErrInvalidAssetID is returned when the Asset ID is not a valid ID.
	ErrInvalidAssetID = newInvalidArgumentError("invalid asset ID")
	// ErrMissingNodeID is returned when the node ID is missing from the request.
	ErrMissingNodeID = newInvalidArgumentError("missing node id")
	// ErrNegativeOrderVersion is returned when a request is made for an
	// order with a negative version.
	ErrNegativeOrderVersion = newInvalidArgumentError("negative order version")
	// ErrLastPaginationNotSupported is returned when last pagination is not supported.
	ErrLastPaginationNotSupported = newInvalidArgumentError("'last' pagination is not supported")
	// ErrMissingMarginFactor is returned when isolated margin mode is specified, but margin factor is not supplied.
	ErrMissingMarginFactor = newInvalidArgumentError("missing margin factor")

	// ErrOracleServiceSpecID is returned when there was no data found for the given ID.
	ErrOracleServiceGetSpec = errors.New("failed to retrieve data for oracle spec")
	// ErrERC20InvalidTokenContractAddress is returned when the ERC20 token contract address is invalid.
	ErrERC20InvalidTokenContractAddress = errors.New("invalid erc20 token contract address")
	ErrSendingGRPCHeader                = errors.New("failed to send header")
	ErrEstimateFee                      = errors.New("failed to estimate fee")
	ErrEstimateMargin                   = errors.New("failed to estimate margin")
	// OrderService...
	ErrOrderServiceGetOrders        = errors.New("failed to get orders")
	ErrOrderServiceGetVersions      = errors.New("failed to get order versions")
	ErrOrderNotFound                = errors.New("order not found")
	ErrOrderServiceGetByMarket      = errors.New("failed to get orders for market")
	ErrOrderServiceGetByMarketAndID = errors.New("failed to get orders for market and ID")
	ErrOrderServiceGetByParty       = errors.New("failed to get orders for party")
	ErrOrderServiceGetByReference   = errors.New("failed to get orders for reference")
	ErrOrderServiceGetByTxHash      = errors.New("failed to get orders for tx hash")
	ErrMissingOrderIDParameter      = errors.New("missing orderID parameter")
	// NodeService...
	ErrNodeServiceGetNodes    = errors.New("failed to get nodes")
	ErrNodeServiceGetNodeData = errors.New("failed to get node data")
	ErrNodeServiceGetByTxHash = errors.New("failed to get nodes for tx hash")
	// TradeService...
	ErrTradeServiceGetByParty          = errors.New("failed to get trades for party")
	ErrTradeServiceGetByMarket         = errors.New("failed to get trades for market")
	ErrTradeServiceList                = errors.New("failed to list trades")
	ErrTradeServiceGetPositionsByParty = errors.New("failed to get positions for party")
	ErrTradeServiceGetByOrderID        = errors.New("failed to get trades for order ID")
	ErrTradeServiceGetByTxHash         = errors.New("failed to get trades for tx hash")
	// MarketService...
	ErrMarketServiceGetByID              = errors.New("failed to get market for ID")
	ErrMarketServiceGetAllPaged          = errors.New("failed to get all markets paged")
	ErrMarketServiceGetMarketData        = errors.New("failed to get market data")
	ErrMarketServiceGetMarketDataHistory = errors.New("failed to get market data history")
	ErrMarketServiceGetMarkets           = errors.New("failed to get markets")
	ErrMarketServiceGetByTxHash          = errors.New("failed to get orders for tx hash")
	ErrMarketServiceGetDepth             = errors.New("failed to get market depth")
	// AccountService...
	ErrAccountServiceListAccounts        = errors.New("failed to get accounts")
	ErrFailedToSendSnapshot              = errors.New("failed to send accounts snapshot")
	ErrAccountServiceGetBalances         = errors.New("failed to get balances")
	ErrAccountServiceGetByTxHash         = errors.New("failed to get accounts for tx hash")
	ErrAccountServiceGetBalancesByTxHash = errors.New("failed to get balances for tx hash")
	// DelegationService...
	ErrDelegationServiceGet = errors.New("failed to get delegation")
	// SummaryService...
	ErrSummaryServiceGet = errors.New("failed to get summary")
	// WithdrawalService...
	ErrWithdrawalServiceGet = errors.New("failed to get withdrawal")
	// PositionService...
	ErrPositionServiceGetByParty   = errors.New("failed to get positions for party")
	ErrPositionServiceSendSnapshot = errors.New("failed to send positions snapshot")
	// RiskService...
	ErrRiskServiceGetMarginLevelsByID = errors.New("failed to get margin levels")
	ErrInvalidOrderSide               = newInvalidArgumentError("invalid order side")
	// RiskFactorService...
	ErrRiskFactorServiceGet = errors.New("failed to get risk factor")
	// GovernanceService...
	ErrGovernanceServiceGet          = errors.New("failed to get proposal")
	ErrGovernanceServiceGetProposals = errors.New("failed to get proposals")
	ErrGovernanceServiceGetVotes     = errors.New("failed to get votes")
	// CandleService...
	ErrCandleServiceGetCandleData       = errors.New("failed to get candle data")
	ErrCandleServiceSubscribeToCandles  = errors.New("failed to subscribe to candle data")
	ErrCandleServiceGetCandlesForMarket = errors.New("failed to get candles for market")
	// PartyService...
	ErrPartyServiceGetAll       = errors.New("failed to get parties")
	ErrPartyServiceListProfiles = errors.New("failed to get parties' profiles")
	ErrPartyServiceGetByID      = errors.New("failed to get party for ID")
	ErrPartyServiceGetByTxHash  = errors.New("failed to get parties for tx hash")
	// NotaryService...
	ErrNotaryServiceGetByResourceID = errors.New("failed to get notary for resource ID")
	// OracleSpecService...
	// ErrOracleSpecServiceGet is returned when there was no data found for the given ID.
	ErrOracleSpecServiceGet = errors.New("failed retrieve data for oracle spec")
	// ErrOracleSpecServiceGetAll is returned when there was no data found for the given ID.
	ErrOracleSpecServiceGetAll = errors.New("failed retrieve data for oracle specs")
	// OracleDataService...
	// ErrOracleDataServiceGet is returned when there was no data found for the given ID.
	ErrOracleDataServiceGet = errors.New("failed retrieve data for oracle data")
	// AssetService...
	ErrAssetServiceGetAll            = errors.New("failed to get assets")
	ErrAssetServiceGetByID           = errors.New("failed to get asset for ID")
	ErrScalingPriceFromMarketToAsset = errors.New("failed to scale price from market to asset")
	// DepositService...
	ErrDepositServiceGet = errors.New("failed to get deposit")
	// TransferService...
	ErrTransferServiceGet            = errors.New("failed to get transfer")
	ErrTransferServiceGetFeeDiscount = errors.New("failed to get current transfer fee discount")
	ErrMissingTransferID             = errors.New("missing transfer id")
	// NetworkLimits...
	ErrGetNetworkLimits = errors.New("failed to get network limits")
	// ErrGetNetworkParameters is returned when the network parameters cannot be retrieved.
	ErrGetNetworkParameters = errors.New("failed to get network parameters")
	// Network History...
	ErrGetConnectedPeerAddresses = errors.New("failed to get connected peer addresses")
	// TimeService...
	ErrTimeServiceGetTimeNow = errors.New("failed to get time now")
	// Blockchain...
	ErrBlockchainBacklogLength = errors.New("failed to get backlog length from blockchain")
	ErrBlockchainNetworkInfo   = errors.New("failed to get network info from blockchain")
	ErrBlockchainGenesisTime   = errors.New("failed to get genesis time from blockchain")
	ErrBlockchainChainID       = errors.New("failed to get chain ID from blockchain")
	// Rewards.
	ErrGetRewards         = errors.New("failed to get rewards")
	ErrRewardsGetByTxHash = errors.New("failed to get rewards for tx hash")
	// Network History.
	ErrGetActivePeerAddresses              = errors.New("failed to get active peer addresses")
	ErrGetMostRecentHistorySegment         = errors.New("failed to get most recent history segment")
	ErrListAllNetworkHistorySegment        = errors.New("failed to list all history segments")
	ErrFetchNetworkHistorySegment          = errors.New("failed to fetch segment")
	ErrNetworkHistoryNotEnabled            = errors.New("network history not enabled")
	ErrCopyHistorySegmentToFile            = errors.New("failed to copy history segment to file")
	ErrGetIpfsAddress                      = errors.New("failed to get node's ipfs address")
	ErrNetworkHistoryNoTableName           = errors.New("no table name for network history supplied")
	ErrNetworkHistoryGetContiguousSegments = newInvalidArgumentError("could not to get contiguous history segments")
	ErrNetworkHistoryOpeningSegment        = errors.New("failed to open network history segment file")
	ErrNetworkHistoryExtractingSegment     = errors.New("failed to extract data from network history segment file")
	ErrNetworkHistoryCreatingZipFile       = errors.New("failed to create zip file writer for network history segment")
	ErrNetworkHistoryServiceNotInitialised = errors.New("network history service not initialised")

	// ErrGetEpoch is returned when the epoch cannot be retrieved.
	ErrGetEpoch     = errors.New("failed to get epoch")
	ErrEpochIDParse = newInvalidArgumentError("failed to parse epoch id")
	// LedgerService...
	ErrLedgerServiceGet    = errors.New("failed to query ledger entries")
	ErrLedgerServiceExport = errors.New("failed to export ledger entries")
	// MultiSigService...
	ErrMultiSigServiceGetAdded   = errors.New("failed to get added multisig events")
	ErrMultiSigServiceGetRemoved = errors.New("failed to get removed multisig events")
	// LiquidityProvisionService...
	ErrLiquidityProvisionServiceGet          = errors.New("failed to get liquidity provision")
	ErrLiquidityProvisionServiceGetProviders = errors.New("failed to get liquidity providers")
	// CheckpointService...
	ErrCheckpointServiceGet = errors.New("failed to get checkpoint")
	// StakeLinkingService...
	ErrStakeLinkingServiceGet = errors.New("failed to get stake linking")
	// CoreSnapshotService...
	ErrCoreSnapshotServiceListSnapshots = errors.New("failed to list core snapshots")
	// ProtocolUpgradeService...
	ErrProtocolUpgradeServiceListProposals = errors.New("failed to list protocol upgrade proposals")
	// KeyRotationService...
	ErrKeyRotationServiceGetPerNode = errors.New("failed to get key rotations for node")
	ErrKeyRotationServiceGetAll     = errors.New("failed to get all key rotations")
	// EthereumKeyRotationService...
	ErrEthereumKeyRotationServiceGetPerNode = errors.New("failed to get ethereum key rotations for node")
	ErrEthereumKeyRotationServiceGetAll     = errors.New("failed to get all ethereum key rotations")
	// BlockService...
	ErrBlockServiceGetLast = errors.New("failed to get last block")
	// Positions...
	ErrPositionsGetByTxHash           = errors.New("failed to get positions for tx hash")
	ErrPositionsInvalidAccountBalance = newInvalidArgumentError("invalid account balance")
	// Ledger entries...
	ErrLedgerEntriesGetByTxHash = errors.New("failed to get ledger entries for tx hash")
	// Transfers...
	ErrTransfersGetByTxHash = errors.New("failed to get transfers for tx hash")
	// Votes...
	ErrVotesGetByTxHash = errors.New("failed to get votes for tx hash")
	// ERC20MultiSigSignerEvents...
	ErrERC20MultiSigSignerAddedEventGetByTxHash   = errors.New("failed to get ERC20 multisig signer add events for tx hash")
	ErrERC20MultiSigSignerRemovedEventGetByTxHash = errors.New("failed to get ERC20 multisig signer removed events for tx hash")
	// Oracles...
	ErrOracleSpecGetByTxHash = errors.New("failed to get oracle spec for tx hash")
	ErrOracleDataGetByTxHash = errors.New("failed to get oracle data for tx hash")
	// Deposits...
	ErrDepositsGetByTxHash = errors.New("failed to get deposits for tx hash")
	// Withdrawals...
	ErrWithdrawalsGetByTxHash = errors.New("failed to get withdrawals for tx hash")
	// Assets...
	ErrAssetsGetByTxHash = errors.New("failed to get assets for tx hash")
	// Liquidity provision...
	ErrLiquidityProvisionGetByTxHash = errors.New("failed to get liquidity provision for tx hash")
	// Proposals...
	ErrProposalsGetByTxHash = errors.New("failed to get proposals for tx hash")
	// Delegations...
	ErrDelegationsGetByTxHash = errors.New("failed to get delegations for tx hash")
	// Signatures...
	ErrSignaturesGetByTxHash = errors.New("failed to get signatures for tx hash")
	// NetworkParamaters...
	ErrNetworkParametersGetByTxHash = errors.New("failed to get network parameters for tx hash")
	ErrNetworkParameterNotFound     = errors.New("network parameter not found")

	// KeyRotations...
	ErrKeyRotationsGetByTxHash = errors.New("failed to get key rotations for tx hash")
	// EthereumKeyRotations...
	ErrEthereumKeyRotationsGetByTxHash = errors.New("failed to get ethereum key rotations for tx hash")
	// ProtocolUpgradeProposals...
	ErrProtocolUpgradeProposalsGetByTxHash = errors.New("failed to get protocol upgrade proposals for tx hash")
	// MarginLevels...
	ErrMarginLevelsGetByTxHash = errors.New("failed to get margin levels for tx hash")

	// TxHashes...
	ErrMissingEmptyTxHash = newInvalidArgumentError("missing or empty transaction hash")
	ErrInvalidTxHash      = newInvalidArgumentError("not a valid transaction hash")

	// Funding Periods.
	ErrListFundingPeriod           = errors.New("failed to get funding periods")
	ErrListFundingPeriodDataPoints = errors.New("failed to get funding period data points")

	// Referral Programs.
	ErrGetCurrentReferralProgram = errors.New("failed to get current referral program")
	ErrGetReferralSetStats       = errors.New("failed to get referral set stats")

	// Fees stats.
	ErrGetFeesStats         = errors.New("failed to get current fees stats")
	ErrFeesStatsRequest     = errors.New("marketID or assetID must be provided")
	ErrGetFeesStatsForParty = errors.New("failed to get current fees stats for party")

	// Teams.
	ErrListTeams              = errors.New("failed to list teams")
	ErrListTeamReferees       = errors.New("failed to list team referees")
	ErrListTeamStatistics     = errors.New("failed to list team statistics")
	ErrListTeamRefereeHistory = errors.New("failed to list team referee history")

	// Volume discount Programs.
	ErrGetCurrentVolumeDiscountProgram = errors.New("failed to get current volume discount program")
	ErrGetVolumeDiscountStats          = errors.New("failed to get volume discount stats")

	// Paid liquidity fees.
	ErrListPaidLiquidityFees = errors.New("failed to list paid liquidity fees")
	// List Games.
	ErrListGames = errors.New("failed to list games")

	// Transfer fee estimates.
	ErrInvalidTransferAmount = newInvalidArgumentError("invalid transfer amount")

	ErrListPartyMarginModes = errors.New("failed to list parties margin modes")
	ErrListAMMPools         = errors.New("failed to list AMM pools")
)

// errorMap contains a mapping between errors and Vega numeric error codes.
var errorMap = map[string]int32{
	// General
	ErrNotMapped.Error(): 10000,
	//   ErrChainNotConnected.Error():          10001,
	ErrChannelClosed.Error():        10002,
	ErrEmptyMissingMarketID.Error(): 10003,
	//   ErrEmptyMissingOrderID.Error():        10004,
	//   ErrEmptyMissingOrderReference.Error(): 10005,
	//   ErrEmptyMissingPartyID.Error():        10006,
	//   ErrEmptyMissingSinceTimestamp.Error(): 10007,
	ErrStreamClosed.Error():   10008,
	ErrServerShutdown.Error(): 10009,
	ErrStreamInternal.Error(): 10010,
	//   ErrInvalidMarketID.Error():            10011,
	ErrMissingOrderID.Error():   10012,
	ErrMissingPartyID.Error():   10014,
	ErrMalformedRequest.Error(): 10015,
	//   ErrMissingAsset.Error():             10017,
	ErrMissingAssetID.Error(): 10017,
	//   ErrSubmitOrder.Error():              10018,
	//   ErrAmendOrder.Error():               10019,
	//   ErrCancelOrder.Error():              10020,
	ErrMissingProposalID.Error():            10021,
	ErrMissingProposalIDOrReference.Error(): 10022,
	ErrMissingProposalIDAndPartyID.Error():  10023,
	ErrMissingProposalIDOrPartyID.Error():   10024,
	ErrMissingResourceID.Error():            10025,
	ErrMissingPrice.Error():                 10026,
	ErrInvalidOrderPrice.Error():            10027,
	ErrInvalidPagination.Error():            10028,
	ErrInvalidFilter.Error():                10029,
	ErrMissingCandleID.Error():              10030,
	ErrMissingOracleSpecID.Error():          10031,
	ErrMissingNodeID.Error():                10034,
	ErrInvalidOrderSide.Error():             10035,
	ErrEpochIDParse.Error():                 10036,
	ErrSendingGRPCHeader.Error():            10037,
	ErrInvalidAssetID.Error():               10038,
	ErrEstimateFee.Error():                  10039,
	ErrEstimateMargin.Error():               10040,
	// Orders
	//   ErrOrderServiceGetByMarket.Error():      20001,
	//   ErrOrderServiceGetByMarketAndID.Error(): 20002,
	//   ErrOrderServiceGetByParty.Error():       20003,
	//   ErrOrderServiceGetByReference.Error():   20004,
	ErrOrderServiceGetVersions.Error(): 20005,
	ErrOrderNotFound.Error():           20006,
	ErrOrderServiceGetOrders.Error():   20007,
	// Markets
	//   ErrMarketServiceGetMarkets.Error():    30001,
	ErrMarketServiceGetByID.Error(): 30002,
	//   ErrMarketServiceGetDepth.Error():      30003,
	ErrMarketServiceGetMarketData.Error():        30004,
	ErrMarketServiceGetAllPaged.Error():          30005,
	ErrMarketServiceGetMarketDataHistory.Error(): 30006,
	// Trades
	ErrTradeServiceGetByMarket.Error(): 40001,
	//   ErrTradeServiceGetByParty.Error():          40002,
	//   ErrTradeServiceGetPositionsByParty.Error(): 40003,
	//   ErrTradeServiceGetByOrderID.Error():        40004,
	ErrTradeServiceList.Error(): 40005,
	// Parties
	ErrPartyServiceGetAll.Error():  50001,
	ErrPartyServiceGetByID.Error(): 50002,
	// Candles
	ErrCandleServiceGetCandleData.Error():       60001,
	ErrCandleServiceSubscribeToCandles.Error():  60002,
	ErrCandleServiceGetCandlesForMarket.Error(): 60003,
	// Risk
	ErrRiskServiceGetMarginLevelsByID.Error(): 70001,
	// Accounts
	//   ErrAccountServiceGetMarketAccounts.Error(): 80001,
	//   ErrAccountServiceGetPartyAccounts.Error():  80002,
	ErrMissingWithdrawalID.Error(): 80003,
	ErrMissingDepositID.Error():    80004,
	// ErrMissingAssetID.Error():             80005,
	ErrAccountServiceListAccounts.Error(): 80006,
	// ErrAccountServiceSQLStoreNotAvailable.Error(): 80007,
	ErrFailedToSendSnapshot.Error(): 80008,
	// Blockchain client
	//   ErrBlockchainBacklogLength.Error(): 90001,
	//   ErrBlockchainNetworkInfo.Error():   90002,
	//   ErrBlockchainGenesisTime.Error():   90003,
	ErrAccountServiceGetBalances.Error(): 90004,
	// Network limits
	ErrGetNetworkLimits.Error():         100001,
	ErrGetNetworkParameters.Error():     100002,
	ErrNetworkParameterNotFound.Error(): 100003,
	// Node
	ErrNodeServiceGetNodes.Error():    110001,
	ErrNodeServiceGetNodeData.Error(): 110002,
	// ERC20
	ErrERC20InvalidTokenContractAddress.Error(): 120001,
	// Delegation
	ErrDelegationServiceGet.Error(): 130001,
	// Summary
	ErrSummaryServiceGet.Error(): 140001,
	// Withdrawal
	ErrWithdrawalServiceGet.Error(): 150001,
	// Position
	ErrPositionServiceGetByParty.Error():   160001,
	ErrPositionServiceSendSnapshot.Error(): 160002,
	// Governance
	ErrGovernanceServiceGet.Error():          170001,
	ErrGovernanceServiceGetProposals.Error(): 170002,
	ErrGovernanceServiceGetVotes.Error():     170003,
	// Notary
	ErrNotaryServiceGetByResourceID.Error(): 180001,
	// Oracle
	ErrOracleSpecServiceGet.Error():    190001,
	ErrOracleSpecServiceGetAll.Error(): 190002,
	ErrOracleDataServiceGet.Error():    190003,
	// Asset
	ErrAssetServiceGetAll.Error():            200001,
	ErrAssetServiceGetByID.Error():           200002,
	ErrScalingPriceFromMarketToAsset.Error(): 200003,
	// Deposit
	ErrDepositServiceGet.Error(): 210001,
	// Transfer
	ErrTransferServiceGet.Error(): 220001,
	// Reward
	ErrGetRewards.Error(): 230001,
	// Network History
	ErrGetConnectedPeerAddresses.Error():    240001,
	ErrGetMostRecentHistorySegment.Error():  240002,
	ErrListAllNetworkHistorySegment.Error(): 240003,
	ErrGetIpfsAddress.Error():               240004,
	// Epoch
	ErrGetEpoch.Error(): 250001,
	// Ledger
	ErrLedgerServiceGet.Error():    260001,
	ErrLedgerServiceExport.Error(): 260002,
	// MultiSig
	ErrMultiSigServiceGetAdded.Error():   270001,
	ErrMultiSigServiceGetRemoved.Error(): 270002,
	// Liquidity Provision
	ErrLiquidityProvisionServiceGet.Error(): 280001,
	// Checkpoint
	ErrCheckpointServiceGet.Error(): 290001,
	// Stake Linking
	ErrStakeLinkingServiceGet.Error(): 300001,
	// Risk Factor
	ErrRiskFactorServiceGet.Error(): 310001,
	// Core Snapshot
	ErrCoreSnapshotServiceListSnapshots.Error(): 320001,
	// Protocol Upgrade
	ErrProtocolUpgradeServiceListProposals.Error(): 330001,
	// Key Rotation
	ErrKeyRotationServiceGetPerNode.Error(): 340001,
	ErrKeyRotationServiceGetAll.Error():     340002,
	// Ethereum Key Rotation
	ErrEthereumKeyRotationServiceGetPerNode.Error(): 350001,
	ErrEthereumKeyRotationServiceGetAll.Error():     350002,
	// Block
	ErrBlockServiceGetLast.Error(): 360001,
	// End of mapping
}

// ErrorMap returns a map of error to code, which is a mapping between
// API errors and Vega API specific numeric codes.
func ErrorMap() map[string]int32 {
	return errorMap
}

// apiError is a helper function to build the Vega specific Error Details that
// can be returned by gRPC API and therefore also REST, GraphQL will be mapped too.
// It takes a standardised grpcCode, a Vega specific apiError, and optionally one
// or more internal errors (error from the core, rather than API).
func apiError(grpcCode codes.Code, apiError error, innerErrors ...error) error {
	s := status.Newf(grpcCode, "%v error", grpcCode)
	// Create the API specific error detail for error e.g. missing party ID
	detail := types.ErrorDetail{
		Message: apiError.Error(),
	}
	// Lookup the API specific error in the table, return not found/not mapped
	// if a code has not yet been added to the map, can happen if developer misses
	// a step, periodic checking/ownership of API package can keep this up to date.
	vegaCode, found := errorMap[apiError.Error()]
	if found {
		detail.Code = vegaCode
	} else {
		detail.Code = errorMap[ErrNotMapped.Error()]
	}
	// If there is an inner error (and possibly in the future, a config to turn this
	// level of detail on/off) then process and append to inner.
	first := true
	for _, err := range innerErrors {
		if !first {
			detail.Inner += ", "
		}
		detail.Inner += err.Error()
		first = false
	}
	// Pack the Vega domain specific errorDetails into the status returned by gRPC domain.
	s, _ = s.WithDetails(&detail)
	return s.Err()
}

type invalidArgumentError struct {
	err error
}

func newInvalidArgumentError(msg string) *invalidArgumentError {
	return &invalidArgumentError{err: errors.New(msg)}
}

func (e *invalidArgumentError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func hasInvalidArgumentError(errs ...error) bool {
	for _, err := range errs {
		if _, ok := err.(*invalidArgumentError); ok || errors.Is(err, entities.ErrInvalidID) {
			return true
		}
	}
	return false
}

func hasNotFoundError(errs ...error) bool {
	for _, err := range errs {
		if errors.Is(err, entities.ErrNotFound) {
			return true
		}
	}
	return false
}

func hasError(errs ...error) bool {
	for _, err := range errs {
		if err != nil {
			return true
		}
	}
	return false
}

func formatE(err error, errs ...error) error {
	allErrs := append(errs, err)

	if !hasError(allErrs...) {
		return nil
	}

	switch {
	// only for "GetOne"-like store methods
	case hasNotFoundError(allErrs...):
		return apiError(codes.NotFound, err, errs...)
	case hasInvalidArgumentError(allErrs...):
		return apiError(codes.InvalidArgument, err, errs...)
	default:
		// could handle more errors like context cancelled,
		// deadline exceeded, but let's see later
		return apiError(codes.Internal, err, errs...)
	}
}

// FormatE exports the formatE function (primarily for testing).
var FormatE = formatE
