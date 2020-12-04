# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [proto/api/trading.proto](#proto/api/trading.proto)
    - [AccountsSubscribeRequest](#api.AccountsSubscribeRequest)
    - [AmendOrderRequest](#api.AmendOrderRequest)
    - [AssetByIDRequest](#api.AssetByIDRequest)
    - [AssetByIDResponse](#api.AssetByIDResponse)
    - [AssetsRequest](#api.AssetsRequest)
    - [AssetsResponse](#api.AssetsResponse)
    - [CancelOrderRequest](#api.CancelOrderRequest)
    - [CandlesRequest](#api.CandlesRequest)
    - [CandlesResponse](#api.CandlesResponse)
    - [CandlesSubscribeRequest](#api.CandlesSubscribeRequest)
    - [DepositRequest](#api.DepositRequest)
    - [DepositResponse](#api.DepositResponse)
    - [DepositsRequest](#api.DepositsRequest)
    - [DepositsResponse](#api.DepositsResponse)
    - [ERC20WithdrawalApprovalRequest](#api.ERC20WithdrawalApprovalRequest)
    - [ERC20WithdrawalApprovalResponse](#api.ERC20WithdrawalApprovalResponse)
    - [EstimateFeeRequest](#api.EstimateFeeRequest)
    - [EstimateFeeResponse](#api.EstimateFeeResponse)
    - [EstimateMarginRequest](#api.EstimateMarginRequest)
    - [EstimateMarginResponse](#api.EstimateMarginResponse)
    - [FeeInfrastructureAccountsRequest](#api.FeeInfrastructureAccountsRequest)
    - [FeeInfrastructureAccountsResponse](#api.FeeInfrastructureAccountsResponse)
    - [GetNetworkParametersProposalsRequest](#api.GetNetworkParametersProposalsRequest)
    - [GetNetworkParametersProposalsResponse](#api.GetNetworkParametersProposalsResponse)
    - [GetNewAssetProposalsRequest](#api.GetNewAssetProposalsRequest)
    - [GetNewAssetProposalsResponse](#api.GetNewAssetProposalsResponse)
    - [GetNewMarketProposalsRequest](#api.GetNewMarketProposalsRequest)
    - [GetNewMarketProposalsResponse](#api.GetNewMarketProposalsResponse)
    - [GetNodeSignaturesAggregateRequest](#api.GetNodeSignaturesAggregateRequest)
    - [GetNodeSignaturesAggregateResponse](#api.GetNodeSignaturesAggregateResponse)
    - [GetProposalByIDRequest](#api.GetProposalByIDRequest)
    - [GetProposalByIDResponse](#api.GetProposalByIDResponse)
    - [GetProposalByReferenceRequest](#api.GetProposalByReferenceRequest)
    - [GetProposalByReferenceResponse](#api.GetProposalByReferenceResponse)
    - [GetProposalsByPartyRequest](#api.GetProposalsByPartyRequest)
    - [GetProposalsByPartyResponse](#api.GetProposalsByPartyResponse)
    - [GetProposalsRequest](#api.GetProposalsRequest)
    - [GetProposalsResponse](#api.GetProposalsResponse)
    - [GetUpdateMarketProposalsRequest](#api.GetUpdateMarketProposalsRequest)
    - [GetUpdateMarketProposalsResponse](#api.GetUpdateMarketProposalsResponse)
    - [GetVotesByPartyRequest](#api.GetVotesByPartyRequest)
    - [GetVotesByPartyResponse](#api.GetVotesByPartyResponse)
    - [LastTradeRequest](#api.LastTradeRequest)
    - [LastTradeResponse](#api.LastTradeResponse)
    - [LiquidityProvisionsRequest](#api.LiquidityProvisionsRequest)
    - [LiquidityProvisionsResponse](#api.LiquidityProvisionsResponse)
    - [MarginLevelsRequest](#api.MarginLevelsRequest)
    - [MarginLevelsResponse](#api.MarginLevelsResponse)
    - [MarginLevelsSubscribeRequest](#api.MarginLevelsSubscribeRequest)
    - [MarketAccountsRequest](#api.MarketAccountsRequest)
    - [MarketAccountsResponse](#api.MarketAccountsResponse)
    - [MarketByIDRequest](#api.MarketByIDRequest)
    - [MarketByIDResponse](#api.MarketByIDResponse)
    - [MarketDataByIDRequest](#api.MarketDataByIDRequest)
    - [MarketDataByIDResponse](#api.MarketDataByIDResponse)
    - [MarketDepthRequest](#api.MarketDepthRequest)
    - [MarketDepthResponse](#api.MarketDepthResponse)
    - [MarketDepthSubscribeRequest](#api.MarketDepthSubscribeRequest)
    - [MarketDepthUpdatesSubscribeRequest](#api.MarketDepthUpdatesSubscribeRequest)
    - [MarketsDataResponse](#api.MarketsDataResponse)
    - [MarketsDataSubscribeRequest](#api.MarketsDataSubscribeRequest)
    - [MarketsResponse](#api.MarketsResponse)
    - [NetworkParametersRequest](#api.NetworkParametersRequest)
    - [NetworkParametersResponse](#api.NetworkParametersResponse)
    - [ObserveEventsRequest](#api.ObserveEventsRequest)
    - [ObserveEventsResponse](#api.ObserveEventsResponse)
    - [ObservePartyProposalsRequest](#api.ObservePartyProposalsRequest)
    - [ObservePartyVotesRequest](#api.ObservePartyVotesRequest)
    - [ObserveProposalVotesRequest](#api.ObserveProposalVotesRequest)
    - [OptionalProposalState](#api.OptionalProposalState)
    - [OrderByIDRequest](#api.OrderByIDRequest)
    - [OrderByMarketAndIdRequest](#api.OrderByMarketAndIdRequest)
    - [OrderByMarketAndIdResponse](#api.OrderByMarketAndIdResponse)
    - [OrderByReferenceRequest](#api.OrderByReferenceRequest)
    - [OrderByReferenceResponse](#api.OrderByReferenceResponse)
    - [OrderVersionsByIDRequest](#api.OrderVersionsByIDRequest)
    - [OrderVersionsResponse](#api.OrderVersionsResponse)
    - [OrdersByMarketRequest](#api.OrdersByMarketRequest)
    - [OrdersByMarketResponse](#api.OrdersByMarketResponse)
    - [OrdersByPartyRequest](#api.OrdersByPartyRequest)
    - [OrdersByPartyResponse](#api.OrdersByPartyResponse)
    - [OrdersStream](#api.OrdersStream)
    - [OrdersSubscribeRequest](#api.OrdersSubscribeRequest)
    - [Pagination](#api.Pagination)
    - [PartiesResponse](#api.PartiesResponse)
    - [PartyAccountsRequest](#api.PartyAccountsRequest)
    - [PartyAccountsResponse](#api.PartyAccountsResponse)
    - [PartyByIDRequest](#api.PartyByIDRequest)
    - [PartyByIDResponse](#api.PartyByIDResponse)
    - [PositionsByPartyRequest](#api.PositionsByPartyRequest)
    - [PositionsByPartyResponse](#api.PositionsByPartyResponse)
    - [PositionsSubscribeRequest](#api.PositionsSubscribeRequest)
    - [PrepareAmendOrderResponse](#api.PrepareAmendOrderResponse)
    - [PrepareCancelOrderResponse](#api.PrepareCancelOrderResponse)
    - [PrepareLiquidityProvisionRequest](#api.PrepareLiquidityProvisionRequest)
    - [PrepareLiquidityProvisionResponse](#api.PrepareLiquidityProvisionResponse)
    - [PrepareProposalRequest](#api.PrepareProposalRequest)
    - [PrepareProposalResponse](#api.PrepareProposalResponse)
    - [PrepareSubmitOrderResponse](#api.PrepareSubmitOrderResponse)
    - [PrepareVoteRequest](#api.PrepareVoteRequest)
    - [PrepareVoteResponse](#api.PrepareVoteResponse)
    - [PrepareWithdrawRequest](#api.PrepareWithdrawRequest)
    - [PrepareWithdrawResponse](#api.PrepareWithdrawResponse)
    - [PropagateChainEventRequest](#api.PropagateChainEventRequest)
    - [PropagateChainEventResponse](#api.PropagateChainEventResponse)
    - [SubmitOrderRequest](#api.SubmitOrderRequest)
    - [SubmitTransactionRequest](#api.SubmitTransactionRequest)
    - [SubmitTransactionResponse](#api.SubmitTransactionResponse)
    - [TradesByMarketRequest](#api.TradesByMarketRequest)
    - [TradesByMarketResponse](#api.TradesByMarketResponse)
    - [TradesByOrderRequest](#api.TradesByOrderRequest)
    - [TradesByOrderResponse](#api.TradesByOrderResponse)
    - [TradesByPartyRequest](#api.TradesByPartyRequest)
    - [TradesByPartyResponse](#api.TradesByPartyResponse)
    - [TradesStream](#api.TradesStream)
    - [TradesSubscribeRequest](#api.TradesSubscribeRequest)
    - [VegaTimeResponse](#api.VegaTimeResponse)
    - [WithdrawalRequest](#api.WithdrawalRequest)
    - [WithdrawalResponse](#api.WithdrawalResponse)
    - [WithdrawalsRequest](#api.WithdrawalsRequest)
    - [WithdrawalsResponse](#api.WithdrawalsResponse)

    - [SubmitTransactionRequest.Type](#api.SubmitTransactionRequest.Type)

    - [trading](#api.trading)
    - [trading_data](#api.trading_data)

- [proto/assets.proto](#proto/assets.proto)
    - [Asset](#vega.Asset)
    - [AssetSource](#vega.AssetSource)
    - [BuiltinAsset](#vega.BuiltinAsset)
    - [DevAssets](#vega.DevAssets)
    - [ERC20](#vega.ERC20)

- [proto/chain_events.proto](#proto/chain_events.proto)
    - [AddValidator](#vega.AddValidator)
    - [BTCDeposit](#vega.BTCDeposit)
    - [BTCEvent](#vega.BTCEvent)
    - [BTCWithdrawal](#vega.BTCWithdrawal)
    - [BitcoinAddress](#vega.BitcoinAddress)
    - [BuiltinAssetDeposit](#vega.BuiltinAssetDeposit)
    - [BuiltinAssetEvent](#vega.BuiltinAssetEvent)
    - [BuiltinAssetWithdrawal](#vega.BuiltinAssetWithdrawal)
    - [ChainEvent](#vega.ChainEvent)
    - [ERC20AssetDelist](#vega.ERC20AssetDelist)
    - [ERC20AssetList](#vega.ERC20AssetList)
    - [ERC20Deposit](#vega.ERC20Deposit)
    - [ERC20Event](#vega.ERC20Event)
    - [ERC20Withdrawal](#vega.ERC20Withdrawal)
    - [EthereumAddress](#vega.EthereumAddress)
    - [Identifier](#vega.Identifier)
    - [RemoveValidator](#vega.RemoveValidator)
    - [ValidatorEvent](#vega.ValidatorEvent)

- [proto/events.proto](#proto/events.proto)
    - [AuctionEvent](#vega.AuctionEvent)
    - [BusEvent](#vega.BusEvent)
    - [LossSocialization](#vega.LossSocialization)
    - [MarketEvent](#vega.MarketEvent)
    - [MarketTick](#vega.MarketTick)
    - [PositionResolution](#vega.PositionResolution)
    - [SettleDistressed](#vega.SettleDistressed)
    - [SettlePosition](#vega.SettlePosition)
    - [TimeUpdate](#vega.TimeUpdate)
    - [TradeSettlement](#vega.TradeSettlement)
    - [TransferResponses](#vega.TransferResponses)
    - [TxErrorEvent](#vega.TxErrorEvent)

    - [BusEventType](#vega.BusEventType)

- [proto/governance.proto](#proto/governance.proto)
    - [FutureProduct](#vega.FutureProduct)
    - [GovernanceData](#vega.GovernanceData)
    - [GovernanceData.NoPartyEntry](#vega.GovernanceData.NoPartyEntry)
    - [GovernanceData.YesPartyEntry](#vega.GovernanceData.YesPartyEntry)
    - [InstrumentConfiguration](#vega.InstrumentConfiguration)
    - [NewAsset](#vega.NewAsset)
    - [NewMarket](#vega.NewMarket)
    - [NewMarketConfiguration](#vega.NewMarketConfiguration)
    - [Proposal](#vega.Proposal)
    - [ProposalTerms](#vega.ProposalTerms)
    - [UpdateMarket](#vega.UpdateMarket)
    - [UpdateNetworkParameter](#vega.UpdateNetworkParameter)
    - [Vote](#vega.Vote)

    - [Proposal.State](#vega.Proposal.State)
    - [ProposalError](#vega.ProposalError)
    - [Vote.Value](#vega.Vote.Value)

- [proto/markets.proto](#proto/markets.proto)
    - [AuctionDuration](#vega.AuctionDuration)
    - [ContinuousTrading](#vega.ContinuousTrading)
    - [DiscreteTrading](#vega.DiscreteTrading)
    - [EthereumEvent](#vega.EthereumEvent)
    - [FeeFactors](#vega.FeeFactors)
    - [Fees](#vega.Fees)
    - [Future](#vega.Future)
    - [Instrument](#vega.Instrument)
    - [InstrumentMetadata](#vega.InstrumentMetadata)
    - [LogNormalModelParams](#vega.LogNormalModelParams)
    - [LogNormalRiskModel](#vega.LogNormalRiskModel)
    - [MarginCalculator](#vega.MarginCalculator)
    - [Market](#vega.Market)
    - [PriceMonitoringParameters](#vega.PriceMonitoringParameters)
    - [PriceMonitoringSettings](#vega.PriceMonitoringSettings)
    - [PriceMonitoringTrigger](#vega.PriceMonitoringTrigger)
    - [ScalingFactors](#vega.ScalingFactors)
    - [SimpleModelParams](#vega.SimpleModelParams)
    - [SimpleRiskModel](#vega.SimpleRiskModel)
    - [TargetStakeParameters](#vega.TargetStakeParameters)
    - [TradableInstrument](#vega.TradableInstrument)

- [proto/tm/replay.proto](#proto/tm/replay.proto)
    - [BlockParams](#vega.BlockParams)
    - [ConsensusParams](#vega.ConsensusParams)
    - [Event](#vega.Event)
    - [EventAttribute](#vega.EventAttribute)
    - [Evidence](#vega.Evidence)
    - [EvidenceParams](#vega.EvidenceParams)
    - [Header](#vega.Header)
    - [LastCommitInfo](#vega.LastCommitInfo)
    - [PublicKey](#vega.PublicKey)
    - [RequestBeginBlock](#vega.RequestBeginBlock)
    - [RequestDeliverTx](#vega.RequestDeliverTx)
    - [RequestInitChain](#vega.RequestInitChain)
    - [ResponseBeginBlock](#vega.ResponseBeginBlock)
    - [ResponseDeliverTx](#vega.ResponseDeliverTx)
    - [ResponseInitChain](#vega.ResponseInitChain)
    - [TmEvent](#vega.TmEvent)
    - [Validator](#vega.Validator)
    - [ValidatorParams](#vega.ValidatorParams)
    - [ValidatorUpdate](#vega.ValidatorUpdate)
    - [VersionParams](#vega.VersionParams)
    - [VoteInfo](#vega.VoteInfo)

    - [EvidenceType](#vega.EvidenceType)

- [proto/vega.proto](#proto/vega.proto)
    - [Account](#vega.Account)
    - [AuctionIndicativeState](#vega.AuctionIndicativeState)
    - [Candle](#vega.Candle)
    - [Deposit](#vega.Deposit)
    - [Erc20WithdrawExt](#vega.Erc20WithdrawExt)
    - [ErrorDetail](#vega.ErrorDetail)
    - [Fee](#vega.Fee)
    - [FinancialAmount](#vega.FinancialAmount)
    - [LedgerEntry](#vega.LedgerEntry)
    - [LiquidityOrder](#vega.LiquidityOrder)
    - [LiquidityOrderReference](#vega.LiquidityOrderReference)
    - [LiquidityProvision](#vega.LiquidityProvision)
    - [LiquidityProvisionSubmission](#vega.LiquidityProvisionSubmission)
    - [MarginLevels](#vega.MarginLevels)
    - [MarketData](#vega.MarketData)
    - [MarketDepth](#vega.MarketDepth)
    - [MarketDepthUpdate](#vega.MarketDepthUpdate)
    - [NetworkParameter](#vega.NetworkParameter)
    - [NodeRegistration](#vega.NodeRegistration)
    - [NodeSignature](#vega.NodeSignature)
    - [NodeVote](#vega.NodeVote)
    - [Order](#vega.Order)
    - [OrderAmendment](#vega.OrderAmendment)
    - [OrderCancellation](#vega.OrderCancellation)
    - [OrderCancellationConfirmation](#vega.OrderCancellationConfirmation)
    - [OrderConfirmation](#vega.OrderConfirmation)
    - [OrderSubmission](#vega.OrderSubmission)
    - [Party](#vega.Party)
    - [PeggedOrder](#vega.PeggedOrder)
    - [Position](#vega.Position)
    - [PositionTrade](#vega.PositionTrade)
    - [Price](#vega.Price)
    - [PriceLevel](#vega.PriceLevel)
    - [PriceMonitoringBounds](#vega.PriceMonitoringBounds)
    - [RiskFactor](#vega.RiskFactor)
    - [RiskResult](#vega.RiskResult)
    - [RiskResult.PredictedNextRiskFactorsEntry](#vega.RiskResult.PredictedNextRiskFactorsEntry)
    - [RiskResult.RiskFactorsEntry](#vega.RiskResult.RiskFactorsEntry)
    - [Signature](#vega.Signature)
    - [SignedBundle](#vega.SignedBundle)
    - [Statistics](#vega.Statistics)
    - [Timestamp](#vega.Timestamp)
    - [Trade](#vega.Trade)
    - [TradeSet](#vega.TradeSet)
    - [Transaction](#vega.Transaction)
    - [Transfer](#vega.Transfer)
    - [TransferBalance](#vega.TransferBalance)
    - [TransferRequest](#vega.TransferRequest)
    - [TransferResponse](#vega.TransferResponse)
    - [WithdrawExt](#vega.WithdrawExt)
    - [WithdrawSubmission](#vega.WithdrawSubmission)
    - [Withdrawal](#vega.Withdrawal)

    - [AccountType](#vega.AccountType)
    - [AuctionTrigger](#vega.AuctionTrigger)
    - [ChainStatus](#vega.ChainStatus)
    - [Deposit.Status](#vega.Deposit.Status)
    - [Interval](#vega.Interval)
    - [LiquidityProvision.Status](#vega.LiquidityProvision.Status)
    - [MarketState](#vega.MarketState)
    - [NodeSignatureKind](#vega.NodeSignatureKind)
    - [Order.Status](#vega.Order.Status)
    - [Order.TimeInForce](#vega.Order.TimeInForce)
    - [Order.Type](#vega.Order.Type)
    - [OrderError](#vega.OrderError)
    - [PeggedReference](#vega.PeggedReference)
    - [Side](#vega.Side)
    - [Trade.Type](#vega.Trade.Type)
    - [TransferType](#vega.TransferType)
    - [Withdrawal.Status](#vega.Withdrawal.Status)

- [Scalar Value Types](#scalar-value-types)



<a name="proto/api/trading.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/api/trading.proto



<a name="api.AccountsSubscribeRequest"></a>

### AccountsSubscribeRequest
Request to subscribe to a stream of (Accounts)[#vega.Account].


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. |
| partyID | [string](#string) |  | Party identifier. |
| asset | [string](#string) |  | Asset identifier. |
| type | [vega.AccountType](#vega.AccountType) |  | Account type to subscribe to. Required field. |






<a name="api.AmendOrderRequest"></a>

### AmendOrderRequest
Request to amend an existing order.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| amendment | [vega.OrderAmendment](#vega.OrderAmendment) |  | An order amendment. |






<a name="api.AssetByIDRequest"></a>

### AssetByIDRequest
Request for an asset given an asset identifier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | Asset identifier. Required field. |






<a name="api.AssetByIDResponse"></a>

### AssetByIDResponse
Response for an asset given an asset identifier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| asset | [vega.Asset](#vega.Asset) |  | An asset record, if found. |






<a name="api.AssetsRequest"></a>

### AssetsRequest
Request for a list of all assets enabled on Vega.






<a name="api.AssetsResponse"></a>

### AssetsResponse
Response for a list of all assets enabled on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| assets | [vega.Asset](#vega.Asset) | repeated | A list of 0 or more assets. |






<a name="api.CancelOrderRequest"></a>

### CancelOrderRequest
Request to cancel an existing order.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cancellation | [vega.OrderCancellation](#vega.OrderCancellation) |  | An order cancellation. |






<a name="api.CandlesRequest"></a>

### CandlesRequest
Request for a list of candles for a market at an interval.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. Required field. |
| sinceTimestamp | [int64](#int64) |  | Timestamp to retrieve candles since, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. Required field. |
| interval | [vega.Interval](#vega.Interval) |  | Time interval for the candles. Required field. |






<a name="api.CandlesResponse"></a>

### CandlesResponse
Response for a list of candles for a market at an interval.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| candles | [vega.Candle](#vega.Candle) | repeated | A list of 0 or more candles. |






<a name="api.CandlesSubscribeRequest"></a>

### CandlesSubscribeRequest
Request to subscribe to a stream of (Candles)[#vega.Candle].


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. Required field. |
| interval | [vega.Interval](#vega.Interval) |  | Time interval for the candles. Required field. |






<a name="api.DepositRequest"></a>

### DepositRequest
A request to get a specific deposit by ID


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | The id of the withdrawal |






<a name="api.DepositResponse"></a>

### DepositResponse
A response for a deposit


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| deposit | [vega.Deposit](#vega.Deposit) |  | The deposit matching the ID from the request |






<a name="api.DepositsRequest"></a>

### DepositsRequest
A request to get a list of deposit from a given party


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | The party to get the deposits for |






<a name="api.DepositsResponse"></a>

### DepositsResponse
The response for a list of deposits


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| deposits | [vega.Deposit](#vega.Deposit) | repeated | The list of deposits for the specified party |






<a name="api.ERC20WithdrawalApprovalRequest"></a>

### ERC20WithdrawalApprovalRequest
The request to get all information required to bundle the call
to finalize the withdrawal on the erc20 bridge


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| withdrawalID | [string](#string) |  | The ID of the withdrawal |






<a name="api.ERC20WithdrawalApprovalResponse"></a>

### ERC20WithdrawalApprovalResponse
The response with all information required to bundle the call
to finalize the withdrawal on the erc20 bridge
function withdraw_asset(address asset_source, uint256 asset_id, uint256 amount, uint256 expiry, uint256 nonce, bytes memory signatures)


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| assetSource | [string](#string) |  | The address of asset on ethereum |
| amount | [string](#string) |  | The amount to be withdrawn |
| expiry | [int64](#int64) |  | The expiry / until what time the request is valid |
| nonce | [string](#string) |  | The nonce, which is actually the internal reference for the withdrawal |
| signatures | [string](#string) |  | The signatures bundle as hex encoded data, forward by 0x e.g: 0x &#43; sig1 &#43; sig2 &#43; ... &#43; sixN |






<a name="api.EstimateFeeRequest"></a>

### EstimateFeeRequest
Request to fetch the estimated fee if an order were to trade immediately


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| order | [vega.Order](#vega.Order) |  | Order to estimate fees for the following fields in the order are required: MarketID (used to specify the fee factors) Price (the price at which the order could trade) Size (the size at which the order could eventually trade) |






<a name="api.EstimateFeeResponse"></a>

### EstimateFeeResponse
Response to a EstimateFeeRequest, containing the estimated fees for a given order


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fee | [vega.Fee](#vega.Fee) |  | Summary of the estimated fees for this order if it were to trade now |






<a name="api.EstimateMarginRequest"></a>

### EstimateMarginRequest
Request to fetch the estimated MarginLevels if an order were to trade immediately


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| order | [vega.Order](#vega.Order) |  | Order to estimate fees for |






<a name="api.EstimateMarginResponse"></a>

### EstimateMarginResponse
Response to a EstimateMarginRequest, containing the estimated marginLevels for a given order


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marginLevels | [vega.MarginLevels](#vega.MarginLevels) |  | Summary of the estimated margins for this order if it were to trade now |






<a name="api.FeeInfrastructureAccountsRequest"></a>

### FeeInfrastructureAccountsRequest
Request for a list of infrastructure fee accounts.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| asset | [string](#string) |  | Asset identifier. Required field. Set to an empty string to return all accounts. Set to an asset ID to return a single infrastructure fee account for a given asset. |






<a name="api.FeeInfrastructureAccountsResponse"></a>

### FeeInfrastructureAccountsResponse
Response for a list of infrastructure fee accounts.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| accounts | [vega.Account](#vega.Account) | repeated | A list of 0 or more infrastructure fee accounts. |






<a name="api.GetNetworkParametersProposalsRequest"></a>

### GetNetworkParametersProposalsRequest
Request for a list of network parameter proposals.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  | Optional proposal state. |






<a name="api.GetNetworkParametersProposalsResponse"></a>

### GetNetworkParametersProposalsResponse
Response for a list of network parameter proposals.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated | A list of 0 or more governance data. |






<a name="api.GetNewAssetProposalsRequest"></a>

### GetNewAssetProposalsRequest
Request for a list of new asset proposals.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  | Optional proposal state. |






<a name="api.GetNewAssetProposalsResponse"></a>

### GetNewAssetProposalsResponse
Response for a list of new asset proposals.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated | A list of 0 or more governance data. |






<a name="api.GetNewMarketProposalsRequest"></a>

### GetNewMarketProposalsRequest
Request for a list of new market proposals.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  | Optional proposal state. |






<a name="api.GetNewMarketProposalsResponse"></a>

### GetNewMarketProposalsResponse
Response for a list of new market proposals.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated | A list of 0 or more governance data. |






<a name="api.GetNodeSignaturesAggregateRequest"></a>

### GetNodeSignaturesAggregateRequest
Request to specify the identifier of the resource we want to retrieve aggregated signatures for.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | Resource identifier. Required field. |






<a name="api.GetNodeSignaturesAggregateResponse"></a>

### GetNodeSignaturesAggregateResponse
Response to specify the identifier of the resource we want to retrieve aggregated signatures for.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| signatures | [vega.NodeSignature](#vega.NodeSignature) | repeated | A list of 0 or more signatures. |






<a name="api.GetProposalByIDRequest"></a>

### GetProposalByIDRequest
Request for a governance proposal given a proposal identifier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| proposalID | [string](#string) |  | Proposal identifier. Required field. |






<a name="api.GetProposalByIDResponse"></a>

### GetProposalByIDResponse
Response for a governance proposal given a proposal identifier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) |  | Governance data, if found. |






<a name="api.GetProposalByReferenceRequest"></a>

### GetProposalByReferenceRequest
Request for a governance proposal given a proposal reference.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reference | [string](#string) |  | Proposal reference. Required field. |






<a name="api.GetProposalByReferenceResponse"></a>

### GetProposalByReferenceResponse
Response for a governance proposal given a proposal reference.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) |  | Governance data, if found. |






<a name="api.GetProposalsByPartyRequest"></a>

### GetProposalsByPartyRequest
Request for a list of proposals for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Required field. |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  | Optional proposal state. |






<a name="api.GetProposalsByPartyResponse"></a>

### GetProposalsByPartyResponse
Response for a list of proposals for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated | A list of 0 or more governance data. |






<a name="api.GetProposalsRequest"></a>

### GetProposalsRequest
Request for a list of proposals.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  | Optional proposal state. |






<a name="api.GetProposalsResponse"></a>

### GetProposalsResponse
Response for a list of proposals.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated | A list of 0 or more governance data. |






<a name="api.GetUpdateMarketProposalsRequest"></a>

### GetUpdateMarketProposalsRequest
Request for a list of update market proposals.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. Required field. |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  | Proposal state. |






<a name="api.GetUpdateMarketProposalsResponse"></a>

### GetUpdateMarketProposalsResponse
Response for a list of update market proposals.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated | A list of 0 or more governance data. |






<a name="api.GetVotesByPartyRequest"></a>

### GetVotesByPartyRequest
Request for a list of votes for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Required field. |






<a name="api.GetVotesByPartyResponse"></a>

### GetVotesByPartyResponse
Response for a list of votes for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| votes | [vega.Vote](#vega.Vote) | repeated | A list of 0 or more votes. |






<a name="api.LastTradeRequest"></a>

### LastTradeRequest
Request for the latest trade that occurred on Vega for a given market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. Required field. |






<a name="api.LastTradeResponse"></a>

### LastTradeResponse
Response for the latest trade that occurred on Vega for a given market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trade | [vega.Trade](#vega.Trade) |  | A trade, if found. |






<a name="api.LiquidityProvisionsRequest"></a>

### LiquidityProvisionsRequest
A message requesting for the list
of liquidity provisions orders for markets
one of the 2 filters is required or both


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| market | [string](#string) |  | the market we want to get liquidity provision orders from |
| party | [string](#string) |  | the party which submitted the liquidity provision orders |






<a name="api.LiquidityProvisionsResponse"></a>

### LiquidityProvisionsResponse
A response containing all of the
vega liquidity provisions orders


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| liquidityProvisions | [vega.LiquidityProvision](#vega.LiquidityProvision) | repeated |  |






<a name="api.MarginLevelsRequest"></a>

### MarginLevelsRequest
Request for margin levels for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Required field. |
| marketID | [string](#string) |  | Market identifier. |






<a name="api.MarginLevelsResponse"></a>

### MarginLevelsResponse
Response for margin levels for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marginLevels | [vega.MarginLevels](#vega.MarginLevels) | repeated | A list of 0 or more margin levels. |






<a name="api.MarginLevelsSubscribeRequest"></a>

### MarginLevelsSubscribeRequest
Request to subscribe to a stream of MarginLevels data matching the given party identifier.
Optionally, the list can be additionally filtered by market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Required field. |
| marketID | [string](#string) |  | Market identifier. |






<a name="api.MarketAccountsRequest"></a>

### MarketAccountsRequest
Request for a list of accounts for a market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. |
| asset | [string](#string) |  | Asset identifier. |






<a name="api.MarketAccountsResponse"></a>

### MarketAccountsResponse
Response for a list of accounts for a market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| accounts | [vega.Account](#vega.Account) | repeated | A list of 0 or more accounts. |






<a name="api.MarketByIDRequest"></a>

### MarketByIDRequest
Request for a market given a market identifier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. Required field. |






<a name="api.MarketByIDResponse"></a>

### MarketByIDResponse
Response for a market given a market identifier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| market | [vega.Market](#vega.Market) |  | A market, if found. |






<a name="api.MarketDataByIDRequest"></a>

### MarketDataByIDRequest
Request for market data for a market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. |






<a name="api.MarketDataByIDResponse"></a>

### MarketDataByIDResponse
Response for market data for a market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketData | [vega.MarketData](#vega.MarketData) |  | Market data, if found. |






<a name="api.MarketDepthRequest"></a>

### MarketDepthRequest
Request for the market depth/order book price levels on a market.
Optionally, a maximum depth can be set to limit the number of levels returned.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. Required field. |
| maxDepth | [uint64](#uint64) |  | Max depth limits the number of levels returned. Default is 0, which returns all levels. |






<a name="api.MarketDepthResponse"></a>

### MarketDepthResponse
Response for the market depth/order book price levels on a market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. |
| buy | [vega.PriceLevel](#vega.PriceLevel) | repeated | Zero or more price levels for the buy side of the market depth data. |
| sell | [vega.PriceLevel](#vega.PriceLevel) | repeated | Zero or more price levels for the sell side of the market depth data. |
| lastTrade | [vega.Trade](#vega.Trade) |  | Last trade recorded on Vega at the time of retrieving the `MarketDepthResponse`. |
| sequenceNumber | [uint64](#uint64) |  | Sequence number incremented after each update |






<a name="api.MarketDepthSubscribeRequest"></a>

### MarketDepthSubscribeRequest
Request to subscribe to a stream of (MarketDepth)[#vega.MarketDepth] data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. Required field. |






<a name="api.MarketDepthUpdatesSubscribeRequest"></a>

### MarketDepthUpdatesSubscribeRequest
Request to subscribe to a stream of (MarketDepth Update)[#vega.MarketDepthUpdate] data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. Required field. |






<a name="api.MarketsDataResponse"></a>

### MarketsDataResponse
Response for market data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketsData | [vega.MarketData](#vega.MarketData) | repeated | A list of 0 or more market data. |






<a name="api.MarketsDataSubscribeRequest"></a>

### MarketsDataSubscribeRequest
Request to subscribe to a stream of MarketsData.
Optionally, the list can be additionally filtered by market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. |






<a name="api.MarketsResponse"></a>

### MarketsResponse
Response for a list of markets on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| markets | [vega.Market](#vega.Market) | repeated | A list of 0 or more markets. |






<a name="api.NetworkParametersRequest"></a>

### NetworkParametersRequest
A message requesting for the list
of all network parameters






<a name="api.NetworkParametersResponse"></a>

### NetworkParametersResponse
A response containing all of the
vega network parameters


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| networkParameters | [vega.NetworkParameter](#vega.NetworkParameter) | repeated |  |






<a name="api.ObserveEventsRequest"></a>

### ObserveEventsRequest
Request to subscribe to a stream of one or more event types from the Vega event bus


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [vega.BusEventType](#vega.BusEventType) | repeated | One or more types of event. Required field. |
| marketID | [string](#string) |  | Market identifier. Optional field. |
| partyID | [string](#string) |  | Party identifier. Optional field. |
| batchSize | [int64](#int64) |  | Batch size. Optional field. If not specified, any events received will be sent immediately. If the client is not ready for the next data-set, data may be dropped a number of times, and eventually the stream is closed. if specified, the first batch will be sent when ready. To receive the next set of events, the client must write an `ObserveEventBatch` message on the stream to flush the buffer. If no message is received in 5 seconds, the stream is closed. Default: 0, send any and all events when they are available. |






<a name="api.ObserveEventsResponse"></a>

### ObserveEventsResponse
Response to a subscribed stream of events from the Vega event bus


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| events | [vega.BusEvent](#vega.BusEvent) | repeated | One or more events |






<a name="api.ObservePartyProposalsRequest"></a>

### ObservePartyProposalsRequest
Request to subscribe to a stream of governance proposals for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Required field. |






<a name="api.ObservePartyVotesRequest"></a>

### ObservePartyVotesRequest
Request to subscribe to a stream of governance votes for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Required field. |






<a name="api.ObserveProposalVotesRequest"></a>

### ObserveProposalVotesRequest
Request to subscribe to a stream of governance votes for a proposal.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| proposalID | [string](#string) |  | Proposal identifier. Required field. |






<a name="api.OptionalProposalState"></a>

### OptionalProposalState
Optional proposal state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [vega.Proposal.State](#vega.Proposal.State) |  | Proposal state value. |






<a name="api.OrderByIDRequest"></a>

### OrderByIDRequest
Request for an order with the specified order identifier.
Optionally, return a specific version of the order with the `version` field.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  | Order identifier. Required field. |
| version | [uint64](#uint64) |  | Version of the order. Set `version` to 0 for most recent version of the order. Set `1` for original version of the order. Set `2` for first amendment, `3` for second amendment, etc. |






<a name="api.OrderByMarketAndIdRequest"></a>

### OrderByMarketAndIdRequest
Request for an order on a market given an order identifier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. Required field. |
| orderID | [string](#string) |  | Order identifier. Required field. |






<a name="api.OrderByMarketAndIdResponse"></a>

### OrderByMarketAndIdResponse
Response for an order on a market given an order identifier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| order | [vega.Order](#vega.Order) |  | An order, if found. |






<a name="api.OrderByReferenceRequest"></a>

### OrderByReferenceRequest
Request for an order given an order reference.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reference | [string](#string) |  | Unique reference. Required field. |






<a name="api.OrderByReferenceResponse"></a>

### OrderByReferenceResponse
Response for an order given an order reference.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| order | [vega.Order](#vega.Order) |  | An order, if found. |






<a name="api.OrderVersionsByIDRequest"></a>

### OrderVersionsByIDRequest
Request for a list of all versions of an order given the specified order identifier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  | Order identifier. Required field. |
| pagination | [Pagination](#api.Pagination) |  | Pagination controls. |






<a name="api.OrderVersionsResponse"></a>

### OrderVersionsResponse
Response to a request for a list of all versions of an order.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orders | [vega.Order](#vega.Order) | repeated | A list of 0 or more orders (list will contain the same order but with different versions, if it has been amended). |






<a name="api.OrdersByMarketRequest"></a>

### OrdersByMarketRequest
Request for a list of orders for a market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. Required field. |
| pagination | [Pagination](#api.Pagination) |  | Optional pagination controls. |






<a name="api.OrdersByMarketResponse"></a>

### OrdersByMarketResponse
Response for a list of orders for a market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orders | [vega.Order](#vega.Order) | repeated | A list of 0 or more orders. |






<a name="api.OrdersByPartyRequest"></a>

### OrdersByPartyRequest
Request for a list of orders for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Required field. |
| pagination | [Pagination](#api.Pagination) |  | Pagination controls. |






<a name="api.OrdersByPartyResponse"></a>

### OrdersByPartyResponse
Response for a list of orders for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orders | [vega.Order](#vega.Order) | repeated | A list of 0 or more orders. |






<a name="api.OrdersStream"></a>

### OrdersStream
A stream of orders.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orders | [vega.Order](#vega.Order) | repeated | A list of 0 or more orders. |






<a name="api.OrdersSubscribeRequest"></a>

### OrdersSubscribeRequest
Request to subscribe to a stream of (Orders)[#vega.Order].

The fields for market identifier and party identifier are optional filters.
If omitted all orders, for all parties on all markets will be returned on the stream.
If market identifier is given, orders from that market will be returned on the stream.
If party identifier is given, orders from that party will be returned on the stream.
Both filters can be combined.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. |
| partyID | [string](#string) |  | Party identifier. |






<a name="api.Pagination"></a>

### Pagination
Pagination controls.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| skip | [uint64](#uint64) |  | Skip the number of records specified. Default is 0. |
| limit | [uint64](#uint64) |  | Limit the number of returned records to the value specified. Default is 50. |
| descending | [bool](#bool) |  | Descending reverses the order of the records returned. Default is true, if false the results will be returned in ascending order. |






<a name="api.PartiesResponse"></a>

### PartiesResponse
Response to a request for a list of parties.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parties | [vega.Party](#vega.Party) | repeated | A list of 0 or more parties. |






<a name="api.PartyAccountsRequest"></a>

### PartyAccountsRequest
Request for a list of accounts for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. |
| marketID | [string](#string) |  | Market identifier. |
| type | [vega.AccountType](#vega.AccountType) |  | Account type. Required field. |
| asset | [string](#string) |  | Asset identifier. |






<a name="api.PartyAccountsResponse"></a>

### PartyAccountsResponse
Response for a list of accounts for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| accounts | [vega.Account](#vega.Account) | repeated | A list of 0 or more accounts. |






<a name="api.PartyByIDRequest"></a>

### PartyByIDRequest
Request for a party given a party identifier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Required field. |






<a name="api.PartyByIDResponse"></a>

### PartyByIDResponse
Response for a party given a party identifier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| party | [vega.Party](#vega.Party) |  | A party, if found. |






<a name="api.PositionsByPartyRequest"></a>

### PositionsByPartyRequest
Request for a list of positions for a party.
Optionally, if a market identifier is set, the results will be filtered for that market only.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Required field. |
| marketID | [string](#string) |  | Market identifier. |






<a name="api.PositionsByPartyResponse"></a>

### PositionsByPartyResponse
Response for a list of positions for a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| positions | [vega.Position](#vega.Position) | repeated | A list of 0 or more positions. |






<a name="api.PositionsSubscribeRequest"></a>

### PositionsSubscribeRequest
Request to subscribe to a stream of (Positions)[#vega.Position].


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Optional field. |
| marketID | [string](#string) |  | Market identifier. Optional field. |






<a name="api.PrepareAmendOrderResponse"></a>

### PrepareAmendOrderResponse
Response for preparing an order amendment.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  | blob is an encoded representation of the order amendment ready to sign using the Vega Wallet and then submit as a transaction. |






<a name="api.PrepareCancelOrderResponse"></a>

### PrepareCancelOrderResponse
Response for preparing an order cancellation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  | blob is an encoded representation of the order cancellation ready to sign using the Vega Wallet and then submit as a transaction. |






<a name="api.PrepareLiquidityProvisionRequest"></a>

### PrepareLiquidityProvisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| submission | [vega.LiquidityProvisionSubmission](#vega.LiquidityProvisionSubmission) |  |  |






<a name="api.PrepareLiquidityProvisionResponse"></a>

### PrepareLiquidityProvisionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  |  |






<a name="api.PrepareProposalRequest"></a>

### PrepareProposalRequest
Request to prepare a governance proposal.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Required field. |
| reference | [string](#string) |  | Unique reference. |
| proposal | [vega.ProposalTerms](#vega.ProposalTerms) |  | Proposal terms. Required field. |






<a name="api.PrepareProposalResponse"></a>

### PrepareProposalResponse
Response to prepare a governance proposal.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  | blob is an encoded representation of the proposal ready to sign using the Vega Wallet and then submit as a transaction. |
| pendingProposal | [vega.Proposal](#vega.Proposal) |  | A copy of the prepared proposal. |






<a name="api.PrepareSubmitOrderResponse"></a>

### PrepareSubmitOrderResponse
Response for preparing an order submission.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  | blob is an encoded representation of the order submission ready to sign using the Vega Wallet and then submit as a transaction. |
| submitID | [string](#string) |  | Submission identifier (order reference). |






<a name="api.PrepareVoteRequest"></a>

### PrepareVoteRequest
Request to prepare a governance vote.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vote | [vega.Vote](#vega.Vote) |  | Vote. Required field. |






<a name="api.PrepareVoteResponse"></a>

### PrepareVoteResponse
Response to prepare a governance vote.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  | blob is an encoded representation of the vote ready to sign using the Vega Wallet and then submit as a transaction. |
| vote | [vega.Vote](#vega.Vote) |  | A copy of the prepared vote. |






<a name="api.PrepareWithdrawRequest"></a>

### PrepareWithdrawRequest
Request for preparing a withdrawal.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| withdraw | [vega.WithdrawSubmission](#vega.WithdrawSubmission) |  | An asset withdrawal. |






<a name="api.PrepareWithdrawResponse"></a>

### PrepareWithdrawResponse
Response for preparing a withdrawal.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  | blob is an encoded representation of the withdrawal ready to sign using the Vega Wallet and then submit as a transaction. |






<a name="api.PropagateChainEventRequest"></a>

### PropagateChainEventRequest
Request for a new event sent by the blockchain queue to be propagated on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| evt | [vega.ChainEvent](#vega.ChainEvent) |  | Chain event. |
| pubKey | [string](#string) |  | Public key. |
| signature | [bytes](#bytes) |  | Signature. |






<a name="api.PropagateChainEventResponse"></a>

### PropagateChainEventResponse
Response for a new event sent by the blockchain queue to be propagated on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  | Success will be true if the event was accepted by the node. Important - success does not mean that the event is confirmed by consensus. |






<a name="api.SubmitOrderRequest"></a>

### SubmitOrderRequest
Request to submit a new order.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| submission | [vega.OrderSubmission](#vega.OrderSubmission) |  | An order submission. |






<a name="api.SubmitTransactionRequest"></a>

### SubmitTransactionRequest
Request for submitting a transaction on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tx | [vega.SignedBundle](#vega.SignedBundle) |  | A bundle of signed payload and signature, to form a transaction that will be submitted to the Vega blockchain. |
| type | [SubmitTransactionRequest.Type](#api.SubmitTransactionRequest.Type) |  |  |






<a name="api.SubmitTransactionResponse"></a>

### SubmitTransactionResponse
Response for submitting a transaction on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  | Success will be true if the transaction was accepted by the node. Important - success does not mean that the transaction is confirmed by consensus. |






<a name="api.TradesByMarketRequest"></a>

### TradesByMarketRequest
Request for a list of trades on a market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. Required field. |
| pagination | [Pagination](#api.Pagination) |  | Pagination controls. |






<a name="api.TradesByMarketResponse"></a>

### TradesByMarketResponse
Response for a list of trades on a market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trades | [vega.Trade](#vega.Trade) | repeated | A list of 0 or more trades. |






<a name="api.TradesByOrderRequest"></a>

### TradesByOrderRequest
Request for a list of trades related to an order.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  | Order identifier. Required field. |






<a name="api.TradesByOrderResponse"></a>

### TradesByOrderResponse
Response for a list of trades related to an order.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trades | [vega.Trade](#vega.Trade) | repeated | A list of 0 or more trades. |






<a name="api.TradesByPartyRequest"></a>

### TradesByPartyRequest
Request for a list of trades relating to the given party.
Optionally, the list can be additionally filtered for trades by market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Party identifier. Required field. |
| marketID | [string](#string) |  | Market identifier. |
| pagination | [Pagination](#api.Pagination) |  | Pagination controls. |






<a name="api.TradesByPartyResponse"></a>

### TradesByPartyResponse
Response for a list of trades relating to a party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trades | [vega.Trade](#vega.Trade) | repeated | A list of 0 or more trades. |






<a name="api.TradesStream"></a>

### TradesStream
A stream of trades.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trades | [vega.Trade](#vega.Trade) | repeated | A list of 0 or more trades. |






<a name="api.TradesSubscribeRequest"></a>

### TradesSubscribeRequest
Request to subscribe to a stream of (Trades)[#vega.Trade].

The fields for market identifier and party identifier are optional filters.
If omitted all trades, for all parties on all markets will be returned on the stream.
If market identifier is given, trades from that market will be returned on the stream.
If party identifier is given, trades from that party will be returned on the stream.
Both filters can be combined.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. |
| partyID | [string](#string) |  | Party identifier. |






<a name="api.VegaTimeResponse"></a>

### VegaTimeResponse
Response for the current consensus coordinated time on the Vega network, referred to as &#34;VegaTime&#34;.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [int64](#int64) |  | Timestamp representation of current VegaTime. Nanoseconds since the epoch, for example `1580473859111222333` corresponds to `2020-01-31T12:30:59.111222333Z` |






<a name="api.WithdrawalRequest"></a>

### WithdrawalRequest
A request to get a specific withdrawal by ID


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | The id of the withdrawal |






<a name="api.WithdrawalResponse"></a>

### WithdrawalResponse
A response for a withdrawal


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| withdrawal | [vega.Withdrawal](#vega.Withdrawal) |  | The withdrawal matching the ID from the request |






<a name="api.WithdrawalsRequest"></a>

### WithdrawalsRequest
A request to get a list of withdrawal from a given party


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | The party to get the withdrawals for |






<a name="api.WithdrawalsResponse"></a>

### WithdrawalsResponse
The response for a list of withdrawals


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| withdrawals | [vega.Withdrawal](#vega.Withdrawal) | repeated | The list of withdrawals for the specified party |








<a name="api.SubmitTransactionRequest.Type"></a>

### SubmitTransactionRequest.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_ASYNC | 1 | The transaction will be submitted without waiting for response. |
| TYPE_SYNC | 2 | The transaction will be submitted, and blocking until the tendermint mempool return a response. |
| TYPE_COMMIT | 3 | The transaction will submitted, and blocking until the tendermint network will have committed it into a block. |







<a name="api.trading"></a>

### trading


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| PrepareSubmitOrder | [SubmitOrderRequest](#api.SubmitOrderRequest) | [PrepareSubmitOrderResponse](#api.PrepareSubmitOrderResponse) | Prepare a submit order request |
| PrepareCancelOrder | [CancelOrderRequest](#api.CancelOrderRequest) | [PrepareCancelOrderResponse](#api.PrepareCancelOrderResponse) | Prepare a cancel order request |
| PrepareAmendOrder | [AmendOrderRequest](#api.AmendOrderRequest) | [PrepareAmendOrderResponse](#api.PrepareAmendOrderResponse) | Prepare an amend order request |
| PrepareWithdraw | [PrepareWithdrawRequest](#api.PrepareWithdrawRequest) | [PrepareWithdrawResponse](#api.PrepareWithdrawResponse) | Request a withdrawal |
| SubmitTransaction | [SubmitTransactionRequest](#api.SubmitTransactionRequest) | [SubmitTransactionResponse](#api.SubmitTransactionResponse) | Submit a signed transaction |
| PrepareProposal | [PrepareProposalRequest](#api.PrepareProposalRequest) | [PrepareProposalResponse](#api.PrepareProposalResponse) | Prepare a governance proposal |
| PrepareVote | [PrepareVoteRequest](#api.PrepareVoteRequest) | [PrepareVoteResponse](#api.PrepareVoteResponse) | Prepare a governance vote |
| PropagateChainEvent | [PropagateChainEventRequest](#api.PropagateChainEventRequest) | [PropagateChainEventResponse](#api.PropagateChainEventResponse) | Propagate a chain event |
| PrepareLiquidityProvision | [PrepareLiquidityProvisionRequest](#api.PrepareLiquidityProvisionRequest) | [PrepareLiquidityProvisionResponse](#api.PrepareLiquidityProvisionResponse) |  |


<a name="api.trading_data"></a>

### trading_data


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| MarketAccounts | [MarketAccountsRequest](#api.MarketAccountsRequest) | [MarketAccountsResponse](#api.MarketAccountsResponse) | Get a list of Accounts by Market |
| PartyAccounts | [PartyAccountsRequest](#api.PartyAccountsRequest) | [PartyAccountsResponse](#api.PartyAccountsResponse) | Get a list of Accounts by Party |
| FeeInfrastructureAccounts | [FeeInfrastructureAccountsRequest](#api.FeeInfrastructureAccountsRequest) | [FeeInfrastructureAccountsResponse](#api.FeeInfrastructureAccountsResponse) | Get the list of infrastructure fees accounts filter eventually by assets |
| Candles | [CandlesRequest](#api.CandlesRequest) | [CandlesResponse](#api.CandlesResponse) | Get a list of Candles by Market |
| MarketDataByID | [MarketDataByIDRequest](#api.MarketDataByIDRequest) | [MarketDataByIDResponse](#api.MarketDataByIDResponse) | Get Market Data by MarketID |
| MarketsData | [.google.protobuf.Empty](#google.protobuf.Empty) | [MarketsDataResponse](#api.MarketsDataResponse) | Get a list of Market Data |
| MarketByID | [MarketByIDRequest](#api.MarketByIDRequest) | [MarketByIDResponse](#api.MarketByIDResponse) | Get a Market by ID |
| MarketDepth | [MarketDepthRequest](#api.MarketDepthRequest) | [MarketDepthResponse](#api.MarketDepthResponse) | Get Market Depth |
| Markets | [.google.protobuf.Empty](#google.protobuf.Empty) | [MarketsResponse](#api.MarketsResponse) | Get a list of Markets |
| OrderByMarketAndID | [OrderByMarketAndIdRequest](#api.OrderByMarketAndIdRequest) | [OrderByMarketAndIdResponse](#api.OrderByMarketAndIdResponse) | Get an Order by Market and OrderID |
| OrderByReference | [OrderByReferenceRequest](#api.OrderByReferenceRequest) | [OrderByReferenceResponse](#api.OrderByReferenceResponse) | Get an Order by Pending Order reference (UUID) |
| OrdersByMarket | [OrdersByMarketRequest](#api.OrdersByMarketRequest) | [OrdersByMarketResponse](#api.OrdersByMarketResponse) | Get a list of Orders by Market |
| OrdersByParty | [OrdersByPartyRequest](#api.OrdersByPartyRequest) | [OrdersByPartyResponse](#api.OrdersByPartyResponse) | Get a list of Orders by Party |
| OrderByID | [OrderByIDRequest](#api.OrderByIDRequest) | [.vega.Order](#vega.Order) | Get a specific order by orderID |
| OrderVersionsByID | [OrderVersionsByIDRequest](#api.OrderVersionsByIDRequest) | [OrderVersionsResponse](#api.OrderVersionsResponse) | Get all versions of the order by its orderID |
| MarginLevels | [MarginLevelsRequest](#api.MarginLevelsRequest) | [MarginLevelsResponse](#api.MarginLevelsResponse) | Get Margin Levels by PartyID |
| Parties | [.google.protobuf.Empty](#google.protobuf.Empty) | [PartiesResponse](#api.PartiesResponse) | Get a list of Parties |
| PartyByID | [PartyByIDRequest](#api.PartyByIDRequest) | [PartyByIDResponse](#api.PartyByIDResponse) | Get a Party by ID |
| PositionsByParty | [PositionsByPartyRequest](#api.PositionsByPartyRequest) | [PositionsByPartyResponse](#api.PositionsByPartyResponse) | Get a list of Positions by Party |
| LastTrade | [LastTradeRequest](#api.LastTradeRequest) | [LastTradeResponse](#api.LastTradeResponse) | Get latest Trade |
| TradesByMarket | [TradesByMarketRequest](#api.TradesByMarketRequest) | [TradesByMarketResponse](#api.TradesByMarketResponse) | Get a list of Trades by Market |
| TradesByOrder | [TradesByOrderRequest](#api.TradesByOrderRequest) | [TradesByOrderResponse](#api.TradesByOrderResponse) | Get a list of Trades by Order |
| TradesByParty | [TradesByPartyRequest](#api.TradesByPartyRequest) | [TradesByPartyResponse](#api.TradesByPartyResponse) | Get a list of Trades by Party |
| GetProposals | [GetProposalsRequest](#api.GetProposalsRequest) | [GetProposalsResponse](#api.GetProposalsResponse) | Get governance data (proposals and votes) for all proposals |
| GetProposalsByParty | [GetProposalsByPartyRequest](#api.GetProposalsByPartyRequest) | [GetProposalsByPartyResponse](#api.GetProposalsByPartyResponse) | Get governance data (proposals and votes) for proposals by party authoring them |
| GetVotesByParty | [GetVotesByPartyRequest](#api.GetVotesByPartyRequest) | [GetVotesByPartyResponse](#api.GetVotesByPartyResponse) | Get votes by party casting them |
| GetNewMarketProposals | [GetNewMarketProposalsRequest](#api.GetNewMarketProposalsRequest) | [GetNewMarketProposalsResponse](#api.GetNewMarketProposalsResponse) | Get governance data (proposals and votes) for proposals that aim creating new markets |
| GetUpdateMarketProposals | [GetUpdateMarketProposalsRequest](#api.GetUpdateMarketProposalsRequest) | [GetUpdateMarketProposalsResponse](#api.GetUpdateMarketProposalsResponse) | Get governance data (proposals and votes) for proposals that aim updating markets |
| GetNetworkParametersProposals | [GetNetworkParametersProposalsRequest](#api.GetNetworkParametersProposalsRequest) | [GetNetworkParametersProposalsResponse](#api.GetNetworkParametersProposalsResponse) | Get governance data (proposals and votes) for proposals that aim updating Vega network parameters |
| GetNewAssetProposals | [GetNewAssetProposalsRequest](#api.GetNewAssetProposalsRequest) | [GetNewAssetProposalsResponse](#api.GetNewAssetProposalsResponse) | Get governance data (proposals and votes) for proposals aiming to create new assets |
| GetProposalByID | [GetProposalByIDRequest](#api.GetProposalByIDRequest) | [GetProposalByIDResponse](#api.GetProposalByIDResponse) | Get governance data (proposals and votes) for a proposal located by ID |
| GetProposalByReference | [GetProposalByReferenceRequest](#api.GetProposalByReferenceRequest) | [GetProposalByReferenceResponse](#api.GetProposalByReferenceResponse) | Get governance data (proposals and votes) for a proposal located by reference |
| ObserveGovernance | [.google.protobuf.Empty](#google.protobuf.Empty) | [.vega.GovernanceData](#vega.GovernanceData) stream | Subscribe to a stream of all governance updates |
| ObservePartyProposals | [ObservePartyProposalsRequest](#api.ObservePartyProposalsRequest) | [.vega.GovernanceData](#vega.GovernanceData) stream | Subscribe to a stream of proposal updates |
| ObservePartyVotes | [ObservePartyVotesRequest](#api.ObservePartyVotesRequest) | [.vega.Vote](#vega.Vote) stream | Subscribe to a stream of votes cast by a specific party |
| ObserveProposalVotes | [ObserveProposalVotesRequest](#api.ObserveProposalVotesRequest) | [.vega.Vote](#vega.Vote) stream | Subscribe to a stream of proposal votes |
| ObserveEventBus | [ObserveEventsRequest](#api.ObserveEventsRequest) stream | [ObserveEventsResponse](#api.ObserveEventsResponse) stream | Subscribe to a stream of events from the core |
| Statistics | [.google.protobuf.Empty](#google.protobuf.Empty) | [.vega.Statistics](#vega.Statistics) | Get Statistics |
| GetVegaTime | [.google.protobuf.Empty](#google.protobuf.Empty) | [VegaTimeResponse](#api.VegaTimeResponse) | Get Time |
| AccountsSubscribe | [AccountsSubscribeRequest](#api.AccountsSubscribeRequest) | [.vega.Account](#vega.Account) stream | Subscribe to a stream of Accounts |
| CandlesSubscribe | [CandlesSubscribeRequest](#api.CandlesSubscribeRequest) | [.vega.Candle](#vega.Candle) stream | Subscribe to a stream of Candles |
| MarginLevelsSubscribe | [MarginLevelsSubscribeRequest](#api.MarginLevelsSubscribeRequest) | [.vega.MarginLevels](#vega.MarginLevels) stream | Subscribe to a stream of Margin Levels |
| MarketDepthSubscribe | [MarketDepthSubscribeRequest](#api.MarketDepthSubscribeRequest) | [.vega.MarketDepth](#vega.MarketDepth) stream | Subscribe to a stream of Market Depth |
| MarketDepthUpdatesSubscribe | [MarketDepthUpdatesSubscribeRequest](#api.MarketDepthUpdatesSubscribeRequest) | [.vega.MarketDepthUpdate](#vega.MarketDepthUpdate) stream | Subscribe to a stream of Market Depth PriceLevel Updates |
| MarketsDataSubscribe | [MarketsDataSubscribeRequest](#api.MarketsDataSubscribeRequest) | [.vega.MarketData](#vega.MarketData) stream | Subscribe to a stream of Markets Data |
| OrdersSubscribe | [OrdersSubscribeRequest](#api.OrdersSubscribeRequest) | [OrdersStream](#api.OrdersStream) stream | Subscribe to a stream of Orders |
| PositionsSubscribe | [PositionsSubscribeRequest](#api.PositionsSubscribeRequest) | [.vega.Position](#vega.Position) stream | Subscribe to a stream of Positions |
| TradesSubscribe | [TradesSubscribeRequest](#api.TradesSubscribeRequest) | [TradesStream](#api.TradesStream) stream | Subscribe to a stream of Trades |
| TransferResponsesSubscribe | [.google.protobuf.Empty](#google.protobuf.Empty) | [.vega.TransferResponse](#vega.TransferResponse) stream | Subscribe to a stream of Transfer Responses |
| GetNodeSignaturesAggregate | [GetNodeSignaturesAggregateRequest](#api.GetNodeSignaturesAggregateRequest) | [GetNodeSignaturesAggregateResponse](#api.GetNodeSignaturesAggregateResponse) | Get an aggregate of signatures from all the nodes of the network. |
| AssetByID | [AssetByIDRequest](#api.AssetByIDRequest) | [AssetByIDResponse](#api.AssetByIDResponse) | Get an asset by its identifier. |
| Assets | [AssetsRequest](#api.AssetsRequest) | [AssetsResponse](#api.AssetsResponse) | Get a list of all assets on Vega. |
| EstimateFee | [EstimateFeeRequest](#api.EstimateFeeRequest) | [EstimateFeeResponse](#api.EstimateFeeResponse) | Get an estimate for the fee to be paid for a given order |
| EstimateMargin | [EstimateMarginRequest](#api.EstimateMarginRequest) | [EstimateMarginResponse](#api.EstimateMarginResponse) | Get an estimate for the margin required for a new order |
| ERC20WithdrawalApproval | [ERC20WithdrawalApprovalRequest](#api.ERC20WithdrawalApprovalRequest) | [ERC20WithdrawalApprovalResponse](#api.ERC20WithdrawalApprovalResponse) | Get the bundle approval for an ERC20 withdrawal these data are being used to bundle the call to the smart contract on the ethereum bridge |
| Withdrawal | [WithdrawalRequest](#api.WithdrawalRequest) | [WithdrawalResponse](#api.WithdrawalResponse) | Get a withdrawal by its ID |
| Withdrawals | [WithdrawalsRequest](#api.WithdrawalsRequest) | [WithdrawalsResponse](#api.WithdrawalsResponse) | Get withdrawals for a party |
| Deposit | [DepositRequest](#api.DepositRequest) | [DepositResponse](#api.DepositResponse) | Get a deposit by its ID |
| Deposits | [DepositsRequest](#api.DepositsRequest) | [DepositsResponse](#api.DepositsResponse) | Get withdrawals for a party |
| NetworkParameters | [NetworkParametersRequest](#api.NetworkParametersRequest) | [NetworkParametersResponse](#api.NetworkParametersResponse) | Get the network parameters |
| LiquidityProvisions | [LiquidityProvisionsRequest](#api.LiquidityProvisionsRequest) | [LiquidityProvisionsResponse](#api.LiquidityProvisionsResponse) | Get the liquidity provision orders |





<a name="proto/assets.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/assets.proto



<a name="vega.Asset"></a>

### Asset
The Vega representation of an external asset.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | Internal identifier of the asset. |
| name | [string](#string) |  | Name of the asset (e.g: Great British Pound). |
| symbol | [string](#string) |  | Symbol of the asset (e.g: GBP). |
| totalSupply | [string](#string) |  | Total circulating supply for the asset. |
| decimals | [uint64](#uint64) |  | Number of decimals / precision handled by this asset. |
| source | [AssetSource](#vega.AssetSource) |  | The definition of the external source for this asset |






<a name="vega.AssetSource"></a>

### AssetSource
Asset source definition.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| builtinAsset | [BuiltinAsset](#vega.BuiltinAsset) |  | A built-in asset. |
| erc20 | [ERC20](#vega.ERC20) |  | An Ethereum ERC20 asset. |






<a name="vega.BuiltinAsset"></a>

### BuiltinAsset
A Vega internal asset.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the asset (e.g: Great British Pound). |
| symbol | [string](#string) |  | Symbol of the asset (e.g: GBP). |
| totalSupply | [string](#string) |  | Total circulating supply for the asset. |
| decimals | [uint64](#uint64) |  | Number of decimal / precision handled by this asset. |
| maxFaucetAmountMint | [string](#string) |  | Maximum amount that can be requested by a party through the built-in asset faucet at a time. |






<a name="vega.DevAssets"></a>

### DevAssets
Dev assets are for use in development networks only.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sources | [AssetSource](#vega.AssetSource) | repeated | Asset sources for development networks. |






<a name="vega.ERC20"></a>

### ERC20
An ERC20 token based asset, living on the ethereum network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contractAddress | [string](#string) |  | The address of the contract for the token, on the ethereum network |















<a name="proto/chain_events.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/chain_events.proto



<a name="vega.AddValidator"></a>

### AddValidator
A message to notify when a new validator is being added to the Vega network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [Identifier](#vega.Identifier) |  | The identifier of the validator. |






<a name="vega.BTCDeposit"></a>

### BTCDeposit
A Bitcoin deposit into Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The Vega network internal identifier of the asset. |
| sourceBTCAddress | [string](#string) |  | The BTC wallet initiating the deposit. |
| targetPartyId | [string](#string) |  | The Vega party identifier (pub-key) which is the target of the deposit. |






<a name="vega.BTCEvent"></a>

### BTCEvent
An event from the Bitcoin network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [uint64](#uint64) |  | The index of the transaction. |
| block | [uint64](#uint64) |  | The block in which the transaction happened. |
| deposit | [BTCDeposit](#vega.BTCDeposit) |  | Deposit BTC asset. |
| withdrawal | [BTCWithdrawal](#vega.BTCWithdrawal) |  | Withdraw BTC asset. |






<a name="vega.BTCWithdrawal"></a>

### BTCWithdrawal
A Bitcoin withdrawal from Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The vega network internal identifier of the asset. |
| sourcePartyId | [string](#string) |  | The party identifier (pub-key) initiating the withdrawal. |
| targetBTCAddress | [string](#string) |  | Target Bitcoin wallet address. |
| referenceNonce | [string](#string) |  | The nonce reference of the transaction. |






<a name="vega.BitcoinAddress"></a>

### BitcoinAddress
Used as a wrapper for a Bitcoin address (wallet).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) |  | A Bitcoin address. |






<a name="vega.BuiltinAssetDeposit"></a>

### BuiltinAssetDeposit
A deposit for a Vega built-in asset.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | A Vega network internal asset identifier. |
| partyID | [string](#string) |  | A Vega party identifier (pub-key) |
| amount | [uint64](#uint64) |  | The amount to be deposited. |






<a name="vega.BuiltinAssetEvent"></a>

### BuiltinAssetEvent
An event related to a Vega built-in asset.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| deposit | [BuiltinAssetDeposit](#vega.BuiltinAssetDeposit) |  | Built-in asset deposit. |
| withdrawal | [BuiltinAssetWithdrawal](#vega.BuiltinAssetWithdrawal) |  | Built-in asset withdrawal. |






<a name="vega.BuiltinAssetWithdrawal"></a>

### BuiltinAssetWithdrawal
A withdrawal for a Vega built-in asset.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | A Vega network internal asset identifier. |
| partyID | [string](#string) |  | A Vega network party identifier (pub-key) |
| amount | [uint64](#uint64) |  | The amount to be withdrawn. |






<a name="vega.ChainEvent"></a>

### ChainEvent
An event forwarded to the Vega network to provide information on events happening on other networks.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| txID | [string](#string) |  | The identifier of the transaction in which the events happened, usually a hash. |
| nonce | [uint64](#uint64) |  | Arbitrary one-time integer used to prevent replay attacks. |
| builtin | [BuiltinAssetEvent](#vega.BuiltinAssetEvent) |  | Built-in asset event. |
| erc20 | [ERC20Event](#vega.ERC20Event) |  | Ethereum ERC20 event. |
| btc | [BTCEvent](#vega.BTCEvent) |  | Bitcoin BTC event. |
| validator | [ValidatorEvent](#vega.ValidatorEvent) |  | Validator event. |






<a name="vega.ERC20AssetDelist"></a>

### ERC20AssetDelist
An asset denylisting for an ERC20 token.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The Vega network internal identifier of the asset. |






<a name="vega.ERC20AssetList"></a>

### ERC20AssetList
An asset allowlisting for an ERC20 token.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The Vega network internal identifier of the asset. |






<a name="vega.ERC20Deposit"></a>

### ERC20Deposit
An asset deposit for an ERC20 token.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The vega network internal identifier of the asset. |
| sourceEthereumAddress | [string](#string) |  | The Ethereum wallet that initiated the deposit. |
| targetPartyID | [string](#string) |  | The Vega party identifier (pub-key) which is the target of the deposit. |






<a name="vega.ERC20Event"></a>

### ERC20Event
An event related to an ERC20 token.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [uint64](#uint64) |  | Index of the transaction |
| block | [uint64](#uint64) |  | The block in which the transaction was added |
| assetList | [ERC20AssetList](#vega.ERC20AssetList) |  | List an ERC20 asset. |
| assetDelist | [ERC20AssetDelist](#vega.ERC20AssetDelist) |  | De-list an ERC20 asset. |
| deposit | [ERC20Deposit](#vega.ERC20Deposit) |  | Deposit ERC20 asset. |
| withdrawal | [ERC20Withdrawal](#vega.ERC20Withdrawal) |  | Withdraw ERC20 asset. |






<a name="vega.ERC20Withdrawal"></a>

### ERC20Withdrawal
An asset withdrawal for an ERC20 token.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The Vega network internal identifier of the asset. |
| targetEthereumAddress | [string](#string) |  | The target Ethereum wallet address. |
| referenceNonce | [string](#string) |  | The reference nonce used for the transaction. |






<a name="vega.EthereumAddress"></a>

### EthereumAddress
Used as a wrapper for an Ethereum address (wallet/contract).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) |  | An Ethereum address. |






<a name="vega.Identifier"></a>

### Identifier
Used as a wrapper type on any possible network address supported by Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ethereumAddress | [EthereumAddress](#vega.EthereumAddress) |  | Ethereum network. |
| bitcoinAddress | [BitcoinAddress](#vega.BitcoinAddress) |  | Bitcoin network. |






<a name="vega.RemoveValidator"></a>

### RemoveValidator
A message to notify when a validator is being removed from the Vega network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [Identifier](#vega.Identifier) |  | The identifier of the validator. |






<a name="vega.ValidatorEvent"></a>

### ValidatorEvent
An event related to validator management with foreign networks.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sourceID | [string](#string) |  | The source identifier of the event. |
| add | [AddValidator](#vega.AddValidator) |  | Add a new validator. |
| rm | [RemoveValidator](#vega.RemoveValidator) |  | Remove an existing validator. |















<a name="proto/events.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/events.proto



<a name="vega.AuctionEvent"></a>

### AuctionEvent
An auction event indicating a change in auction state, for example starting or ending an auction


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier for the event |
| openingAuction | [bool](#bool) |  | True if the event indicates an auction opening and False otherwise |
| leave | [bool](#bool) |  | True if the event indicates leaving auction mode and False otherwise |
| start | [int64](#int64) |  | Timestamp containing the start time for an auction |
| end | [int64](#int64) |  | Timestamp containing the end time for an auction |
| trigger | [AuctionTrigger](#vega.AuctionTrigger) |  | the reason this market is/was in auction |






<a name="vega.BusEvent"></a>

### BusEvent
A bus event is a container for event bus events emitted by Vega


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | A unique event identifier for the message |
| block | [string](#string) |  | The batch (or block) of transactions that the events relate to |
| type | [BusEventType](#vega.BusEventType) |  | The type of bus event (one of the list below) |
| timeUpdate | [TimeUpdate](#vega.TimeUpdate) |  | Time update events, see [TimeUpdate](#vega.TimeUpdate) |
| transferResponses | [TransferResponses](#vega.TransferResponses) |  | Transfer responses update events, see [TransferResponses](#vega.TransferResponses) |
| positionResolution | [PositionResolution](#vega.PositionResolution) |  | Position resolution events, see [PositionResolution](#vega.PositionResolution) |
| order | [Order](#vega.Order) |  | Order events |
| account | [Account](#vega.Account) |  | Account events |
| party | [Party](#vega.Party) |  | Party events |
| trade | [Trade](#vega.Trade) |  | Trade events |
| marginLevels | [MarginLevels](#vega.MarginLevels) |  | Margin level update events |
| proposal | [Proposal](#vega.Proposal) |  | Proposal events (for governance) |
| vote | [Vote](#vega.Vote) |  | Vote events (for governance) |
| marketData | [MarketData](#vega.MarketData) |  | Market data events |
| nodeSignature | [NodeSignature](#vega.NodeSignature) |  | Node signature events |
| lossSocialization | [LossSocialization](#vega.LossSocialization) |  | Loss socialization events, see [LossSocialization](#vega.LossSocialization) |
| settlePosition | [SettlePosition](#vega.SettlePosition) |  | Position settlement events, see [SettlePosition](#vega.SettlePosition) |
| settleDistressed | [SettleDistressed](#vega.SettleDistressed) |  | Position distressed events, see [SettleDistressed](#vega.SettleDistressed) |
| marketCreated | [Market](#vega.Market) |  | Market created events |
| asset | [Asset](#vega.Asset) |  | Asset events |
| marketTick | [MarketTick](#vega.MarketTick) |  | Market tick events, see [MarketTick](#vega.MarketTick) |
| withdrawal | [Withdrawal](#vega.Withdrawal) |  | Withdrawal events |
| deposit | [Deposit](#vega.Deposit) |  | Deposit events |
| auction | [AuctionEvent](#vega.AuctionEvent) |  | Auction events, see [AuctionEvent](#vega.AuctionEvent) |
| riskFactor | [RiskFactor](#vega.RiskFactor) |  | Risk factor events |
| networkParameter | [NetworkParameter](#vega.NetworkParameter) |  | Network parameter events |
| liquidityProvision | [LiquidityProvision](#vega.LiquidityProvision) |  | LiquidityProvision events |
| market | [MarketEvent](#vega.MarketEvent) |  | Market tick events, see [MarketEvent](#vega.MarketEvent) |
| txErrEvent | [TxErrorEvent](#vega.TxErrorEvent) |  | Transaction error events - separate category, not included in ALL events |






<a name="vega.LossSocialization"></a>

### LossSocialization
A loss socialization event contains details on the amount of wins unable to be distributed


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier for the event |
| partyID | [string](#string) |  | Party identifier (public key) for the event |
| amount | [int64](#int64) |  | Amount distributed |






<a name="vega.MarketEvent"></a>

### MarketEvent
MarketEvent - the common denominator for all market events
interface has a method to return a string for logging


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier for the event |
| payload | [string](#string) |  | Payload is a unique information string. |






<a name="vega.MarketTick"></a>

### MarketTick
A market ticket event contains the time value for when a particular market was last processed on Vega


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | Market identifier for the event |
| time | [int64](#int64) |  | Timestamp containing latest update from Vega blockchain aka Vega-time |






<a name="vega.PositionResolution"></a>

### PositionResolution
A position resolution event contains information on distressed trades


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier for the event |
| distressed | [int64](#int64) |  | Number of distressed traders |
| closed | [int64](#int64) |  | Number of close outs |
| markPrice | [uint64](#uint64) |  | Mark price, as an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places |






<a name="vega.SettleDistressed"></a>

### SettleDistressed
A settle distressed event contains information on distressed trading parties who are closed out

Note: Any PositionResolution event (market level) will most likely be followed by a number of these events


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier for the event |
| partyID | [string](#string) |  | Party identifier (public key) for the event |
| margin | [uint64](#uint64) |  | Margin value as an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places |
| price | [uint64](#uint64) |  | Price as an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places |






<a name="vega.SettlePosition"></a>

### SettlePosition
A settle position event contains position settlement information for a party


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier for the event |
| partyID | [string](#string) |  | Party identifier (public key) for the event |
| price | [uint64](#uint64) |  | Price of settlement as an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places |
| tradeSettlements | [TradeSettlement](#vega.TradeSettlement) | repeated | A collection of 1 or more trade settlements |






<a name="vega.TimeUpdate"></a>

### TimeUpdate
A time update event contains the latest time update from Vega blockchain


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [int64](#int64) |  | Timestamp containing latest update from Vega blockchain aka Vega-time |






<a name="vega.TradeSettlement"></a>

### TradeSettlement
A trade settlement is part of the settle position event


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| size | [int64](#int64) |  | Size of trade settlement |
| price | [uint64](#uint64) |  | Price of settlement as an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places |






<a name="vega.TransferResponses"></a>

### TransferResponses
A transfer responses event contains a collection of transfer information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| responses | [TransferResponse](#vega.TransferResponse) | repeated | 1 or more entries containing internal transfer information |






<a name="vega.TxErrorEvent"></a>

### TxErrorEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| PartyID | [string](#string) |  | the party who had a tx fail |
| errMsg | [string](#string) |  | error message describing what went wrong |
| orderSubmission | [OrderSubmission](#vega.OrderSubmission) |  |  |
| orderAmendment | [OrderAmendment](#vega.OrderAmendment) |  |  |
| orderCancellation | [OrderCancellation](#vega.OrderCancellation) |  |  |
| proposal | [Proposal](#vega.Proposal) |  |  |
| vote | [Vote](#vega.Vote) |  |  |








<a name="vega.BusEventType"></a>

### BusEventType
An (event) bus event type is used to specify a type of event
It has 2 styles of event:
Single values (e.g. BUS_EVENT_TYPE_ORDER) where they represent one data item
Group values (e.g. BUS_EVENT_TYPE_AUCTION) where they represent a group of data items

| Name | Number | Description |
| ---- | ------ | ----------- |
| BUS_EVENT_TYPE_UNSPECIFIED | 0 | Default value, always invalid |
| BUS_EVENT_TYPE_ALL | 1 | Events of ALL event types, used when filtering stream from event bus |
| BUS_EVENT_TYPE_TIME_UPDATE | 2 | Event for blockchain time updates |
| BUS_EVENT_TYPE_TRANSFER_RESPONSES | 3 | Event for when a transfer happens internally, contains the transfer information |
| BUS_EVENT_TYPE_POSITION_RESOLUTION | 4 | Event indicating position resolution has occurred |
| BUS_EVENT_TYPE_ORDER | 5 | Event for order updates, both new and existing orders |
| BUS_EVENT_TYPE_ACCOUNT | 6 | Event for account updates |
| BUS_EVENT_TYPE_PARTY | 7 | Event for party updates |
| BUS_EVENT_TYPE_TRADE | 8 | Event indicating a new trade has occurred |
| BUS_EVENT_TYPE_MARGIN_LEVELS | 9 | Event indicating margin levels have changed for a party |
| BUS_EVENT_TYPE_PROPOSAL | 10 | Event for proposal updates (for governance) |
| BUS_EVENT_TYPE_VOTE | 11 | Event indicating a new vote has occurred (for governance) |
| BUS_EVENT_TYPE_MARKET_DATA | 12 | Event for market data updates |
| BUS_EVENT_TYPE_NODE_SIGNATURE | 13 | Event for a new signature for a Vega node |
| BUS_EVENT_TYPE_LOSS_SOCIALIZATION | 14 | Event indicating loss socialisation occurred for a party |
| BUS_EVENT_TYPE_SETTLE_POSITION | 15 | Event for when a position is being settled |
| BUS_EVENT_TYPE_SETTLE_DISTRESSED | 16 | Event for when a position is distressed |
| BUS_EVENT_TYPE_MARKET_CREATED | 17 | Event indicating a new market was created |
| BUS_EVENT_TYPE_ASSET | 18 | Event for when an asset is added to Vega |
| BUS_EVENT_TYPE_MARKET_TICK | 19 | Event indicating a market tick event |
| BUS_EVENT_TYPE_WITHDRAWAL | 20 | Event for when a withdrawal occurs |
| BUS_EVENT_TYPE_DEPOSIT | 21 | Event for when a deposit occurs |
| BUS_EVENT_TYPE_AUCTION | 22 | Event indicating a change in auction state, for example starting or ending an auction |
| BUS_EVENT_TYPE_RISK_FACTOR | 23 | Event indicating a risk factor has been updated |
| BUS_EVENT_TYPE_NETWORK_PARAMETER | 24 | Event indicating a network parameter has been added or updated |
| BUS_EVENT_TYPE_LIQUIDITY_PROVISION | 25 | Event indicating a liquidity provision has been created or updated |
| BUS_EVENT_TYPE_MARKET | 101 | Event indicating a market related event, for example when a market opens |
| BUS_EVENT_TYPE_TX_ERROR | 201 | Event used to report failed transactions back to a user - excluded from the ALL type |










<a name="proto/governance.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/governance.proto



<a name="vega.FutureProduct"></a>

### FutureProduct
Future product configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maturity | [string](#string) |  | Future product maturity (ISO8601/RFC3339 timestamp). |
| asset | [string](#string) |  | Product asset name. |






<a name="vega.GovernanceData"></a>

### GovernanceData
Governance data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| proposal | [Proposal](#vega.Proposal) |  | Proposal. |
| yes | [Vote](#vega.Vote) | repeated | All &#34;yes&#34; votes in favour of the proposal above. |
| no | [Vote](#vega.Vote) | repeated | All &#34;no&#34; votes against the proposal above. |
| yesParty | [GovernanceData.YesPartyEntry](#vega.GovernanceData.YesPartyEntry) | repeated | All latest YES votes by party (guaranteed to be unique). key (string) is the party ID (public key) value (Vote) is the vote cast by the given party |
| noParty | [GovernanceData.NoPartyEntry](#vega.GovernanceData.NoPartyEntry) | repeated | All latest NO votes by party (guaranteed to be unique). key (string) is the party ID (public key) value (Vote) is the vote cast by the given party |






<a name="vega.GovernanceData.NoPartyEntry"></a>

### GovernanceData.NoPartyEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [Vote](#vega.Vote) |  |  |






<a name="vega.GovernanceData.YesPartyEntry"></a>

### GovernanceData.YesPartyEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [Vote](#vega.Vote) |  |  |






<a name="vega.InstrumentConfiguration"></a>

### InstrumentConfiguration
Instrument configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Instrument name. |
| code | [string](#string) |  | Instrument code. |
| quoteName | [string](#string) |  | Quote (secondary) security. |
| future | [FutureProduct](#vega.FutureProduct) |  | Futures. |






<a name="vega.NewAsset"></a>

### NewAsset
New asset on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changes | [AssetSource](#vega.AssetSource) |  | Asset configuration. |






<a name="vega.NewMarket"></a>

### NewMarket
New market on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changes | [NewMarketConfiguration](#vega.NewMarketConfiguration) |  | Configuration. |






<a name="vega.NewMarketConfiguration"></a>

### NewMarketConfiguration
Configuration for a new market on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instrument | [InstrumentConfiguration](#vega.InstrumentConfiguration) |  | New market instrument configuration. |
| decimalPlaces | [uint64](#uint64) |  | Decimal places used for the new market. |
| metadata | [string](#string) | repeated | Optional new market meta data, tags. |
| openingAuctionDuration | [int64](#int64) |  | Time duration for the opening auction to last. |
| priceMonitoringParameters | [PriceMonitoringParameters](#vega.PriceMonitoringParameters) |  | price monitoring configuration |
| simple | [SimpleModelParams](#vega.SimpleModelParams) |  | Simple risk model parameters, valid only if MODEL_SIMPLE is selected |
| logNormal | [LogNormalRiskModel](#vega.LogNormalRiskModel) |  | Log normal risk model parameters, valid only if MODEL_LOG_NORMAL is selected |
| continuous | [ContinuousTrading](#vega.ContinuousTrading) |  | Continuous trading. |
| discrete | [DiscreteTrading](#vega.DiscreteTrading) |  | Discrete trading. |






<a name="vega.Proposal"></a>

### Proposal
Governance proposal.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | Unique proposal identifier. |
| reference | [string](#string) |  | Proposal reference. |
| partyID | [string](#string) |  | Party identifier of the author (the party submitting the proposal). |
| state | [Proposal.State](#vega.Proposal.State) |  | Proposal state (see (Proposal.State)[#vega.Proposal.State] definition). |
| timestamp | [int64](#int64) |  | Proposal timestamp for date and time (in nanoseconds) when proposal was submitted to the network. |
| terms | [ProposalTerms](#vega.ProposalTerms) |  | Proposal configuration and the actual change that is meant to be executed when proposal is enacted. |
| reason | [ProposalError](#vega.ProposalError) |  | A reason for the current state of the proposal, this may be set in case of REJECTED and FAILED statuses. |






<a name="vega.ProposalTerms"></a>

### ProposalTerms
Terms for a governance proposal on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| closingTimestamp | [int64](#int64) |  | Timestamp (Unix time in seconds) when voting closes for this proposal. Constrained by `minCloseInSeconds` and `maxCloseInSeconds` network parameters. |
| enactmentTimestamp | [int64](#int64) |  | Timestamp (Unix time in seconds) when proposal gets enacted (if passed). Constrained by `minEnactInSeconds` and `maxEnactInSeconds` network parameters. |
| validationTimestamp | [int64](#int64) |  | Validation timestamp (Unix time in seconds). |
| updateMarket | [UpdateMarket](#vega.UpdateMarket) |  | Proposal change for modifying an existing market on Vega. |
| newMarket | [NewMarket](#vega.NewMarket) |  | Proposal change for creating new market on Vega. |
| updateNetworkParameter | [UpdateNetworkParameter](#vega.UpdateNetworkParameter) |  | Proposal change for updating Vega network parameters. |
| newAsset | [NewAsset](#vega.NewAsset) |  | Proposal change for creating new assets on Vega. |






<a name="vega.UpdateMarket"></a>

### UpdateMarket
Update an existing market on Vega.






<a name="vega.UpdateNetworkParameter"></a>

### UpdateNetworkParameter
Update network configuration on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changes | [NetworkParameter](#vega.NetworkParameter) |  | The network parameter to update |






<a name="vega.Vote"></a>

### Vote
Governance vote.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Voter&#39;s party identifier. |
| value | [Vote.Value](#vega.Vote.Value) |  | Actual vote. |
| proposalID | [string](#string) |  | Identifier of the proposal being voted on. |
| timestamp | [int64](#int64) |  | Vote timestamp for date and time (in nanoseconds) when vote was submitted to the network. |








<a name="vega.Proposal.State"></a>

### Proposal.State
Proposal state transition:
Open -&gt;
  - Passed -&gt; Enacted.
  - Passed -&gt; Failed.
  - Declined
Rejected
Proposal can enter Failed state from any other state.

| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 | Default value, always invalid. |
| STATE_FAILED | 1 | Proposal enactment has failed - even though proposal has passed, its execution could not be performed. |
| STATE_OPEN | 2 | Proposal is open for voting. |
| STATE_PASSED | 3 | Proposal has gained enough support to be executed. |
| STATE_REJECTED | 4 | Proposal wasn&#39;t accepted (proposal terms failed validation due to wrong configuration or failing to meet network requirements). |
| STATE_DECLINED | 5 | Proposal didn&#39;t get enough votes (either failing to gain required participation or majority level). |
| STATE_ENACTED | 6 | Proposal enacted. |
| STATE_WAITING_FOR_NODE_VOTE | 7 | Waiting for node validation of the proposal |



<a name="vega.ProposalError"></a>

### ProposalError
A list of possible errors that can cause a proposal to be in state rejected or failed.

| Name | Number | Description |
| ---- | ------ | ----------- |
| PROPOSAL_ERROR_UNSPECIFIED | 0 | Default value. |
| PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON | 1 | The specified close time is too early base on network parameters. |
| PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE | 2 | The specified close time is too late based on network parameters. |
| PROPOSAL_ERROR_ENACT_TIME_TOO_SOON | 3 | The specified enact time is too early based on network parameters. |
| PROPOSAL_ERROR_ENACT_TIME_TOO_LATE | 4 | The specified enact time is too late based on network parameters. |
| PROPOSAL_ERROR_INSUFFICIENT_TOKENS | 5 | The proposer for this proposal as insufficient tokens. |
| PROPOSAL_ERROR_INVALID_INSTRUMENT_SECURITY | 6 | The instrument quote name and base name were the same. |
| PROPOSAL_ERROR_NO_PRODUCT | 7 | The proposal has no product. |
| PROPOSAL_ERROR_UNSUPPORTED_PRODUCT | 8 | The specified product is not supported. |
| PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT_TIMESTAMP | 9 | Invalid future maturity timestamp (expect RFC3339). |
| PROPOSAL_ERROR_PRODUCT_MATURITY_IS_PASSED | 10 | The product maturity is past. |
| PROPOSAL_ERROR_NO_TRADING_MODE | 11 | The proposal has no trading mode. |
| PROPOSAL_ERROR_UNSUPPORTED_TRADING_MODE | 12 | The proposal has an unsupported trading mode. |
| PROPOSAL_ERROR_NODE_VALIDATION_FAILED | 13 | The proposal failed node validation. |
| PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD | 14 | A field is missing in a builtin asset source. |
| PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS | 15 | The contract address is missing in the ERC20 asset source. |
| PROPOSAL_ERROR_INVALID_ASSET | 16 | The asset id refer to no assets in vega. |
| PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS | 17 | Proposal terms timestamps are not compatible (Validation &lt; Closing &lt; Enactment). |
| PROPOSAL_ERROR_NO_RISK_PARAMETERS | 18 | No risk parameteres were specified |
| PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY | 19 | Invalid key in update network parameter proposal |
| PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE | 20 | Invalid valid in update network parameter proposal |
| PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED | 21 | Validation failed for network parameter proposal |
| PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_SMALL | 22 | Opening auction duration is less than the network minimum opening auction time |
| PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_LARGE | 23 | Opening auction duration is more than the network minimum opening auction time |



<a name="vega.Vote.Value"></a>

### Vote.Value
Vote value.

| Name | Number | Description |
| ---- | ------ | ----------- |
| VALUE_UNSPECIFIED | 0 | Default value, always invalid. |
| VALUE_NO | 1 | A vote against the proposal. |
| VALUE_YES | 2 | A vote in favour of the proposal. |










<a name="proto/markets.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/markets.proto



<a name="vega.AuctionDuration"></a>

### AuctionDuration
An auction duration is used to configure 3 auction periods:
1. `duration &gt; 0`, `volume == 0`:
  The auction will last for at least N seconds.
2. `duration == 0`, `volume &gt; 0`:
  The auction will end once we can close with given traded volume.
3. `duration &gt; 0`, `volume &gt; 0`:
  The auction will take at least N seconds, but can end sooner if we can trade a certain volume.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| duration | [int64](#int64) |  | Duration of the auction in seconds. |
| volume | [uint64](#uint64) |  | Target uncrossing trading volume. |






<a name="vega.ContinuousTrading"></a>

### ContinuousTrading
Continuous trading.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tickSize | [string](#string) |  | Tick size. |






<a name="vega.DiscreteTrading"></a>

### DiscreteTrading
Discrete trading.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| durationNs | [int64](#int64) |  | Duration in nanoseconds, maximum 1 month (2592000000000000 ns) |
| tickSize | [string](#string) |  | Tick size. |






<a name="vega.EthereumEvent"></a>

### EthereumEvent
Ethereum event (for oracles).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contractID | [string](#string) |  | Ethereum contract identifier. |
| event | [string](#string) |  | Event. |
| value | [uint64](#uint64) |  | Value. |






<a name="vega.FeeFactors"></a>

### FeeFactors
Fee factors definition.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| makerFee | [string](#string) |  | Maker fee. |
| infrastructureFee | [string](#string) |  | Infrastructure fee. |
| liquidityFee | [string](#string) |  | Liquidity fee. |






<a name="vega.Fees"></a>

### Fees
Fees definition.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| factors | [FeeFactors](#vega.FeeFactors) |  | Fee factors. |






<a name="vega.Future"></a>

### Future
Future product definition.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maturity | [string](#string) |  | The maturity for the future. |
| asset | [string](#string) |  | The asset for the future. |
| ethereumEvent | [EthereumEvent](#vega.EthereumEvent) |  | Ethereum events. |






<a name="vega.Instrument"></a>

### Instrument
Instrument definition.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Instrument identifier. |
| code | [string](#string) |  | Code for the instrument. |
| name | [string](#string) |  | Name of the instrument. |
| quoteName | [string](#string) |  | Quote name of the instrument. |
| metadata | [InstrumentMetadata](#vega.InstrumentMetadata) |  | A collection of instrument meta-data. |
| initialMarkPrice | [uint64](#uint64) |  | An initial mark price for the instrument. |
| future | [Future](#vega.Future) |  | Future. |






<a name="vega.InstrumentMetadata"></a>

### InstrumentMetadata
Instrument metadata definition.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tags | [string](#string) | repeated | A list of 0 or more tags. |






<a name="vega.LogNormalModelParams"></a>

### LogNormalModelParams
Risk model parameters for log normal.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| mu | [double](#double) |  | Mu param. |
| r | [double](#double) |  | R param. |
| sigma | [double](#double) |  | Sigma param. |






<a name="vega.LogNormalRiskModel"></a>

### LogNormalRiskModel
Risk model for log normal.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| riskAversionParameter | [double](#double) |  | Risk Aversion Parameter. |
| tau | [double](#double) |  | Tau. |
| params | [LogNormalModelParams](#vega.LogNormalModelParams) |  | Risk model parameters for log normal. |






<a name="vega.MarginCalculator"></a>

### MarginCalculator
Margin Calculator definition.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| scalingFactors | [ScalingFactors](#vega.ScalingFactors) |  | Scaling factors for margin calculation. |






<a name="vega.Market"></a>

### Market
Market definition.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Unique identifier. |
| tradableInstrument | [TradableInstrument](#vega.TradableInstrument) |  | Tradable instrument configuration. |
| decimalPlaces | [uint64](#uint64) |  | Number of decimal places that a price must be shifted by in order to get a correct price denominated in the currency of the market. For example: `realPrice = price / 10^decimalPlaces` |
| fees | [Fees](#vega.Fees) |  | Fees configuration. |
| openingAuction | [AuctionDuration](#vega.AuctionDuration) |  | Auction duration specifies how long the opening auction will run (minimum duration and optionally a minimum traded volume). |
| continuous | [ContinuousTrading](#vega.ContinuousTrading) |  | Continuous. |
| discrete | [DiscreteTrading](#vega.DiscreteTrading) |  | Discrete. |
| priceMonitoringSettings | [PriceMonitoringSettings](#vega.PriceMonitoringSettings) |  | PriceMonitoringSettings for the market. |
| targetStakeParameters | [TargetStakeParameters](#vega.TargetStakeParameters) |  | TargetStakeParameters for the market |






<a name="vega.PriceMonitoringParameters"></a>

### PriceMonitoringParameters
PriceMonitoringParameters contain a collection of triggers to be used for a given market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| triggers | [PriceMonitoringTrigger](#vega.PriceMonitoringTrigger) | repeated |  |






<a name="vega.PriceMonitoringSettings"></a>

### PriceMonitoringSettings



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parameters | [PriceMonitoringParameters](#vega.PriceMonitoringParameters) |  | Specifies PriceMonitoringParameters to be used for price monitoring purposes |
| updateFrequency | [int64](#int64) |  | Specifies how often (expressed in seconds) the price monitoring bounds should be updated. |






<a name="vega.PriceMonitoringTrigger"></a>

### PriceMonitoringTrigger
PriceMonitoringTrigger holds together price projection horizon τ, probability level p, and auction extension duration


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| horizon | [int64](#int64) |  | Price monitoring projection horizon τ in seconds. |
| probability | [double](#double) |  | Price monitoirng probability level p. |
| auctionExtension | [int64](#int64) |  | Price monitoring auction extension duration in seconds should the price breach it&#39;s theoretical level over the specified horizon at the specified probability level. |






<a name="vega.ScalingFactors"></a>

### ScalingFactors
Scaling Factors (for use in margin calculation).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| searchLevel | [double](#double) |  | Search level. |
| initialMargin | [double](#double) |  | Initial margin level. |
| collateralRelease | [double](#double) |  | Collateral release level. |






<a name="vega.SimpleModelParams"></a>

### SimpleModelParams
Risk model parameters for simple modelling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| factorLong | [double](#double) |  | Pre-defined risk factor value for long. |
| factorShort | [double](#double) |  | Pre-defined risk factor value for short. |
| maxMoveUp | [double](#double) |  | Pre-defined maximum price move up that the model considers as valid. |
| minMoveDown | [double](#double) |  | Pre-defined minimum price move down that the model considers as valid. |
| probabilityOfTrading | [double](#double) |  | Pre-defined constant probability of trading |






<a name="vega.SimpleRiskModel"></a>

### SimpleRiskModel
Risk model for simple modelling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | [SimpleModelParams](#vega.SimpleModelParams) |  | Risk model params for simple modelling. |






<a name="vega.TargetStakeParameters"></a>

### TargetStakeParameters
TargetStakeParameters contains parameters used in target stake calculation


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timeWindow | [int64](#int64) |  | Specifies length of time window expressed in seconds for target stake calculation |
| scalingFactor | [double](#double) |  | Specifies scaling factors used in target stake calculation |






<a name="vega.TradableInstrument"></a>

### TradableInstrument
Tradable Instrument definition.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instrument | [Instrument](#vega.Instrument) |  | Instrument details. |
| marginCalculator | [MarginCalculator](#vega.MarginCalculator) |  | Margin calculator for the instrument. |
| logNormalRiskModel | [LogNormalRiskModel](#vega.LogNormalRiskModel) |  | Log normal. |
| simpleRiskModel | [SimpleRiskModel](#vega.SimpleRiskModel) |  | Simple. |















<a name="proto/tm/replay.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/tm/replay.proto



<a name="vega.BlockParams"></a>

### BlockParams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maxBytes | [int64](#int64) |  |  |
| maxGas | [int64](#int64) |  |  |






<a name="vega.ConsensusParams"></a>

### ConsensusParams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| block | [BlockParams](#vega.BlockParams) |  |  |
| evidence | [EvidenceParams](#vega.EvidenceParams) |  |  |
| validator | [ValidatorParams](#vega.ValidatorParams) |  |  |
| version | [VersionParams](#vega.VersionParams) |  |  |






<a name="vega.Event"></a>

### Event



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |
| attributes | [EventAttribute](#vega.EventAttribute) | repeated |  |






<a name="vega.EventAttribute"></a>

### EventAttribute



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [bytes](#bytes) |  |  |
| value | [bytes](#bytes) |  |  |
| index | [bool](#bool) |  | nondeterministic |






<a name="vega.Evidence"></a>

### Evidence



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [EvidenceType](#vega.EvidenceType) |  |  |
| validator | [Validator](#vega.Validator) |  |  |
| height | [int64](#int64) |  |  |
| time | [int64](#int64) |  |  |
| totalVotingPower | [int64](#int64) |  |  |






<a name="vega.EvidenceParams"></a>

### EvidenceParams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maxAgeNumBlocks | [int64](#int64) |  |  |
| maxAgeDuration | [int64](#int64) |  |  |
| max_num | [uint32](#uint32) |  |  |






<a name="vega.Header"></a>

### Header



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| chainId | [string](#string) |  |  |
| height | [int64](#int64) |  |  |
| time | [int64](#int64) |  |  |






<a name="vega.LastCommitInfo"></a>

### LastCommitInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| round | [int32](#int32) |  |  |
| votes | [VoteInfo](#vega.VoteInfo) | repeated |  |






<a name="vega.PublicKey"></a>

### PublicKey



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ed25519 | [bytes](#bytes) |  |  |






<a name="vega.RequestBeginBlock"></a>

### RequestBeginBlock



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hash | [bytes](#bytes) |  |  |
| header | [Header](#vega.Header) |  |  |
| lastCommitInfo | [LastCommitInfo](#vega.LastCommitInfo) |  |  |
| byzantineValidators | [Evidence](#vega.Evidence) | repeated |  |






<a name="vega.RequestDeliverTx"></a>

### RequestDeliverTx



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tx | [bytes](#bytes) |  |  |






<a name="vega.RequestInitChain"></a>

### RequestInitChain



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| time | [int64](#int64) |  |  |
| chainID | [string](#string) |  |  |
| consensusParams | [ConsensusParams](#vega.ConsensusParams) |  |  |
| validators | [ValidatorUpdate](#vega.ValidatorUpdate) | repeated |  |
| appStateBytes | [bytes](#bytes) |  |  |
| initialHeight | [int64](#int64) |  |  |






<a name="vega.ResponseBeginBlock"></a>

### ResponseBeginBlock



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| events | [Event](#vega.Event) | repeated |  |






<a name="vega.ResponseDeliverTx"></a>

### ResponseDeliverTx



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [uint32](#uint32) |  |  |
| data | [bytes](#bytes) |  |  |
| log | [string](#string) |  | nondeterministic |
| info | [string](#string) |  | nondeterministic |
| gasWanted | [int64](#int64) |  |  |
| gasUsed | [int64](#int64) |  |  |
| events | [Event](#vega.Event) | repeated |  |
| codespace | [string](#string) |  |  |






<a name="vega.ResponseInitChain"></a>

### ResponseInitChain



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| consensusParams | [ConsensusParams](#vega.ConsensusParams) |  |  |
| validators | [ValidatorUpdate](#vega.ValidatorUpdate) | repeated |  |
| appHash | [bytes](#bytes) |  |  |






<a name="vega.TmEvent"></a>

### TmEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reqInitChain | [RequestInitChain](#vega.RequestInitChain) |  |  |
| resInitChain | [ResponseInitChain](#vega.ResponseInitChain) |  |  |
| reqDeliverTx | [RequestDeliverTx](#vega.RequestDeliverTx) |  |  |
| resDeliverTx | [ResponseDeliverTx](#vega.ResponseDeliverTx) |  |  |
| reqBeginBlock | [RequestBeginBlock](#vega.RequestBeginBlock) |  |  |
| resBeginBlock | [ResponseBeginBlock](#vega.ResponseBeginBlock) |  |  |






<a name="vega.Validator"></a>

### Validator



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [bytes](#bytes) |  |  |
| power | [int64](#int64) |  |  |






<a name="vega.ValidatorParams"></a>

### ValidatorParams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pubKeyTypes | [string](#string) | repeated |  |






<a name="vega.ValidatorUpdate"></a>

### ValidatorUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pub_key | [PublicKey](#vega.PublicKey) |  |  |
| power | [int64](#int64) |  |  |






<a name="vega.VersionParams"></a>

### VersionParams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| appVersion | [uint64](#uint64) |  |  |






<a name="vega.VoteInfo"></a>

### VoteInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| validator | [Validator](#vega.Validator) |  |  |
| signedLastBlock | [bool](#bool) |  |  |








<a name="vega.EvidenceType"></a>

### EvidenceType


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 |  |
| DUPLICATE_VOTE | 1 |  |
| LIGHT_CLIENT_ATTACK | 2 |  |










<a name="proto/vega.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/vega.proto



<a name="vega.Account"></a>

### Account
Represents an account for an asset on Vega for a particular owner or party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Unique account identifier (used internally by Vega). |
| owner | [string](#string) |  | The party that the account belongs to. Special values include `network`, which represents the Vega network and is most commonly seen during liquidation of distressed trading positions. |
| balance | [uint64](#uint64) |  | Balance of the asset, the balance is an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. Balances cannot be negative. |
| asset | [string](#string) |  | Asset identifier for the account. |
| marketID | [string](#string) |  | Market identifier for the account. If [`AccountType`](#vega.AccountType).`ACCOUNT_TYPE_GENERAL` this will be empty. |
| type | [AccountType](#vega.AccountType) |  | The account type related to this account. |






<a name="vega.AuctionIndicativeState"></a>

### AuctionIndicativeState
AuctionIndicativeState is used to emit an event with the indicative price/volume per market during an auction.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | The market identifier for which this state relates to. |
| indicativePrice | [uint64](#uint64) |  | The Indicative Uncrossing Price is the price at which all trades would occur if we uncrossed the auction now. |
| indicativeVolume | [uint64](#uint64) |  | The Indicative Uncrossing Volume is the volume available at the Indicative crossing price if we uncrossed the auction now. |
| auctionStart | [int64](#int64) |  | The timestamp at which the auction started. |
| auctionEnd | [int64](#int64) |  | The timestamp at which the auction is meant to stop. |






<a name="vega.Candle"></a>

### Candle
Represents the high, low, open, and closing prices for an interval of trading,
referred to commonly as a candlestick or candle.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [int64](#int64) |  | Timestamp for the point in time when the candle was initially created/opened, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| datetime | [string](#string) |  | An ISO-8601 datetime with nanosecond precision for when the candle was last updated. |
| high | [uint64](#uint64) |  | Highest price for trading during the candle interval. |
| low | [uint64](#uint64) |  | Lowest price for trading during the candle interval. |
| open | [uint64](#uint64) |  | Open trade price. |
| close | [uint64](#uint64) |  | Closing trade price. |
| volume | [uint64](#uint64) |  | Total trading volume during the candle interval. |
| interval | [Interval](#vega.Interval) |  | Time interval for the candle. See [`Interval`](#vega.Interval). |






<a name="vega.Deposit"></a>

### Deposit
The representation of a deposit in the vega network


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | The ID of the deposit |
| status | [Deposit.Status](#vega.Deposit.Status) |  | Status of the deposit |
| partyID | [string](#string) |  | The party initiating the deposit |
| asset | [string](#string) |  | The vega asset targeted by this deposit |
| amount | [string](#string) |  | The amount to be deposited |
| txHash | [string](#string) |  | The hash of the transaction from the foreign chain |
| creditedTimestamp | [int64](#int64) |  | The time where the vega account got updated with the deposit |
| createdTimestamp | [int64](#int64) |  | The time the deposit has been created in the vega network |






<a name="vega.Erc20WithdrawExt"></a>

### Erc20WithdrawExt
An extension of data required for the withdraw submissions


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| receiverAddress | [string](#string) |  | The address into which the bridge will release the funds |






<a name="vega.ErrorDetail"></a>

### ErrorDetail
Represents Vega domain specific error information over gRPC/Protobuf.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [int32](#int32) |  | A Vega API domain specific unique error code, useful for client side mappings. e.g. 10004 |
| message | [string](#string) |  | A message that describes the error in more detail, should describe the problem encountered. |
| inner | [string](#string) |  | Any inner error information that could add more context, or be helpful for error reporting. |






<a name="vega.Fee"></a>

### Fee
Represents any fees paid by a party, resulting from a trade.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| makerFee | [uint64](#uint64) |  | Fee amount paid to the non-aggressive party of the trade. |
| infrastructureFee | [uint64](#uint64) |  | Fee amount paid for maintaining the Vega infrastructure. |
| liquidityFee | [uint64](#uint64) |  | Fee amount paid to market makers. |






<a name="vega.FinancialAmount"></a>

### FinancialAmount
Asset value information used within a transfer.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| amount | [int64](#int64) |  | A signed integer amount of asset. |
| asset | [string](#string) |  | Asset identifier. |






<a name="vega.LedgerEntry"></a>

### LedgerEntry
Represents a ledger entry on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fromAccount | [string](#string) |  | One or more accounts to transfer from. |
| toAccount | [string](#string) |  | One or more accounts to transfer to. |
| amount | [uint64](#uint64) |  | An amount to transfer. |
| reference | [string](#string) |  | A reference for auditing purposes. |
| type | [string](#string) |  | Type of ledger entry. |
| timestamp | [int64](#int64) |  | Timestamp for the time the ledger entry was created, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |






<a name="vega.LiquidityOrder"></a>

### LiquidityOrder



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reference | [PeggedReference](#vega.PeggedReference) |  |  |
| proportion | [uint32](#uint32) |  |  |
| offset | [int64](#int64) |  |  |






<a name="vega.LiquidityOrderReference"></a>

### LiquidityOrderReference
A pair of a liquidity order and the id of the generated order by the core


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  | The ID of the pegged order generated by the core to fullfill this liquidityOrder |
| liquidityOrder | [LiquidityOrder](#vega.LiquidityOrder) |  | The liquidity order from the original submission |






<a name="vega.LiquidityProvision"></a>

### LiquidityProvision
An Liquidity provider commitment


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | LiquidityProvision identifier |
| partyID | [string](#string) |  | Party who created the provision |
| createdAt | [int64](#int64) |  | Timestamp for when the order was created at, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| updatedAt | [int64](#int64) |  | Timestamp for when the order was updated at, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| marketID | [string](#string) |  | Market identifier for the order. Required field. |
| commitmentAmount | [uint64](#uint64) |  | Specified as a unitless number that represents the amount of settlement asset of the market. |
| fee | [string](#string) |  | nominated liquidity fee factor, which is an input to the calculation of taker fees on the market, as per seeting fees and rewarding liquidity providers. |
| sells | [LiquidityOrderReference](#vega.LiquidityOrderReference) | repeated | a set of liquidity sell orders to meet the liquidity provision obligation, see MM orders spec. |
| buys | [LiquidityOrderReference](#vega.LiquidityOrderReference) | repeated | a set of liquidity buy orders to meet the liquidity provision obligation, see MM orders spec. |
| version | [string](#string) |  | version of this LiquidityProvision order |
| status | [LiquidityProvision.Status](#vega.LiquidityProvision.Status) |  | the status of this liquidity provision |






<a name="vega.LiquidityProvisionSubmission"></a>

### LiquidityProvisionSubmission



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier for the order. Required field. |
| commitmentAmount | [uint64](#uint64) |  | Specified as a unitless number that represents the amount of settlement asset of the market. |
| fee | [string](#string) |  | nominated liquidity fee factor, which is an input to the calculation of taker fees on the market, as per seeting fees and rewarding liquidity providers. |
| Sells | [LiquidityOrder](#vega.LiquidityOrder) | repeated | a set of liquidity sell orders to meet the liquidity provision obligation, see MM orders spec. |
| Buys | [LiquidityOrder](#vega.LiquidityOrder) | repeated | a set of liquidity buy orders to meet the liquidity provision obligation, see MM orders spec. |






<a name="vega.MarginLevels"></a>

### MarginLevels
Represents the margin levels for a party on a market at a given time.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maintenanceMargin | [uint64](#uint64) |  | Maintenance margin value. |
| searchLevel | [uint64](#uint64) |  | Search level value. |
| initialMargin | [uint64](#uint64) |  | Initial margin value. |
| collateralReleaseLevel | [uint64](#uint64) |  | Collateral release level value. |
| partyID | [string](#string) |  | Party identifier. |
| marketID | [string](#string) |  | Market identifier. |
| asset | [string](#string) |  | Asset identifier. |
| timestamp | [int64](#int64) |  | Timestamp for the time the ledger entry was created, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |






<a name="vega.MarketData"></a>

### MarketData
Represents data generated by a market when open.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| markPrice | [uint64](#uint64) |  | Mark price, as an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. |
| bestBidPrice | [uint64](#uint64) |  | Highest price level on an order book for buy orders, as an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. |
| bestBidVolume | [uint64](#uint64) |  | Aggregated volume being bid at the best bid price. |
| bestOfferPrice | [uint64](#uint64) |  | Lowest price level on an order book for offer orders. |
| bestOfferVolume | [uint64](#uint64) |  | Aggregated volume being offered at the best offer price, as an integer, for example `123456` is a correctly // formatted price of `1.23456` assuming market configured to 5 decimal places. |
| bestStaticBidPrice | [uint64](#uint64) |  | Highest price on the order book for buy orders not including pegged orders |
| bestStaticBidVolume | [uint64](#uint64) |  | Total volume at the best static bid price excluding pegged orders |
| bestStaticOfferPrice | [uint64](#uint64) |  | Lowest price on the order book for sell orders not including pegged orders |
| bestStaticOfferVolume | [uint64](#uint64) |  | Total volume at the best static offer price excluding pegged orders |
| midPrice | [uint64](#uint64) |  | Arithmetic average of the best bid price and best offer price, as an integer, for example `123456` is a correctly // formatted price of `1.23456` assuming market configured to 5 decimal places. |
| staticMidPrice | [uint64](#uint64) |  | Arithmetic average of the best static bid price and best static offer price |
| market | [string](#string) |  | Market identifier for the data. |
| timestamp | [int64](#int64) |  | Timestamp at which this mark price was relevant, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| openInterest | [uint64](#uint64) |  | The sum of the size of all positions greater than 0 on the market. |
| auctionEnd | [int64](#int64) |  | Time in seconds until the end of the auction (0 if currently not in auction period). |
| auctionStart | [int64](#int64) |  | Time until next auction (used in FBA&#39;s) - currently always 0. |
| indicativePrice | [uint64](#uint64) |  | indicative price (zero if not in auction) |
| indicativeVolume | [uint64](#uint64) |  | indicative volume (zero if not in auction) |
| marketState | [MarketState](#vega.MarketState) |  | the current state of the market |
| trigger | [AuctionTrigger](#vega.AuctionTrigger) |  | if the market is in auction state, this field indicates what triggered the auction |
| targetStake | [string](#string) |  | the targeted stake for the given market |
| suppliedStake | [string](#string) |  | the available stake fo the given market |
| priceMonitoringBounds | [PriceMonitoringBounds](#vega.PriceMonitoringBounds) | repeated | a list of price monitoring bounds per current time |






<a name="vega.MarketDepth"></a>

### MarketDepth
Represents market depth or order book data for the specified market on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. |
| buy | [PriceLevel](#vega.PriceLevel) | repeated | Collection of price levels for the buy side of the book. |
| sell | [PriceLevel](#vega.PriceLevel) | repeated | Collection of price levels for the sell side of the book. |
| sequenceNumber | [uint64](#uint64) |  | Sequence number for the market depth dump |






<a name="vega.MarketDepthUpdate"></a>

### MarketDepthUpdate
Represents the changed market depth since the last update


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. |
| buy | [PriceLevel](#vega.PriceLevel) | repeated | Collection of updated price levels for the buy side of the book. |
| sell | [PriceLevel](#vega.PriceLevel) | repeated | Collection of updated price levels for the sell side of the book. |
| sequenceNumber | [uint64](#uint64) |  | Sequence number for the market depth update |






<a name="vega.NetworkParameter"></a>

### NetworkParameter



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Key | [string](#string) |  |  |
| Value | [string](#string) |  |  |






<a name="vega.NodeRegistration"></a>

### NodeRegistration
Used to Register a node as a validator during network start-up.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pubKey | [bytes](#bytes) |  | Public key, required field. |
| chainPubKey | [bytes](#bytes) |  | Public key for the blockchain, required field. |






<a name="vega.NodeSignature"></a>

### NodeSignature
Represents a signature from a validator, to be used by a foreign chain in order to recognise a decision taken by the Vega network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | The identifier of the resource being signed. |
| sig | [bytes](#bytes) |  | The signature. |
| kind | [NodeSignatureKind](#vega.NodeSignatureKind) |  | The kind of resource being signed. |






<a name="vega.NodeVote"></a>

### NodeVote
Used when a node votes for validating a given resource exists or is valid.
For example, an ERC20 deposit is valid and exists on ethereum.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pubKey | [bytes](#bytes) |  | Public key, required field. |
| reference | [string](#string) |  | Reference, required field. |






<a name="vega.Order"></a>

### Order
An order can be submitted, amended and cancelled on Vega in an attempt to make trades with other parties.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Unique identifier for the order (set by the system after consensus). |
| marketID | [string](#string) |  | Market identifier for the order. |
| partyID | [string](#string) |  | Party identifier for the order. |
| side | [Side](#vega.Side) |  | Side for the order, e.g. SIDE_BUY or SIDE_SELL. See [`Side`](#vega.Side). |
| price | [uint64](#uint64) |  | Price for the order, the price is an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. |
| size | [uint64](#uint64) |  | Size for the order, for example, in a futures market the size equals the number of contracts. |
| remaining | [uint64](#uint64) |  | Size remaining, when this reaches 0 then the order is fully filled and status becomes STATUS_FILLED. |
| timeInForce | [Order.TimeInForce](#vega.Order.TimeInForce) |  | Time in force indicates how long an order will remain active before it is executed or expires. See [`Order.TimeInForce`](#vega.Order.TimeInForce). |
| type | [Order.Type](#vega.Order.Type) |  | Type for the order. See [`Order.Type`](#vega.Order.Type). |
| createdAt | [int64](#int64) |  | Timestamp for when the order was created at, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| status | [Order.Status](#vega.Order.Status) |  | The current status for the order. See [`Order.Status`](#vega.Order.Status). For detail on `STATUS_REJECTED` please check the [`OrderError`](#vega.OrderError) value given in the `reason` field. |
| expiresAt | [int64](#int64) |  | Timestamp for when the order will expire, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. Valid only for [`Order.TimeInForce`](#vega.Order.TimeInForce)`.TIF_GTT`. |
| reference | [string](#string) |  | Reference given for the order, this is typically used to retrieve an order submitted through consensus. Currently set internally by the node to return a unique reference identifier for the order submission. |
| reason | [OrderError](#vega.OrderError) |  | If the Order `status` is `STATUS_REJECTED` then an [`OrderError`](#vega.OrderError) reason will be specified. The default for this field is `ORDER_ERROR_NONE` which signifies that there were no errors. |
| updatedAt | [int64](#int64) |  | Timestamp for when the Order was last updated, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| version | [uint64](#uint64) |  | The version for the order, initial value is version 1 and is incremented after each successful amend |
| batchID | [uint64](#uint64) |  | Batch identifier for the order, used internally for orders submitted during auctions to keep track of the auction batch this order falls under (required for fees calculation). |
| peggedOrder | [PeggedOrder](#vega.PeggedOrder) |  | If this order represents a pegged order, the details are supplied here |






<a name="vega.OrderAmendment"></a>

### OrderAmendment
An order amendment is a request to amend or update an existing order on Vega.

The `orderID`, `partyID` and `marketID` fields are used for lookup of the order only and cannot be amended by this command.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  | Order identifier, this is required to find the order and will not be updated. Required field. |
| partyID | [string](#string) |  | Party identifier, this is required to find the order and will not be updated. Required field. |
| marketID | [string](#string) |  | Market identifier, this is required to find the order and will not be updated. |
| price | [Price](#vega.Price) |  | Amend the price for the order, if the Price value is set, otherwise price will remain unchanged. See [`Price`](#vega.Price). |
| sizeDelta | [int64](#int64) |  | Amend the size for the order by the delta specified. To reduce the size from the current value set a negative integer value. To increase the size from the current value, set a positive integer value. To leave the size unchanged set a value of zero. |
| expiresAt | [Timestamp](#vega.Timestamp) |  | Amend the expiry time for the order, if the Timestamp value is set, otherwise expiry time will remain unchanged. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| timeInForce | [Order.TimeInForce](#vega.Order.TimeInForce) |  | Amend the time in force for the order, set to TIF_UNSPECIFIED to remain unchanged. See [`TimeInForce`](#api.VegaTimeResponse).`timestamp`. |
| peggedOffset | [google.protobuf.Int64Value](#google.protobuf.Int64Value) |  | Amend the pegged order offset for the order |
| peggedReference | [PeggedReference](#vega.PeggedReference) |  | Amend the pegged order reference for the order See [`PeggedReference`](#api.PeggedReference). |






<a name="vega.OrderCancellation"></a>

### OrderCancellation
An order cancellation is a request to cancel an existing order on Vega.

The following three fields are used for lookup of the order only:


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  | Unique identifier for the order (set by the system after consensus). Required field. |
| marketID | [string](#string) |  | Market identifier for the order. Required field. |
| partyID | [string](#string) |  | Party identifier for the order. Required field. |






<a name="vega.OrderCancellationConfirmation"></a>

### OrderCancellationConfirmation
Used when cancelling an Order.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| order | [Order](#vega.Order) |  | The order that was cancelled. |






<a name="vega.OrderConfirmation"></a>

### OrderConfirmation
Used when confirming an Order.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| order | [Order](#vega.Order) |  | The order that was confirmed. |
| trades | [Trade](#vega.Trade) | repeated | 0 or more trades that were emitted. |
| passiveOrdersAffected | [Order](#vega.Order) | repeated | 0 or more passive orders that were affected. |






<a name="vega.OrderSubmission"></a>

### OrderSubmission
An order submission is a request to submit or create a new order on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Unique identifier for the order (set by the system after consensus). |
| marketID | [string](#string) |  | Market identifier for the order. Required field. |
| partyID | [string](#string) |  | Party identifier for the order. Required field. |
| price | [uint64](#uint64) |  | Price for the order, the price is an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. Required field for Limit orders, however it is not required for market orders. |
| size | [uint64](#uint64) |  | Size for the order, for example, in a futures market the size equals the number of contracts. Cannot be negative. |
| side | [Side](#vega.Side) |  | Side for the order, e.g. SIDE_BUY or SIDE_SELL. See [`Side`](#vega.Side). Required field. |
| timeInForce | [Order.TimeInForce](#vega.Order.TimeInForce) |  | Time in force indicates how long an order will remain active before it is executed or expires. See [`Order.TimeInForce`](#vega.Order.TimeInForce). Required field. |
| expiresAt | [int64](#int64) |  | Timestamp for when the order will expire, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. Required field only for [`Order.TimeInForce`](#vega.Order.TimeInForce)`.TIF_GTT`. |
| type | [Order.Type](#vega.Order.Type) |  | Type for the order. See [`Order.Type`](#vega.Order.Type). Required field. |
| reference | [string](#string) |  | Reference given for the order, this is typically used to retrieve an order submitted through consensus. Currently set internally by the node to return a unique reference identifier for the order submission. |
| peggedOrder | [PeggedOrder](#vega.PeggedOrder) |  | Pegged order details. If this sub message is supplied then the fields are used to configure a pegged order |






<a name="vega.Party"></a>

### Party
A party represents an entity who wishes to trade on or query a Vega network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | A unique identifier for the party, typically represented by a public key. |






<a name="vega.PeggedOrder"></a>

### PeggedOrder



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reference | [PeggedReference](#vega.PeggedReference) |  | Which price point are we linked to |
| offset | [int64](#int64) |  | Offset from the price reference |






<a name="vega.Position"></a>

### Position
Represents position data for a party on the specified market on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. |
| partyID | [string](#string) |  | Party identifier. |
| openVolume | [int64](#int64) |  | Open volume for the position. Value is signed &#43;ve for long and -ve for short. |
| realisedPNL | [int64](#int64) |  | Realised profit and loss for the position. Value is signed &#43;ve for long and -ve for short. |
| unrealisedPNL | [int64](#int64) |  | Unrealised profit and loss for the position. Value is signed &#43;ve for long and -ve for short. |
| averageEntryPrice | [uint64](#uint64) |  | Average entry price for the position, the price is an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. |
| updatedAt | [int64](#int64) |  | last time the position was updated |






<a name="vega.PositionTrade"></a>

### PositionTrade



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| volume | [int64](#int64) |  | Volume for the position trade. Value is signed &#43;ve for long and -ve for short. |
| price | [uint64](#uint64) |  | Price for the position trade, the price is an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. |






<a name="vega.Price"></a>

### Price



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [uint64](#uint64) |  | Price value, given as an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. |






<a name="vega.PriceLevel"></a>

### PriceLevel
Represents a price level from market depth or order book data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| price | [uint64](#uint64) |  | Price for the price level, the price is an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. |
| numberOfOrders | [uint64](#uint64) |  | Number of orders at the price level. |
| volume | [uint64](#uint64) |  | Volume at the price level. |






<a name="vega.PriceMonitoringBounds"></a>

### PriceMonitoringBounds
Represents a list of valid (at the current time) price ranges per associated trigger


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| minValidPrice | [uint64](#uint64) |  | Minimum price that isn&#39;t currently breaching the specified price monitoring trigger |
| maxValidPrice | [uint64](#uint64) |  | Maximum price that isn&#39;t currently breaching the specified price monitoring trigger |
| trigger | [PriceMonitoringTrigger](#vega.PriceMonitoringTrigger) |  | Price monitoring trigger associated with the bounds |
| referencePrice | [double](#double) |  | Reference price used to calculate the valid price range |






<a name="vega.RiskFactor"></a>

### RiskFactor
Risk factors are used to calculate the current risk associated with orders trading on a given market.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| market | [string](#string) |  | Market ID that relates to this risk factor. |
| short | [double](#double) |  | Short Risk factor value. |
| long | [double](#double) |  | Long Risk factor value. |






<a name="vega.RiskResult"></a>

### RiskResult
Risk results are calculated internally by Vega to attempt to maintain safe trading.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| updatedTimestamp | [int64](#int64) |  | Timestamp for when risk factors were generated. |
| riskFactors | [RiskResult.RiskFactorsEntry](#vega.RiskResult.RiskFactorsEntry) | repeated | Risk factors (long and short) for each margin-able asset/currency (usually == settlement assets) in the market. |
| nextUpdateTimestamp | [int64](#int64) |  | Timestamp for when risk factors are expected to change (or empty if risk factors are continually updated). |
| predictedNextRiskFactors | [RiskResult.PredictedNextRiskFactorsEntry](#vega.RiskResult.PredictedNextRiskFactorsEntry) | repeated | Predicted risk factors at next change (what they would be if the change occurred now). |






<a name="vega.RiskResult.PredictedNextRiskFactorsEntry"></a>

### RiskResult.PredictedNextRiskFactorsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [RiskFactor](#vega.RiskFactor) |  |  |






<a name="vega.RiskResult.RiskFactorsEntry"></a>

### RiskResult.RiskFactorsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [RiskFactor](#vega.RiskFactor) |  |  |






<a name="vega.Signature"></a>

### Signature
A signature to be authenticate a transaction
and to be verified by the vega network


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sig | [bytes](#bytes) |  | The bytes of the signature |
| algo | [string](#string) |  | The algorithm used to create the signature |
| version | [uint64](#uint64) |  | The version of the signature used to create the signature |






<a name="vega.SignedBundle"></a>

### SignedBundle
A bundle of a transaction and it&#39;s signature.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tx | [bytes](#bytes) |  | Transaction payload (proto marshalled). |
| sig | [Signature](#vega.Signature) |  | The signature authenticating the transaction. |






<a name="vega.Statistics"></a>

### Statistics
Vega domain specific statistics as reported by the node the caller is connected to.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blockHeight | [uint64](#uint64) |  | Current block height as reported by the Vega blockchain. |
| backlogLength | [uint64](#uint64) |  | Current backlog length (number of transactions) that are waiting to be included in a block. |
| totalPeers | [uint64](#uint64) |  | Total number of connected peers to this node. |
| genesisTime | [string](#string) |  | Genesis block date and time formatted in ISO-8601 datetime format with nanosecond precision. |
| currentTime | [string](#string) |  | Current system date and time formatted in ISO-8601 datetime format with nanosecond precision. |
| vegaTime | [string](#string) |  | Current Vega date and time formatted in ISO-8601 datetime format with nanosecond precision. |
| status | [ChainStatus](#vega.ChainStatus) |  | Status of the connection to the Vega blockchain. See [`ChainStatus`](#vega.ChainStatus). |
| txPerBlock | [uint64](#uint64) |  | Transactions per block. |
| averageTxBytes | [uint64](#uint64) |  | Average transaction size in bytes. |
| averageOrdersPerBlock | [uint64](#uint64) |  | Average orders per block. |
| tradesPerSecond | [uint64](#uint64) |  | Trades emitted per second. |
| ordersPerSecond | [uint64](#uint64) |  | Orders affected per second. |
| totalMarkets | [uint64](#uint64) |  | Total markets on this Vega network. |
| totalAmendOrder | [uint64](#uint64) |  | Total number of order amendments since genesis (on all markets). |
| totalCancelOrder | [uint64](#uint64) |  | Total number of order cancellations since genesis (on all markets). |
| totalCreateOrder | [uint64](#uint64) |  | Total number of order submissions since genesis (on all markets). |
| totalOrders | [uint64](#uint64) |  | Total number of orders affected since genesis (on all markets). |
| totalTrades | [uint64](#uint64) |  | Total number of trades emitted since genesis (on all markets). |
| orderSubscriptions | [uint32](#uint32) |  | Current number of stream subscribers to order data. |
| tradeSubscriptions | [uint32](#uint32) |  | Current number of stream subscribers to trade data. |
| candleSubscriptions | [uint32](#uint32) |  | Current number of stream subscribers to candle-stick data. |
| marketDepthSubscriptions | [uint32](#uint32) |  | Current number of stream subscribers to market depth data. |
| positionsSubscriptions | [uint32](#uint32) |  | Current number of stream subscribers to positions data. |
| accountSubscriptions | [uint32](#uint32) |  | Current number of stream subscribers to account data. |
| marketDataSubscriptions | [uint32](#uint32) |  | Current number of stream subscribers to market data. |
| appVersionHash | [string](#string) |  | The version hash of the Vega node software. |
| appVersion | [string](#string) |  | The version of the Vega node software. |
| chainVersion | [string](#string) |  | The version of the underlying Vega blockchain. |
| blockDuration | [uint64](#uint64) |  | Current block duration, in nanoseconds. |
| uptime | [string](#string) |  | Total uptime for this node formatted in ISO-8601 datetime format with nanosecond precision. |
| chainID | [string](#string) |  | Unique identifier for the underlying Vega blockchain. |
| marketDepthUpdatesSubscriptions | [uint32](#uint32) |  | Current number of stream subscribers to market depth update data. |






<a name="vega.Timestamp"></a>

### Timestamp
A timestamp in nanoseconds since epoch.
See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [int64](#int64) |  | Timestamp value. |






<a name="vega.Trade"></a>

### Trade
A trade occurs when an aggressive order crosses one or more passive orders on the order book for a market on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Unique identifier for the trade (generated by Vega). |
| marketID | [string](#string) |  | Market identifier (the market that the trade occurred on). |
| price | [uint64](#uint64) |  | Price for the trade, the price is an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. |
| size | [uint64](#uint64) |  | Size filled for the trade. |
| buyer | [string](#string) |  | Unique party identifier for the buyer. |
| seller | [string](#string) |  | Unique party identifier for the seller. |
| aggressor | [Side](#vega.Side) |  | Direction of the aggressive party e.g. SIDE_BUY or SIDE_SELL. See [`Side`](#vega.Side). |
| buyOrder | [string](#string) |  | Identifier of the order from the buy side. |
| sellOrder | [string](#string) |  | Identifier of the order from the sell side. |
| timestamp | [int64](#int64) |  | Timestamp for when the trade occurred, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| type | [Trade.Type](#vega.Trade.Type) |  | Type for the trade. See [`Trade.Type`](#vega.Trade.Type). |
| buyerFee | [Fee](#vega.Fee) |  | Fee amount charged to the buyer party for the trade. |
| sellerFee | [Fee](#vega.Fee) |  | Fee amount charged to the seller party for the trade. |
| buyerAuctionBatch | [uint64](#uint64) |  | Auction batch number that the buy side order was placed in. |
| sellerAuctionBatch | [uint64](#uint64) |  | Auction batch number that the sell side order was placed in. |






<a name="vega.TradeSet"></a>

### TradeSet



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trades | [Trade](#vega.Trade) | repeated | A set of one or more trades. |






<a name="vega.Transaction"></a>

### Transaction
Represents a transaction to be sent to Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| inputData | [bytes](#bytes) |  | One of the set of Vega commands (proto marshalled). |
| nonce | [uint64](#uint64) |  | A random number used to provided uniqueness and prevents against replay attack. |
| blockHeight | [uint64](#uint64) |  | The block height associated to the transaction. This should always be current height of the node at the time of sending the Tx. BlockHeight is used as a mechanism for replay protection. |
| address | [bytes](#bytes) |  | The address of the sender. |
| pubKey | [bytes](#bytes) |  | The public key of the sender. |






<a name="vega.Transfer"></a>

### Transfer
Represents a financial transfer within Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | [string](#string) |  | Party identifier for the owner of the transfer. |
| amount | [FinancialAmount](#vega.FinancialAmount) |  | A financial amount (of an asset) to transfer. |
| type | [TransferType](#vega.TransferType) |  | The type of transfer, gives the reason for the transfer. |
| minAmount | [int64](#int64) |  | A minimum amount. |






<a name="vega.TransferBalance"></a>

### TransferBalance
Represents the balance for an account during a transfer.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account | [Account](#vega.Account) |  | The account relating to the transfer |
| balance | [uint64](#uint64) |  | The balance relating to the transfer |






<a name="vega.TransferRequest"></a>

### TransferRequest
Represents a request to transfer from one set of accounts to another.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fromAccount | [Account](#vega.Account) | repeated | One or more accounts to transfer from. |
| toAccount | [Account](#vega.Account) | repeated | One or more accounts to transfer to. |
| amount | [uint64](#uint64) |  | An amount to transfer for the asset. |
| minAmount | [uint64](#uint64) |  | A minimum amount. |
| asset | [string](#string) |  | Asset identifier. |
| reference | [string](#string) |  | A reference for auditing purposes. |






<a name="vega.TransferResponse"></a>

### TransferResponse
Represents the response from a transfer.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| transfers | [LedgerEntry](#vega.LedgerEntry) | repeated | One or more ledger entries representing the transfers. |
| balances | [TransferBalance](#vega.TransferBalance) | repeated | One or more account balances. |






<a name="vega.WithdrawExt"></a>

### WithdrawExt



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| erc20 | [Erc20WithdrawExt](#vega.Erc20WithdrawExt) |  |  |






<a name="vega.WithdrawSubmission"></a>

### WithdrawSubmission
A request for withdrawing funds from a trader


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | The party which wants to withdraw funds |
| amount | [uint64](#uint64) |  | The amount to be withdrawn |
| asset | [string](#string) |  | The asset we want to withdraw |
| ext | [WithdrawExt](#vega.WithdrawExt) |  | foreign chain specifics |






<a name="vega.Withdrawal"></a>

### Withdrawal
The representation of a withdrawal in the vega network


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | The id of the withdrawal |
| partyID | [string](#string) |  | The party initiating the withdrawal |
| amount | [uint64](#uint64) |  | The amount to be withdrawn |
| asset | [string](#string) |  | The asset we want to withdraw funds from |
| status | [Withdrawal.Status](#vega.Withdrawal.Status) |  | The status of this withdrawal |
| ref | [string](#string) |  | The reference which is used by the foreign chain to refer to this withdrawal |
| expiry | [int64](#int64) |  | The time until when the withdrawal is valid |
| txHash | [string](#string) |  | The hash of the foreign chain for this transaction |
| createdTimestamp | [int64](#int64) |  | The time at which the network started to process this withdrawal |
| withdrawnTimestamp | [int64](#int64) |  | The time at which the withdrawal was finalized by the network |
| ext | [WithdrawExt](#vega.WithdrawExt) |  | foreign chain specifis |








<a name="vega.AccountType"></a>

### AccountType
Various collateral/account types as used by Vega.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ACCOUNT_TYPE_UNSPECIFIED | 0 | Default value. |
| ACCOUNT_TYPE_INSURANCE | 1 | Insurance pool accounts contain insurance pool funds for a market. |
| ACCOUNT_TYPE_SETTLEMENT | 2 | Settlement accounts exist only during settlement or mark-to-market. |
| ACCOUNT_TYPE_MARGIN | 3 | Margin accounts contain margin funds for a party and each party will have multiple margin accounts, one for each market they have traded in.

Margin account funds will alter as margin requirements on positions change. |
| ACCOUNT_TYPE_GENERAL | 4 | General accounts contains general funds for a party. A party will have multiple general accounts, one for each asset they want to trade with.

General accounts are where funds are initially deposited or withdrawn from. It is also the account where funds are taken to fulfil fees and initial margin requirements. |
| ACCOUNT_TYPE_FEES_INFRASTRUCTURE | 5 | Infrastructure accounts contain fees earned by providing infrastructure on Vega. |
| ACCOUNT_TYPE_FEES_LIQUIDITY | 6 | Liquidity accounts contain fees earned by providing liquidity on Vega markets. |
| ACCOUNT_TYPE_FEES_MAKER | 7 | This account is created to hold fees earned by placing orders that sit on the book and are then matched with an incoming order to create a trade. These fees reward traders who provide the best priced liquidity that actually allows trading to take place. |
| ACCOUNT_TYPE_LOCK_WITHDRAW | 8 | This account is created to lock funds to be withdrawn by parties |
| ACCOUNT_TYPE_BOND | 9 | This account is created to maintain liquidity providers funds commitments |



<a name="vega.AuctionTrigger"></a>

### AuctionTrigger
What triggered an auction (if any)

| Name | Number | Description |
| ---- | ------ | ----------- |
| AUCTION_TRIGGER_UNSPECIFIED | 0 | No auction triggered |
| AUCTION_TRIGGER_BATCH | 1 | Batch auction |
| AUCTION_TRIGGER_OPENING | 2 | Opening auction |
| AUCTION_TRIGGER_PRICE | 3 | Price monitoring trigger |
| AUCTION_TRIGGER_LIQUIDITY | 4 | liquidity monitoring trigger |



<a name="vega.ChainStatus"></a>

### ChainStatus
The Vega blockchain status as reported by the node the caller is connected to.

| Name | Number | Description |
| ---- | ------ | ----------- |
| CHAIN_STATUS_UNSPECIFIED | 0 | Default value, always invalid. |
| CHAIN_STATUS_DISCONNECTED | 1 | Blockchain is disconnected. |
| CHAIN_STATUS_REPLAYING | 2 | Blockchain is replaying historic transactions. |
| CHAIN_STATUS_CONNECTED | 3 | Blockchain is connected and receiving transactions. |



<a name="vega.Deposit.Status"></a>

### Deposit.Status
The status of the deposit

| Name | Number | Description |
| ---- | ------ | ----------- |
| DEPOSIT_STATUS_UNSPECIFIED | 0 | Default value, always invalid. |
| DEPOSIT_STATUS_OPEN | 1 | The deposit is being processed by the network |
| DEPOSIT_STATUS_CANCELLED | 2 | The deposit has been cancelled by the network |
| DEPOSIT_STATUS_FINALIZED | 3 | The deposit has been finalized and accounts have been updated |



<a name="vega.Interval"></a>

### Interval
Represents a set of time intervals that are used when querying for candle-stick data.

| Name | Number | Description |
| ---- | ------ | ----------- |
| INTERVAL_UNSPECIFIED | 0 | Default value, always invalid. |
| INTERVAL_I1M | 60 | 1 minute. |
| INTERVAL_I5M | 300 | 5 minutes. |
| INTERVAL_I15M | 900 | 15 minutes. |
| INTERVAL_I1H | 3600 | 1 hour. |
| INTERVAL_I6H | 21600 | 6 hours. |
| INTERVAL_I1D | 86400 | 1 day. |



<a name="vega.LiquidityProvision.Status"></a>

### LiquidityProvision.Status
the status of a liquidity provision order

| Name | Number | Description |
| ---- | ------ | ----------- |
| LIQUIDITY_PROVISION_STATUS_UNSPECIFIED | 0 | The default value |
| LIQUIDITY_PROVISION_STATUS_ACTIVE | 1 | The liquidity provision is active |
| LIQUIDITY_PROVISION_STATUS_STOPPED | 2 | The liquidity provision was stopped by the network |
| LIQUIDITY_PROVISION_STATUS_CANCELLED | 3 | The liquidity provision was cancelled by the MM. |
| LIQUIDITY_PROVISION_STATUS_REJECTED | 4 | The liquidity provision was invalid and got rejected. |



<a name="vega.MarketState"></a>

### MarketState
What mode is the market currently running, also known as market state.

| Name | Number | Description |
| ---- | ------ | ----------- |
| MARKET_STATE_UNSPECIFIED | 0 | Default value, this is invalid |
| MARKET_STATE_CONTINUOUS | 1 | Normal trading |
| MARKET_STATE_BATCH_AUCTION | 2 | Auction trading (FBA) |
| MARKET_STATE_OPENING_AUCTION | 3 | Opening auction |
| MARKET_STATE_MONITORING_AUCTION | 4 | Auction triggered by monitoring |



<a name="vega.NodeSignatureKind"></a>

### NodeSignatureKind
The kind of the signature created by a node, for example, allowlisting a new asset, withdrawal etc.

| Name | Number | Description |
| ---- | ------ | ----------- |
| NODE_SIGNATURE_KIND_UNSPECIFIED | 0 | represents a unspecified / missing value from the input |
| NODE_SIGNATURE_KIND_ASSET_NEW | 1 | represents a signature for a new asset allowlisting |
| NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL | 2 | represents a signature for a asset withdrawal |



<a name="vega.Order.Status"></a>

### Order.Status
Status values for an order.
See resulting status in [What order types are available to trade on Vega?](https://docs.vega.xyz/docs/trading-questions/#what-order-types-are-available-to-trade-on-vega) for more detail.

| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_INVALID | 0 | Default value, always invalid. |
| STATUS_ACTIVE | 1 | Used for active unfilled or partially filled orders. |
| STATUS_EXPIRED | 2 | Used for expired GTT orders. |
| STATUS_CANCELLED | 3 | Used for orders cancelled by the party that created the order. |
| STATUS_STOPPED | 4 | Used for unfilled FOK or IOC orders, and for orders that were stopped by the network. |
| STATUS_FILLED | 5 | Used for closed fully filled orders. |
| STATUS_REJECTED | 6 | Used for orders when not enough collateral was available to fill the margin requirements. |
| STATUS_PARTIALLY_FILLED | 7 | Used for closed partially filled IOC orders. |
| STATUS_PARKED | 8 | The order ha been removed from the book and has been parked, applies to pegged order only |



<a name="vega.Order.TimeInForce"></a>

### Order.TimeInForce
Time in Force for an order.
See [What order types are available to trade on Vega?](https://docs.vega.xyz/docs/trading-questions/#what-order-types-are-available-to-trade-on-vega) for more detail.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TIF_UNSPECIFIED | 0 | Default value for TimeInForce, can be valid for an amend. |
| TIF_GTC | 1 | Good until cancelled. |
| TIF_GTT | 2 | Good until specified time. |
| TIF_IOC | 3 | Immediate or cancel. |
| TIF_FOK | 4 | Fill or kill. |
| TIF_GFA | 5 | good for auction |
| TIF_GFN | 6 | good for normal |



<a name="vega.Order.Type"></a>

### Order.Type
Type values for an order.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Default value, always invalid. |
| TYPE_LIMIT | 1 | Used for Limit orders. |
| TYPE_MARKET | 2 | Used for Market orders. |
| TYPE_NETWORK | 3 | Used for orders where the initiating party is the network (with distressed traders). |



<a name="vega.OrderError"></a>

### OrderError
OrderError codes are returned in the `[Order](#vega.Order).reason` field.
If there is an issue with an order during it&#39;s life-cycle, it will be marked with `status.ORDER_STATUS_REJECTED`.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ORDER_ERROR_NONE | 0 | Default value, no error reported. |
| ORDER_ERROR_INVALID_MARKET_ID | 1 | Order was submitted for a market that does not exist. |
| ORDER_ERROR_INVALID_ORDER_ID | 2 | Order was submitted with an invalid identifier. |
| ORDER_ERROR_OUT_OF_SEQUENCE | 3 | Order was amended with a sequence number that was not previous version &#43; 1. |
| ORDER_ERROR_INVALID_REMAINING_SIZE | 4 | Order was amended with an invalid remaining size (e.g. remaining greater than total size). |
| ORDER_ERROR_TIME_FAILURE | 5 | Node was unable to get Vega (blockchain) time. |
| ORDER_ERROR_REMOVAL_FAILURE | 6 | Failed to remove an order from the book. |
| ORDER_ERROR_INVALID_EXPIRATION_DATETIME | 7 | An order with `TimeInForce.TIF_GTT` was submitted or amended with an expiration that was badly formatted or otherwise invalid. |
| ORDER_ERROR_INVALID_ORDER_REFERENCE | 8 | Order was submitted or amended with an invalid reference field. |
| ORDER_ERROR_EDIT_NOT_ALLOWED | 9 | Order amend was submitted for an order field that cannot not be amended (e.g. order identifier). |
| ORDER_ERROR_AMEND_FAILURE | 10 | Amend failure because amend details do not match original order. |
| ORDER_ERROR_NOT_FOUND | 11 | Order not found in an order book or store. |
| ORDER_ERROR_INVALID_PARTY_ID | 12 | Order was submitted with an invalid or missing party identifier. |
| ORDER_ERROR_MARKET_CLOSED | 13 | Order was submitted for a market that has closed. |
| ORDER_ERROR_MARGIN_CHECK_FAILED | 14 | Order was submitted, but the party did not have enough collateral to cover the order. |
| ORDER_ERROR_MISSING_GENERAL_ACCOUNT | 15 | Order was submitted, but the party did not have an account for this asset. |
| ORDER_ERROR_INTERNAL_ERROR | 16 | Unspecified internal error. |
| ORDER_ERROR_INVALID_SIZE | 17 | Order was submitted with an invalid or missing size (e.g. 0). |
| ORDER_ERROR_INVALID_PERSISTENCE | 18 | Order was submitted with an invalid persistence for its type. |
| ORDER_ERROR_INVALID_TYPE | 19 | Order was submitted with an invalid type field. |
| ORDER_ERROR_SELF_TRADING | 20 | Order was stopped as it would have traded with another order submitted from the same party. |
| ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES | 21 | Order was submitted, but the party did not have enough collateral to cover the fees for the order. |
| ORDER_ERROR_INCORRECT_MARKET_TYPE | 22 | Order was submitted with an incorrect or invalid market type. |
| ORDER_ERROR_INVALID_TIME_IN_FORCE | 23 | Order was submitted with invalid time in force |
| ORDER_ERROR_GFN_ORDER_DURING_AN_AUCTION | 24 | A GFN order has got to the market when it is in auction mode |
| ORDER_ERROR_GFA_ORDER_DURING_CONTINUOUS_TRADING | 25 | A GFA order has got to the market when it is in continuous trading mode |
| ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT | 26 | Attempt to amend order to GTT without ExpiryAt |
| ORDER_ERROR_EXPIRYAT_BEFORE_CREATEDAT | 27 | Attempt to amend ExpiryAt to a value before CreatedAt |
| ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT | 28 | Attempt to amend to GTC without an ExpiryAt value |
| ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC | 29 | Amending to FOK or IOC is invalid |
| ORDER_ERROR_CANNOT_AMEND_TO_GFA_OR_GFN | 30 | Amending to GFA or GFN is invalid |
| ORDER_ERROR_CANNOT_AMEND_FROM_GFA_OR_GFN | 31 | Amending from GFA or GFN is invalid |
| ORDER_ERROR_CANNOT_SEND_IOC_ORDER_DURING_AUCTION | 32 | IOC orders are not allowed during auction |
| ORDER_ERROR_CANNOT_SEND_FOK_ORDER_DURING_AUCTION | 33 | FOK orders are not allowed during auction |
| ORDER_ERROR_MUST_BE_LIMIT_ORDER | 34 | Pegged orders must be LIMIT orders |
| ORDER_ERROR_MUST_BE_GTT_OR_GTC | 35 | Pegged orders can only have TIF GTC or GTT |
| ORDER_ERROR_WITHOUT_REFERENCE_PRICE | 36 | Pegged order must have a reference price |
| ORDER_ERROR_BUY_CANNOT_REFERENCE_BEST_ASK_PRICE | 37 | Buy pegged order cannot reference best ask price |
| ORDER_ERROR_OFFSET_MUST_BE_LESS_OR_EQUAL_TO_ZERO | 38 | Pegged order offset must be &lt;= 0 |
| ORDER_ERROR_OFFSET_MUST_BE_LESS_THAN_ZERO | 39 | Pegged order offset must be &lt; 0 |
| ORDER_ERROR_OFFSET_MUST_BE_GREATER_OR_EQUAL_TO_ZERO | 40 | Pegged order offset must be &gt;= 0 |
| ORDER_ERROR_SELL_CANNOT_REFERENCE_BEST_BID_PRICE | 41 | Sell pegged order cannot reference best bid price |
| ORDER_ERROR_OFFSET_MUST_BE_GREATER_THAN_ZERO | 42 | Pegged order offset must be &gt; zero |
| ORDER_ERROR_INSUFFICIENT_ASSET_BALANCE | 43 | The party have an insufficient balance, or don&#39;t have a general account to submit the order (no deposits made for the required asset). |
| ORDER_ERROR_CANNOT_AMEND_PEGGED_ORDER_DETAILS_ON_NON_PEGGED_ORDER | 44 | Cannot amend a non pegged orders details |
| ORDER_ERROR_UNABLE_TO_REPRICE_PEGGED_ORDER | 45 | We are unable to reprice a pegged order because a market price is unavailable |



<a name="vega.PeggedReference"></a>

### PeggedReference
Which price point is the pegged order linked to

| Name | Number | Description |
| ---- | ------ | ----------- |
| PEGGED_REFERENCE_UNSPECIFIED | 0 | No reference given |
| PEGGED_REFERENCE_MID | 1 | MID price |
| PEGGED_REFERENCE_BEST_BID | 2 | BEST BID price |
| PEGGED_REFERENCE_BEST_ASK | 3 | BEST ASK price |



<a name="vega.Side"></a>

### Side
A side relates to the direction of an order, to Buy, or Sell.

| Name | Number | Description |
| ---- | ------ | ----------- |
| SIDE_UNSPECIFIED | 0 | Default value, always invalid. |
| SIDE_BUY | 1 | Buy order. |
| SIDE_SELL | 2 | Sell order. |



<a name="vega.Trade.Type"></a>

### Trade.Type
Type values for a trade.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Default value, always invalid. |
| TYPE_DEFAULT | 1 | Normal trading between two parties. |
| TYPE_NETWORK_CLOSE_OUT_GOOD | 2 | Trading initiated by the network with another party on the book, which helps to zero-out the positions of one or more distressed parties. |
| TYPE_NETWORK_CLOSE_OUT_BAD | 3 | Trading initiated by the network with another party off the book, with a distressed party in order to zero-out the position of the party. |



<a name="vega.TransferType"></a>

### TransferType
Transfers can occur between parties on Vega, these are the types that indicate why a transfer took place.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TRANSFER_TYPE_UNSPECIFIED | 0 | Default value, always invalid. |
| TRANSFER_TYPE_LOSS | 1 | Loss. |
| TRANSFER_TYPE_WIN | 2 | Win. |
| TRANSFER_TYPE_CLOSE | 3 | Close. |
| TRANSFER_TYPE_MTM_LOSS | 4 | Mark to market loss. |
| TRANSFER_TYPE_MTM_WIN | 5 | Mark to market win. |
| TRANSFER_TYPE_MARGIN_LOW | 6 | Margin too low. |
| TRANSFER_TYPE_MARGIN_HIGH | 7 | Margin too high. |
| TRANSFER_TYPE_MARGIN_CONFISCATED | 8 | Margin was confiscated. |
| TRANSFER_TYPE_MAKER_FEE_PAY | 9 | Pay maker fee. |
| TRANSFER_TYPE_MAKER_FEE_RECEIVE | 10 | Receive maker fee. |
| TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY | 11 | Pay infrastructure fee. |
| TRANSFER_TYPE_LIQUIDITY_FEE_PAY | 12 | Pay liquidity fee. |
| TRANSFER_TYPE_BOND_LOW | 13 | Bond too low. |
| TRANSFER_TYPE_BOND_HIGH | 14 | Bond too high. |



<a name="vega.Withdrawal.Status"></a>

### Withdrawal.Status
The status of the withdrawal

| Name | Number | Description |
| ---- | ------ | ----------- |
| WITHDRAWAL_STATUS_UNSPECIFIED | 0 | The default value |
| WITHDRAWAL_STATUS_OPEN | 1 | The withdrawal is open and being processed by the network |
| WITHDRAWAL_STATUS_CANCELLED | 2 | The withdrawal have been cancelled |
| WITHDRAWAL_STATUS_FINALIZED | 3 | The withdrawal went through and is fully finalized (funds remove from the vega network, and unlocked from the foreign chain bridge) |










## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

