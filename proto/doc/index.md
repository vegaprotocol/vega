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
    - [MarketsDataResponse](#api.MarketsDataResponse)
    - [MarketsDataSubscribeRequest](#api.MarketsDataSubscribeRequest)
    - [MarketsResponse](#api.MarketsResponse)
    - [ObservePartyProposalsRequest](#api.ObservePartyProposalsRequest)
    - [ObservePartyVotesRequest](#api.ObservePartyVotesRequest)
    - [ObserveProposalVotesRequest](#api.ObserveProposalVotesRequest)
    - [OptionalProposalState](#api.OptionalProposalState)
    - [OrderByIDRequest](#api.OrderByIDRequest)
    - [OrderByMarketAndIdRequest](#api.OrderByMarketAndIdRequest)
    - [OrderByMarketAndIdResponse](#api.OrderByMarketAndIdResponse)
    - [OrderByReferenceIDRequest](#api.OrderByReferenceIDRequest)
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
    - [PrepareProposalRequest](#api.PrepareProposalRequest)
    - [PrepareProposalResponse](#api.PrepareProposalResponse)
    - [PrepareSubmitOrderResponse](#api.PrepareSubmitOrderResponse)
    - [PrepareVoteRequest](#api.PrepareVoteRequest)
    - [PrepareVoteResponse](#api.PrepareVoteResponse)
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
    - [WithdrawRequest](#api.WithdrawRequest)
    - [WithdrawResponse](#api.WithdrawResponse)

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

- [proto/governance.proto](#proto/governance.proto)
    - [FeeFactorsConfiguration](#vega.FeeFactorsConfiguration)
    - [FutureProduct](#vega.FutureProduct)
    - [GovernanceData](#vega.GovernanceData)
    - [GovernanceData.NoPartyEntry](#vega.GovernanceData.NoPartyEntry)
    - [GovernanceData.YesPartyEntry](#vega.GovernanceData.YesPartyEntry)
    - [InstrumentConfiguration](#vega.InstrumentConfiguration)
    - [NetworkConfiguration](#vega.NetworkConfiguration)
    - [NewAsset](#vega.NewAsset)
    - [NewMarket](#vega.NewMarket)
    - [NewMarketConfiguration](#vega.NewMarketConfiguration)
    - [Proposal](#vega.Proposal)
    - [ProposalTerms](#vega.ProposalTerms)
    - [UpdateMarket](#vega.UpdateMarket)
    - [UpdateNetwork](#vega.UpdateNetwork)
    - [Vote](#vega.Vote)

    - [Proposal.State](#vega.Proposal.State)
    - [ProposalError](#vega.ProposalError)
    - [Vote.Value](#vega.Vote.Value)

- [proto/markets.proto](#proto/markets.proto)
    - [AuctionDuration](#vega.AuctionDuration)
    - [ContinuousTrading](#vega.ContinuousTrading)
    - [DiscreteTrading](#vega.DiscreteTrading)
    - [EthereumEvent](#vega.EthereumEvent)
    - [ExternalRiskModel](#vega.ExternalRiskModel)
    - [ExternalRiskModel.ConfigEntry](#vega.ExternalRiskModel.ConfigEntry)
    - [FeeFactors](#vega.FeeFactors)
    - [Fees](#vega.Fees)
    - [Future](#vega.Future)
    - [Instrument](#vega.Instrument)
    - [InstrumentMetadata](#vega.InstrumentMetadata)
    - [LogNormalModelParams](#vega.LogNormalModelParams)
    - [LogNormalRiskModel](#vega.LogNormalRiskModel)
    - [MarginCalculator](#vega.MarginCalculator)
    - [Market](#vega.Market)
    - [ScalingFactors](#vega.ScalingFactors)
    - [SimpleModelParams](#vega.SimpleModelParams)
    - [SimpleRiskModel](#vega.SimpleRiskModel)
    - [TradableInstrument](#vega.TradableInstrument)

- [proto/vega.proto](#proto/vega.proto)
    - [Account](#vega.Account)
    - [AuctionIndicativeState](#vega.AuctionIndicativeState)
    - [Candle](#vega.Candle)
    - [ErrorDetail](#vega.ErrorDetail)
    - [Fee](#vega.Fee)
    - [FinancialAmount](#vega.FinancialAmount)
    - [LedgerEntry](#vega.LedgerEntry)
    - [MarginLevels](#vega.MarginLevels)
    - [MarketData](#vega.MarketData)
    - [MarketDepth](#vega.MarketDepth)
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
    - [Position](#vega.Position)
    - [PositionTrade](#vega.PositionTrade)
    - [Price](#vega.Price)
    - [PriceLevel](#vega.PriceLevel)
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
    - [Withdraw](#vega.Withdraw)

    - [AccountType](#vega.AccountType)
    - [ChainStatus](#vega.ChainStatus)
    - [Interval](#vega.Interval)
    - [MarketState](#vega.MarketState)
    - [NodeSignatureKind](#vega.NodeSignatureKind)
    - [Order.Status](#vega.Order.Status)
    - [Order.TimeInForce](#vega.Order.TimeInForce)
    - [Order.Type](#vega.Order.Type)
    - [OrderError](#vega.OrderError)
    - [Side](#vega.Side)
    - [Trade.Type](#vega.Trade.Type)
    - [TransferType](#vega.TransferType)

- [Scalar Value Types](#scalar-value-types)



<a name="proto/api/trading.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/api/trading.proto



<a name="api.AccountsSubscribeRequest"></a>

### AccountsSubscribeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| partyID | [string](#string) |  |  |
| asset | [string](#string) |  |  |
| type | [vega.AccountType](#vega.AccountType) |  |  |






<a name="api.AmendOrderRequest"></a>

### AmendOrderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| amendment | [vega.OrderAmendment](#vega.OrderAmendment) |  |  |






<a name="api.AssetByIDRequest"></a>

### AssetByIDRequest
The request message to get an AssetByID


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | ID of the asset to get |






<a name="api.AssetByIDResponse"></a>

### AssetByIDResponse
The response message to get an AssetByID


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| asset | [vega.Asset](#vega.Asset) |  | The asset corresponding to the requested ID |






<a name="api.AssetsRequest"></a>

### AssetsRequest
The request to get the lit of all assets in vega






<a name="api.AssetsResponse"></a>

### AssetsResponse
The response containing the list of all assets enabled in vega


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| assets | [vega.Asset](#vega.Asset) | repeated | The list of assets |






<a name="api.CancelOrderRequest"></a>

### CancelOrderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cancellation | [vega.OrderCancellation](#vega.OrderCancellation) |  |  |






<a name="api.CandlesRequest"></a>

### CandlesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| sinceTimestamp | [int64](#int64) |  | nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| interval | [vega.Interval](#vega.Interval) |  |  |






<a name="api.CandlesResponse"></a>

### CandlesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| candles | [vega.Candle](#vega.Candle) | repeated |  |






<a name="api.CandlesSubscribeRequest"></a>

### CandlesSubscribeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| interval | [vega.Interval](#vega.Interval) |  |  |






<a name="api.FeeInfrastructureAccountsRequest"></a>

### FeeInfrastructureAccountsRequest
Request for the infrastructure fees accounts


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| asset | [string](#string) |  | an empty string to return all accounts an asset ID to return a single infrastructure fee fee account for a given asset |






<a name="api.FeeInfrastructureAccountsResponse"></a>

### FeeInfrastructureAccountsResponse
Response for the infrastructure fees accounts


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| accounts | [vega.Account](#vega.Account) | repeated | A list of infrastructure fee accounts for all or a specific asset |






<a name="api.GetNetworkParametersProposalsRequest"></a>

### GetNetworkParametersProposalsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  |  |






<a name="api.GetNetworkParametersProposalsResponse"></a>

### GetNetworkParametersProposalsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated |  |






<a name="api.GetNewAssetProposalsRequest"></a>

### GetNewAssetProposalsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  |  |






<a name="api.GetNewAssetProposalsResponse"></a>

### GetNewAssetProposalsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated |  |






<a name="api.GetNewMarketProposalsRequest"></a>

### GetNewMarketProposalsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  |  |






<a name="api.GetNewMarketProposalsResponse"></a>

### GetNewMarketProposalsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated |  |






<a name="api.GetNodeSignaturesAggregateRequest"></a>

### GetNodeSignaturesAggregateRequest
The request message to specify the ID of the resource we want to retrieve
the aggregated signatures for


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | The ID of the resource |






<a name="api.GetNodeSignaturesAggregateResponse"></a>

### GetNodeSignaturesAggregateResponse
The response of the GetNodeSIgnatureAggregate rpc


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| signatures | [vega.NodeSignature](#vega.NodeSignature) | repeated | The list of signatures |






<a name="api.GetProposalByIDRequest"></a>

### GetProposalByIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| proposalID | [string](#string) |  |  |






<a name="api.GetProposalByIDResponse"></a>

### GetProposalByIDResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) |  |  |






<a name="api.GetProposalByReferenceRequest"></a>

### GetProposalByReferenceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Reference | [string](#string) |  |  |






<a name="api.GetProposalByReferenceResponse"></a>

### GetProposalByReferenceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) |  |  |






<a name="api.GetProposalsByPartyRequest"></a>

### GetProposalsByPartyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  |  |






<a name="api.GetProposalsByPartyResponse"></a>

### GetProposalsByPartyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated |  |






<a name="api.GetProposalsRequest"></a>

### GetProposalsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  |  |






<a name="api.GetProposalsResponse"></a>

### GetProposalsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated |  |






<a name="api.GetUpdateMarketProposalsRequest"></a>

### GetUpdateMarketProposalsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| selectInState | [OptionalProposalState](#api.OptionalProposalState) |  |  |






<a name="api.GetUpdateMarketProposalsResponse"></a>

### GetUpdateMarketProposalsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [vega.GovernanceData](#vega.GovernanceData) | repeated |  |






<a name="api.GetVotesByPartyRequest"></a>

### GetVotesByPartyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |






<a name="api.GetVotesByPartyResponse"></a>

### GetVotesByPartyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| votes | [vega.Vote](#vega.Vote) | repeated |  |






<a name="api.LastTradeRequest"></a>

### LastTradeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |






<a name="api.LastTradeResponse"></a>

### LastTradeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trade | [vega.Trade](#vega.Trade) |  |  |






<a name="api.MarginLevelsRequest"></a>

### MarginLevelsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |
| marketID | [string](#string) |  |  |






<a name="api.MarginLevelsResponse"></a>

### MarginLevelsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marginLevels | [vega.MarginLevels](#vega.MarginLevels) | repeated |  |






<a name="api.MarginLevelsSubscribeRequest"></a>

### MarginLevelsSubscribeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |
| marketID | [string](#string) |  |  |






<a name="api.MarketAccountsRequest"></a>

### MarketAccountsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| asset | [string](#string) |  |  |






<a name="api.MarketAccountsResponse"></a>

### MarketAccountsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| accounts | [vega.Account](#vega.Account) | repeated |  |






<a name="api.MarketByIDRequest"></a>

### MarketByIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |






<a name="api.MarketByIDResponse"></a>

### MarketByIDResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| market | [vega.Market](#vega.Market) |  |  |






<a name="api.MarketDataByIDRequest"></a>

### MarketDataByIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |






<a name="api.MarketDataByIDResponse"></a>

### MarketDataByIDResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketData | [vega.MarketData](#vega.MarketData) |  |  |






<a name="api.MarketDepthRequest"></a>

### MarketDepthRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| maxDepth | [uint64](#uint64) |  |  |






<a name="api.MarketDepthResponse"></a>

### MarketDepthResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| buy | [vega.PriceLevel](#vega.PriceLevel) | repeated |  |
| sell | [vega.PriceLevel](#vega.PriceLevel) | repeated |  |
| lastTrade | [vega.Trade](#vega.Trade) |  |  |






<a name="api.MarketDepthSubscribeRequest"></a>

### MarketDepthSubscribeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |






<a name="api.MarketsDataResponse"></a>

### MarketsDataResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketsData | [vega.MarketData](#vega.MarketData) | repeated |  |






<a name="api.MarketsDataSubscribeRequest"></a>

### MarketsDataSubscribeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |






<a name="api.MarketsResponse"></a>

### MarketsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| markets | [vega.Market](#vega.Market) | repeated | a list of Markets |






<a name="api.ObservePartyProposalsRequest"></a>

### ObservePartyProposalsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |






<a name="api.ObservePartyVotesRequest"></a>

### ObservePartyVotesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |






<a name="api.ObserveProposalVotesRequest"></a>

### ObserveProposalVotesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| proposalID | [string](#string) |  |  |






<a name="api.OptionalProposalState"></a>

### OptionalProposalState



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [vega.Proposal.State](#vega.Proposal.State) |  |  |






<a name="api.OrderByIDRequest"></a>

### OrderByIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  |  |
| version | [uint64](#uint64) |  | version of the order (0 for most recent; 1 for original; 2 for first amendment, etc) |






<a name="api.OrderByMarketAndIdRequest"></a>

### OrderByMarketAndIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| orderID | [string](#string) |  |  |






<a name="api.OrderByMarketAndIdResponse"></a>

### OrderByMarketAndIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| order | [vega.Order](#vega.Order) |  |  |






<a name="api.OrderByReferenceIDRequest"></a>

### OrderByReferenceIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| referenceID | [string](#string) |  |  |






<a name="api.OrderByReferenceRequest"></a>

### OrderByReferenceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reference | [string](#string) |  |  |






<a name="api.OrderByReferenceResponse"></a>

### OrderByReferenceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| order | [vega.Order](#vega.Order) |  |  |






<a name="api.OrderVersionsByIDRequest"></a>

### OrderVersionsByIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  |  |
| pagination | [Pagination](#api.Pagination) |  |  |






<a name="api.OrderVersionsResponse"></a>

### OrderVersionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orders | [vega.Order](#vega.Order) | repeated |  |






<a name="api.OrdersByMarketRequest"></a>

### OrdersByMarketRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| pagination | [Pagination](#api.Pagination) |  |  |






<a name="api.OrdersByMarketResponse"></a>

### OrdersByMarketResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orders | [vega.Order](#vega.Order) | repeated |  |






<a name="api.OrdersByPartyRequest"></a>

### OrdersByPartyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |
| pagination | [Pagination](#api.Pagination) |  |  |






<a name="api.OrdersByPartyResponse"></a>

### OrdersByPartyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orders | [vega.Order](#vega.Order) | repeated |  |






<a name="api.OrdersStream"></a>

### OrdersStream



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orders | [vega.Order](#vega.Order) | repeated |  |






<a name="api.OrdersSubscribeRequest"></a>

### OrdersSubscribeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| partyID | [string](#string) |  |  |






<a name="api.Pagination"></a>

### Pagination



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| skip | [uint64](#uint64) |  |  |
| limit | [uint64](#uint64) |  |  |
| descending | [bool](#bool) |  |  |






<a name="api.PartiesResponse"></a>

### PartiesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parties | [vega.Party](#vega.Party) | repeated |  |






<a name="api.PartyAccountsRequest"></a>

### PartyAccountsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |
| marketID | [string](#string) |  |  |
| type | [vega.AccountType](#vega.AccountType) |  |  |
| asset | [string](#string) |  |  |






<a name="api.PartyAccountsResponse"></a>

### PartyAccountsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| accounts | [vega.Account](#vega.Account) | repeated |  |






<a name="api.PartyByIDRequest"></a>

### PartyByIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |






<a name="api.PartyByIDResponse"></a>

### PartyByIDResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| party | [vega.Party](#vega.Party) |  |  |






<a name="api.PositionsByPartyRequest"></a>

### PositionsByPartyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |
| marketID | [string](#string) |  |  |






<a name="api.PositionsByPartyResponse"></a>

### PositionsByPartyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| positions | [vega.Position](#vega.Position) | repeated |  |






<a name="api.PositionsSubscribeRequest"></a>

### PositionsSubscribeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |






<a name="api.PrepareAmendOrderResponse"></a>

### PrepareAmendOrderResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  |  |






<a name="api.PrepareCancelOrderResponse"></a>

### PrepareCancelOrderResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  |  |






<a name="api.PrepareProposalRequest"></a>

### PrepareProposalRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |
| reference | [string](#string) |  |  |
| proposal | [vega.ProposalTerms](#vega.ProposalTerms) |  |  |






<a name="api.PrepareProposalResponse"></a>

### PrepareProposalResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  |  |
| pendingProposal | [vega.Proposal](#vega.Proposal) |  |  |






<a name="api.PrepareSubmitOrderResponse"></a>

### PrepareSubmitOrderResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  |  |
| submitID | [string](#string) |  |  |






<a name="api.PrepareVoteRequest"></a>

### PrepareVoteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vote | [vega.Vote](#vega.Vote) |  |  |






<a name="api.PrepareVoteResponse"></a>

### PrepareVoteResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  |  |
| vote | [vega.Vote](#vega.Vote) |  |  |






<a name="api.PropagateChainEventRequest"></a>

### PropagateChainEventRequest
The request for a new event sent by the blockchain queue to be propagated into vega


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| evt | [vega.ChainEvent](#vega.ChainEvent) |  | The event |
| pubKey | [string](#string) |  |  |
| signature | [bytes](#bytes) |  |  |






<a name="api.PropagateChainEventResponse"></a>

### PropagateChainEventResponse
The response for a new event sent to vega


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  | Did the event get accepted by the node successfully |






<a name="api.SubmitOrderRequest"></a>

### SubmitOrderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| submission | [vega.OrderSubmission](#vega.OrderSubmission) |  | the bulk of the Order, including market, party, price, size, side, time in force, etc. |






<a name="api.SubmitTransactionRequest"></a>

### SubmitTransactionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tx | [vega.SignedBundle](#vega.SignedBundle) |  |  |






<a name="api.SubmitTransactionResponse"></a>

### SubmitTransactionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |






<a name="api.TradesByMarketRequest"></a>

### TradesByMarketRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| pagination | [Pagination](#api.Pagination) |  |  |






<a name="api.TradesByMarketResponse"></a>

### TradesByMarketResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trades | [vega.Trade](#vega.Trade) | repeated |  |






<a name="api.TradesByOrderRequest"></a>

### TradesByOrderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  |  |






<a name="api.TradesByOrderResponse"></a>

### TradesByOrderResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trades | [vega.Trade](#vega.Trade) | repeated |  |






<a name="api.TradesByPartyRequest"></a>

### TradesByPartyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |
| marketID | [string](#string) |  |  |
| pagination | [Pagination](#api.Pagination) |  |  |






<a name="api.TradesByPartyResponse"></a>

### TradesByPartyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trades | [vega.Trade](#vega.Trade) | repeated |  |






<a name="api.TradesStream"></a>

### TradesStream



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trades | [vega.Trade](#vega.Trade) | repeated |  |






<a name="api.TradesSubscribeRequest"></a>

### TradesSubscribeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| partyID | [string](#string) |  |  |






<a name="api.VegaTimeResponse"></a>

### VegaTimeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [int64](#int64) |  | nanoseconds since the epoch, for example `1580473859111222333` corresponds to `2020-01-31T12:30:59.111222333Z` |






<a name="api.WithdrawRequest"></a>

### WithdrawRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| withdraw | [vega.Withdraw](#vega.Withdraw) |  |  |






<a name="api.WithdrawResponse"></a>

### WithdrawResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |












<a name="api.trading"></a>

### trading


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| PrepareSubmitOrder | [SubmitOrderRequest](#api.SubmitOrderRequest) | [PrepareSubmitOrderResponse](#api.PrepareSubmitOrderResponse) | Prepare a submit order request |
| PrepareCancelOrder | [CancelOrderRequest](#api.CancelOrderRequest) | [PrepareCancelOrderResponse](#api.PrepareCancelOrderResponse) | Cancel an Order |
| PrepareAmendOrder | [AmendOrderRequest](#api.AmendOrderRequest) | [PrepareAmendOrderResponse](#api.PrepareAmendOrderResponse) | Amend an Order |
| Withdraw | [WithdrawRequest](#api.WithdrawRequest) | [WithdrawResponse](#api.WithdrawResponse) | Request withdrawal |
| SubmitTransaction | [SubmitTransactionRequest](#api.SubmitTransactionRequest) | [SubmitTransactionResponse](#api.SubmitTransactionResponse) | Submit a signed transaction |
| PrepareProposal | [PrepareProposalRequest](#api.PrepareProposalRequest) | [PrepareProposalResponse](#api.PrepareProposalResponse) | Prepare proposal that can be sent out to the chain (via SubmitTransaction) |
| PrepareVote | [PrepareVoteRequest](#api.PrepareVoteRequest) | [PrepareVoteResponse](#api.PrepareVoteResponse) | Prepare a vote to be put on the chain (via SubmitTransaction) |
| PropagateChainEvent | [PropagateChainEventRequest](#api.PropagateChainEventRequest) | [PropagateChainEventResponse](#api.PropagateChainEventResponse) | chain events |


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
| OrderByReferenceID | [OrderByReferenceIDRequest](#api.OrderByReferenceIDRequest) | [.vega.Order](#vega.Order) | Get a specific order by referenceID |
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
| Statistics | [.google.protobuf.Empty](#google.protobuf.Empty) | [.vega.Statistics](#vega.Statistics) | Get Statistics |
| GetVegaTime | [.google.protobuf.Empty](#google.protobuf.Empty) | [VegaTimeResponse](#api.VegaTimeResponse) | Get Time |
| AccountsSubscribe | [AccountsSubscribeRequest](#api.AccountsSubscribeRequest) | [.vega.Account](#vega.Account) stream | Subscribe to a stream of Accounts |
| CandlesSubscribe | [CandlesSubscribeRequest](#api.CandlesSubscribeRequest) | [.vega.Candle](#vega.Candle) stream | Subscribe to a stream of Candles |
| MarginLevelsSubscribe | [MarginLevelsSubscribeRequest](#api.MarginLevelsSubscribeRequest) | [.vega.MarginLevels](#vega.MarginLevels) stream | Subscribe to a stream of Margin Levels |
| MarketDepthSubscribe | [MarketDepthSubscribeRequest](#api.MarketDepthSubscribeRequest) | [.vega.MarketDepth](#vega.MarketDepth) stream | Subscribe to a stream of Market Depth |
| MarketsDataSubscribe | [MarketsDataSubscribeRequest](#api.MarketsDataSubscribeRequest) | [.vega.MarketData](#vega.MarketData) stream | Subscribe to a stream of Markets Data |
| OrdersSubscribe | [OrdersSubscribeRequest](#api.OrdersSubscribeRequest) | [OrdersStream](#api.OrdersStream) stream | Subscribe to a stream of Orders |
| PositionsSubscribe | [PositionsSubscribeRequest](#api.PositionsSubscribeRequest) | [.vega.Position](#vega.Position) stream | Subscribe to a stream of Positions |
| TradesSubscribe | [TradesSubscribeRequest](#api.TradesSubscribeRequest) | [TradesStream](#api.TradesStream) stream | Subscribe to a stream of Trades |
| TransferResponsesSubscribe | [.google.protobuf.Empty](#google.protobuf.Empty) | [.vega.TransferResponse](#vega.TransferResponse) stream | Subscribe to a stream of Transfer Responses |
| GetNodeSignaturesAggregate | [GetNodeSignaturesAggregateRequest](#api.GetNodeSignaturesAggregateRequest) | [GetNodeSignaturesAggregateResponse](#api.GetNodeSignaturesAggregateResponse) | Get an aggregate of signature from all the node of the network |
| AssetByID | [AssetByIDRequest](#api.AssetByIDRequest) | [AssetByIDResponse](#api.AssetByIDResponse) | Get an asset by its ID |
| Assets | [AssetsRequest](#api.AssetsRequest) | [AssetsResponse](#api.AssetsResponse) | Get the list of all assets in vega |





<a name="proto/assets.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/assets.proto



<a name="vega.Asset"></a>

### Asset
The vega representation of an external asset


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | The vega internal ID of the asset |
| name | [string](#string) |  | The name of the asset (e.g: Great British Pound) |
| symbol | [string](#string) |  | The symbol of the asset (e.g: GBP) |
| totalSupply | [string](#string) |  | The total circulating supply for the asset |
| decimals | [uint64](#uint64) |  | The number of decimal / precision handled by this asset |
| source | [AssetSource](#vega.AssetSource) |  | The definition of the external source for this asset |






<a name="vega.AssetSource"></a>

### AssetSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| builtinAsset | [BuiltinAsset](#vega.BuiltinAsset) |  |  |
| erc20 | [ERC20](#vega.ERC20) |  |  |






<a name="vega.BuiltinAsset"></a>

### BuiltinAsset
A vega internal asset


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the asset (e.g: Great British Pound) |
| symbol | [string](#string) |  | The symbol of the asset (e.g: GBP) |
| totalSupply | [string](#string) |  | The total circulating supply for the asset |
| decimals | [uint64](#uint64) |  | The number of decimal / precision handled by this asset |
| maxFaucetAmountMint | [string](#string) |  | This is the maximum amount that can be requested by a party through the builtin asset faucet at a time |






<a name="vega.DevAssets"></a>

### DevAssets



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sources | [AssetSource](#vega.AssetSource) | repeated |  |






<a name="vega.ERC20"></a>

### ERC20
An ERC20 token based asset, living on the ethereum network


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contractAddress | [string](#string) |  | The address of the contract for the token, on the ethereum network |















<a name="proto/chain_events.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/chain_events.proto



<a name="vega.AddValidator"></a>

### AddValidator
A message to notify a new validator being added to the vega network


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [Identifier](#vega.Identifier) |  | The identifier of this validator |






<a name="vega.BTCDeposit"></a>

### BTCDeposit
A Bitcoin deposit into vega


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The vega network internally ID of the asset |
| sourceBTCAddress | [string](#string) |  | The BTC wallet inititing the Deposit |
| targetPartyId | [string](#string) |  | The Vega public key of the target Vega user |






<a name="vega.BTCEvent"></a>

### BTCEvent
An event from Bitcoin


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [uint64](#uint64) |  | The index of the transaction |
| block | [uint64](#uint64) |  | The block in which the transaction happenned |
| deposit | [BTCDeposit](#vega.BTCDeposit) |  |  |
| withdrawal | [BTCWithdrawal](#vega.BTCWithdrawal) |  |  |






<a name="vega.BTCWithdrawal"></a>

### BTCWithdrawal
A Bitcoin withdrawl from vega


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The vega network internally ID of the asset |
| sourcePartyId | [string](#string) |  | The party inititing the withdrawal |
| targetBTCAddress | [string](#string) |  | Target BTC wallet address |
| referenceNonce | [string](#string) |  | The nonce reference of the transaction |






<a name="vega.BitcoinAddress"></a>

### BitcoinAddress
Wrapper for a Bitcoin address (wallet)


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) |  | A Bitcoin address |






<a name="vega.BuiltinAssetDeposit"></a>

### BuiltinAssetDeposit
A deposit for an vega builtin asset


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | A vega network internal asset ID |
| partyID | [string](#string) |  | A vega party ID (pubkey) |
| amount | [uint64](#uint64) |  | The amount to be deposited |






<a name="vega.BuiltinAssetEvent"></a>

### BuiltinAssetEvent
An event related to a vega builtin asset


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| deposit | [BuiltinAssetDeposit](#vega.BuiltinAssetDeposit) |  |  |
| withdrawal | [BuiltinAssetWithdrawal](#vega.BuiltinAssetWithdrawal) |  |  |






<a name="vega.BuiltinAssetWithdrawal"></a>

### BuiltinAssetWithdrawal
A Withdrawal for a vega builtin asset


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | A vega network internal asset ID |
| partyID | [string](#string) |  | A vega network party ID (pubkey) |
| amount | [uint64](#uint64) |  | The amount to be withdrawan |






<a name="vega.ChainEvent"></a>

### ChainEvent
An event being forwarded to the vega network
providing information on things happening on other networks


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| txID | [string](#string) |  | The ID of the transaction in which the things happened usually a hash |
| nonce | [uint64](#uint64) |  | Arbitrary one-time integer used to prevent replay attacks |
| builtin | [BuiltinAssetEvent](#vega.BuiltinAssetEvent) |  |  |
| erc20 | [ERC20Event](#vega.ERC20Event) |  |  |
| btc | [BTCEvent](#vega.BTCEvent) |  |  |
| validator | [ValidatorEvent](#vega.ValidatorEvent) |  |  |






<a name="vega.ERC20AssetDelist"></a>

### ERC20AssetDelist
An asset blacklisting for a erc20 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The vega network internally ID of the asset |






<a name="vega.ERC20AssetList"></a>

### ERC20AssetList
An asset whitelisting for a erc20 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The vega network internally ID of the asset |






<a name="vega.ERC20Deposit"></a>

### ERC20Deposit
An asset deposit for an erc20 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The vega network internally ID of the asset |
| sourceEthereumAddress | [string](#string) |  | The ethereum wallet that initiated the deposit |
| targetPartyID | [string](#string) |  | The Vega public key of the target vega user |






<a name="vega.ERC20Event"></a>

### ERC20Event
An event related to an erc20 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [uint64](#uint64) |  | Index of the transaction |
| block | [uint64](#uint64) |  | The block in which the transaction was added |
| assetList | [ERC20AssetList](#vega.ERC20AssetList) |  |  |
| assetDelist | [ERC20AssetDelist](#vega.ERC20AssetDelist) |  |  |
| deposit | [ERC20Deposit](#vega.ERC20Deposit) |  |  |
| withdrawal | [ERC20Withdrawal](#vega.ERC20Withdrawal) |  |  |






<a name="vega.ERC20Withdrawal"></a>

### ERC20Withdrawal
An asset withdrawal for an erc20 token


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vegaAssetID | [string](#string) |  | The vega network internally ID of the asset |
| sourcePartyId | [string](#string) |  | The party inititing the withdrawal |
| targetEthereumAddress | [string](#string) |  | The target Ethereum wallet address |
| referenceNonce | [string](#string) |  | The reference nonce used for the transaction |






<a name="vega.EthereumAddress"></a>

### EthereumAddress
Wrapper for an Ethereum address (wallet/contract)


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [string](#string) |  | An Ethereum address |






<a name="vega.Identifier"></a>

### Identifier
A wrapper type on any possible network address supported by vega


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ethereumAddress | [EthereumAddress](#vega.EthereumAddress) |  |  |
| bitcoinAddress | [BitcoinAddress](#vega.BitcoinAddress) |  |  |






<a name="vega.RemoveValidator"></a>

### RemoveValidator
A message to notify a new validator being removed to the vega network


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [Identifier](#vega.Identifier) |  | The identifier of this validator |






<a name="vega.ValidatorEvent"></a>

### ValidatorEvent
An event related to validator management with foreign networks


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sourceID | [string](#string) |  | The source ID of the event |
| add | [AddValidator](#vega.AddValidator) |  |  |
| rm | [RemoveValidator](#vega.RemoveValidator) |  |  |















<a name="proto/governance.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/governance.proto



<a name="vega.FeeFactorsConfiguration"></a>

### FeeFactorsConfiguration
FeeFactors set at the network level


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| infrastructureFee | [string](#string) |  | the infrastructure fee, needs to be a valid float |
| makerFee | [string](#string) |  | the maker fee, needs to be a valid float |
| liquidityFee | [string](#string) |  | this is the liquidity fee, it needs to be a valid float |






<a name="vega.FutureProduct"></a>

### FutureProduct
Future product configuration


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maturity | [string](#string) |  | Future product maturity (ISO8601/RFC3339 timestamp) |
| asset | [string](#string) |  | Product asset name |






<a name="vega.GovernanceData"></a>

### GovernanceData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| proposal | [Proposal](#vega.Proposal) |  | Proposal |
| yes | [Vote](#vega.Vote) | repeated | All &#34;yes&#34; votes in favour of the proposal above. |
| no | [Vote](#vega.Vote) | repeated | All &#34;no&#34; votes against the proposal above. |
| yesParty | [GovernanceData.YesPartyEntry](#vega.GovernanceData.YesPartyEntry) | repeated | All latest YES votes by party (guaranteed to be unique) |
| noParty | [GovernanceData.NoPartyEntry](#vega.GovernanceData.NoPartyEntry) | repeated | All latest NO votes by party (unique) |






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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Instrument name |
| code | [string](#string) |  | Instrument code |
| baseName | [string](#string) |  | Base security used as the reference |
| quoteName | [string](#string) |  | Quote (secondary) security |
| future | [FutureProduct](#vega.FutureProduct) |  |  |






<a name="vega.NetworkConfiguration"></a>

### NetworkConfiguration



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| minCloseInSeconds | [int64](#int64) |  | Constrains minimum duration since submission (in seconds) when vote closing time is allowed to be set for a proposal. |
| maxCloseInSeconds | [int64](#int64) |  | Constrains maximum duration since submission (in seconds) when vote closing time is allowed to be set for a proposal. |
| minEnactInSeconds | [int64](#int64) |  | Constrains minimum duration since submission (in seconds) when enactment is allowed to be set for a proposal. |
| maxEnactInSeconds | [int64](#int64) |  | Constrains maximum duration since submission (in seconds) when enactment is allowed to be set for a proposal. |
| requiredParticipation | [float](#float) |  | Participation level required for any proposal to pass. Value from `0` to `1`. |
| requiredMajority | [float](#float) |  | Majority level required for any proposal to pass. Value from `0.5` to `1`. |
| minProposerBalance | [float](#float) |  | Minimum balance required for a party to be able to submit a new proposal. Value greater than `0` to `1`. |
| minVoterBalance | [float](#float) |  | Minimum balance required for a party to be able to cast a vote. Value greater than `0` to `1`. |
| marginConfiguration | [ScalingFactors](#vega.ScalingFactors) |  | Scaling factors for all markets created via governance. |
| feeFactorsConfiguration | [FeeFactorsConfiguration](#vega.FeeFactorsConfiguration) |  | FeeFactors which are not set via proposal |






<a name="vega.NewAsset"></a>

### NewAsset
To be implemented


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changes | [AssetSource](#vega.AssetSource) |  |  |






<a name="vega.NewMarket"></a>

### NewMarket



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changes | [NewMarketConfiguration](#vega.NewMarketConfiguration) |  |  |






<a name="vega.NewMarketConfiguration"></a>

### NewMarketConfiguration



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instrument | [InstrumentConfiguration](#vega.InstrumentConfiguration) |  | New market instrument configuration |
| decimalPlaces | [uint64](#uint64) |  | Decimal places used for the new market |
| metadata | [string](#string) | repeated | Optional new market meta data, tags |
| openingAuctionDuration | [int64](#int64) |  | for now, just specify a time for the opening auction to last |
| simple | [SimpleModelParams](#vega.SimpleModelParams) |  | Simple risk model parameters, valid only if MODEL_SIMPLE is selected |
| logNormal | [LogNormalRiskModel](#vega.LogNormalRiskModel) |  | Log normal risk model parameters, valid only if MODEL_LOG_NORMAL is selected |
| continuous | [ContinuousTrading](#vega.ContinuousTrading) |  |  |
| discrete | [DiscreteTrading](#vega.DiscreteTrading) |  |  |






<a name="vega.Proposal"></a>

### Proposal



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  | Proposal unique identifier. |
| reference | [string](#string) |  | Proposal reference. |
| partyID | [string](#string) |  | Proposal author, identifier of the party submitting the proposal. |
| state | [Proposal.State](#vega.Proposal.State) |  | Proposal state (see Proposal.State definition) |
| timestamp | [int64](#int64) |  | Proposal timestamp for date and time (in nanoseconds) when proposal was submitted to the network. |
| terms | [ProposalTerms](#vega.ProposalTerms) |  | Proposal configuration and the actual change that is meant to be executed when proposal is enacted. |
| reason | [ProposalError](#vega.ProposalError) |  | A reason for the current state of the proposal this may be set in case of REJECTED and FAILED status |






<a name="vega.ProposalTerms"></a>

### ProposalTerms



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| closingTimestamp | [int64](#int64) |  | Timestamp (Unix time in seconds) when voting closes for this proposal. Constrained by `minCloseInSeconds` and `maxCloseInSeconds` network parameters. |
| enactmentTimestamp | [int64](#int64) |  | Timestamp (Unix time in seconds) when proposal gets enacted (if passed). Constrained by `minEnactInSeconds` and `maxEnactInSeconds` network parameters. |
| validationTimestamp | [int64](#int64) |  | TODO: this should be moved into `NewAsset` definition. |
| updateMarket | [UpdateMarket](#vega.UpdateMarket) |  | Proposal change for modifying an existing market on Vega. |
| newMarket | [NewMarket](#vega.NewMarket) |  | Proposal change for creating new market on Vega. |
| updateNetwork | [UpdateNetwork](#vega.UpdateNetwork) |  | Proposal change for updating Vega network parameters. |
| newAsset | [NewAsset](#vega.NewAsset) |  | Proposal change for creating new assets on Vega. |






<a name="vega.UpdateMarket"></a>

### UpdateMarket
TODO






<a name="vega.UpdateNetwork"></a>

### UpdateNetwork



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changes | [NetworkConfiguration](#vega.NetworkConfiguration) |  |  |






<a name="vega.Vote"></a>

### Vote



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
| STATE_FAILED | 1 | Proposal enactment has failed - even though proposal has passed, its execusion could not be performed. |
| STATE_OPEN | 2 | Proposal is open for voting. |
| STATE_PASSED | 3 | Proposal has gained enough support to be executed. |
| STATE_REJECTED | 4 | Proposal wasn&#39;t accepted (proposal terms failed validation due to wrong configuration or failing to meet network requirements). |
| STATE_DECLINED | 5 | Proposal didn&#39;t get enough votes (either failing to gain required participation or majority level). |
| STATE_ENACTED | 6 |  |
| STATE_WAITING_FOR_NODE_VOTE | 7 | waiting for validators validation of the proposal |



<a name="vega.ProposalError"></a>

### ProposalError
A list of possible error which could have happenned
and the cause for an proposal being rejected of failed

| Name | Number | Description |
| ---- | ------ | ----------- |
| PROPOSAL_ERROR_UNSPECIFIED | 0 | default value |
| PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON | 1 | the specified close time is too early base on network parameters |
| PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE | 2 | the specified close time is too late based on network parameters |
| PROPOSAL_ERROR_ENACT_TIME_TOO_SOON | 3 | the specified enact time is too early base on network parameters |
| PROPOSAL_ERROR_ENACT_TIME_TOO_LATE | 4 | the specified enact time is too late based on network parameters |
| PROPOSAL_ERROR_INSUFFICIENT_TOKENS | 5 | the proposer for this proposal as insufficient token |
| PROPOSAL_ERROR_INVALID_INSTRUMENT_SECURITY | 6 | the instrument quote name and base name were the same |
| PROPOSAL_ERROR_NO_PRODUCT | 7 | the proposal has not product |
| PROPOSAL_ERROR_UNSUPPORTED_PRODUCT | 8 | the specified product is not supported |
| PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT_TIMESTAMP | 9 | invalid future maturity timestamp (expect RFC3339) |
| PROPOSAL_ERROR_PRODUCT_MATURITY_IS_PASSED | 10 | the product maturity is past |
| PROPOSAL_ERROR_NO_TRADING_MODE | 11 | the proposal has not trading mode |
| PROPOSAL_ERROR_UNSUPPORTED_TRADING_MODE | 12 | the proposal has an unsupported trading mode |
| PROPOSAL_ERROR_NODE_VALIDATION_FAILED | 13 | the proposal failed node validation |
| PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD | 14 | a field is missing in a builtin asset source |
| PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS | 15 | the contract address is missing in the ERC20 asset source |
| PROPOSAL_ERROR_INVALID_ASSET | 16 | the asset id refer to no assets in vega |



<a name="vega.Vote.Value"></a>

### Vote.Value


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
AuctionDuration can be used to configure 3 auction periods:
1) duration &gt; 0, volume == 0: The auction will last for at least N seconds
2) Duration == 0, volume &gt; 0: Auction period will end once we can close with given traded volume
3) Duration &gt; 0 &amp; volume &gt; 0: Auction period will take at least N seconds, but can end sooner if we can trade a certain volume


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| duration | [int64](#int64) |  |  |
| volume | [uint64](#uint64) |  |  |






<a name="vega.ContinuousTrading"></a>

### ContinuousTrading



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tickSize | [string](#string) |  |  |






<a name="vega.DiscreteTrading"></a>

### DiscreteTrading



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| durationNs | [int64](#int64) |  | Duration in nanoseconds, maximum 1 month (2592000000000000 ns) |
| tickSize | [string](#string) |  |  |






<a name="vega.EthereumEvent"></a>

### EthereumEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contractID | [string](#string) |  |  |
| event | [string](#string) |  |  |
| value | [uint64](#uint64) |  |  |






<a name="vega.ExternalRiskModel"></a>

### ExternalRiskModel



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| socket | [string](#string) |  |  |
| config | [ExternalRiskModel.ConfigEntry](#vega.ExternalRiskModel.ConfigEntry) | repeated |  |






<a name="vega.ExternalRiskModel.ConfigEntry"></a>

### ExternalRiskModel.ConfigEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="vega.FeeFactors"></a>

### FeeFactors



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| makerFee | [string](#string) |  |  |
| infrastructureFee | [string](#string) |  |  |
| liquidityFee | [string](#string) |  |  |






<a name="vega.Fees"></a>

### Fees



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| factors | [FeeFactors](#vega.FeeFactors) |  |  |






<a name="vega.Future"></a>

### Future



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maturity | [string](#string) |  |  |
| asset | [string](#string) |  |  |
| ethereumEvent | [EthereumEvent](#vega.EthereumEvent) |  |  |






<a name="vega.Instrument"></a>

### Instrument



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| code | [string](#string) |  |  |
| name | [string](#string) |  |  |
| baseName | [string](#string) |  |  |
| quoteName | [string](#string) |  |  |
| metadata | [InstrumentMetadata](#vega.InstrumentMetadata) |  |  |
| initialMarkPrice | [uint64](#uint64) |  |  |
| future | [Future](#vega.Future) |  |  |






<a name="vega.InstrumentMetadata"></a>

### InstrumentMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tags | [string](#string) | repeated |  |






<a name="vega.LogNormalModelParams"></a>

### LogNormalModelParams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| mu | [double](#double) |  |  |
| r | [double](#double) |  |  |
| sigma | [double](#double) |  |  |






<a name="vega.LogNormalRiskModel"></a>

### LogNormalRiskModel



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| riskAversionParameter | [double](#double) |  |  |
| tau | [double](#double) |  |  |
| params | [LogNormalModelParams](#vega.LogNormalModelParams) |  |  |






<a name="vega.MarginCalculator"></a>

### MarginCalculator



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| scalingFactors | [ScalingFactors](#vega.ScalingFactors) |  |  |






<a name="vega.Market"></a>

### Market



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| tradableInstrument | [TradableInstrument](#vega.TradableInstrument) |  |  |
| decimalPlaces | [uint64](#uint64) |  | the number of decimal places that a price must be shifted by in order to get a correct price denominated in the currency of the Market. ie `realPrice = price / 10^decimalPlaces` |
| fees | [Fees](#vega.Fees) |  | fees configuration |
| openingAuction | [AuctionDuration](#vega.AuctionDuration) |  | Specifies how long the opening auction will run (min duration &#43; optionally minimum traded volume) |
| continuous | [ContinuousTrading](#vega.ContinuousTrading) |  |  |
| discrete | [DiscreteTrading](#vega.DiscreteTrading) |  |  |






<a name="vega.ScalingFactors"></a>

### ScalingFactors



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| searchLevel | [double](#double) |  |  |
| initialMargin | [double](#double) |  |  |
| collateralRelease | [double](#double) |  |  |






<a name="vega.SimpleModelParams"></a>

### SimpleModelParams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| factorLong | [double](#double) |  |  |
| factorShort | [double](#double) |  |  |






<a name="vega.SimpleRiskModel"></a>

### SimpleRiskModel



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | [SimpleModelParams](#vega.SimpleModelParams) |  |  |






<a name="vega.TradableInstrument"></a>

### TradableInstrument



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instrument | [Instrument](#vega.Instrument) |  |  |
| marginCalculator | [MarginCalculator](#vega.MarginCalculator) |  |  |
| logNormalRiskModel | [LogNormalRiskModel](#vega.LogNormalRiskModel) |  |  |
| externalRiskModel | [ExternalRiskModel](#vega.ExternalRiskModel) |  |  |
| simpleRiskModel | [SimpleRiskModel](#vega.SimpleRiskModel) |  |  |















<a name="proto/vega.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/vega.proto



<a name="vega.Account"></a>

### Account
Represents an account for an asset on Vega for a particular owner or party.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Unique identifier (used internally by Vega). |
| owner | [string](#string) |  | The party that the account belongs to. Special values include `network`, which represents the network and is most commonly seen during liquidation of distressed trading positions. |
| balance | [uint64](#uint64) |  | Balance of the asset, the balance is an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. Balances cannot be negative. |
| asset | [string](#string) |  | Asset identifier for the account. |
| marketID | [string](#string) |  | Market identifier for the account. If [`AccountType`](#vega.AccountType).`ACCOUNT_TYPE_GENERAL` this will be empty. |
| type | [AccountType](#vega.AccountType) |  | The account type related to this account. |






<a name="vega.AuctionIndicativeState"></a>

### AuctionIndicativeState
Whenever a change to the book occurs during an auction, this message will be used
to emit an event with the indicative price/volume per market


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






<a name="vega.ErrorDetail"></a>

### ErrorDetail
Represents Vega domain specific error information over gRPC/Protobuf.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [int32](#int32) |  | a Vega API domain specific unique error code, useful for client side mappings. e.g. 10004 |
| message | [string](#string) |  | a message that describes the error in more detail, should describe the problem encountered. |
| inner | [string](#string) |  | any inner error information that could add more context, or be helpful for error reporting. |






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
| midPrice | [uint64](#uint64) |  | Arithmetic average of the best bid price and best offer price, as an integer, for example `123456` is a correctly // formatted price of `1.23456` assuming market configured to 5 decimal places. |
| market | [string](#string) |  | Market identifier for the data. |
| timestamp | [int64](#int64) |  | Timestamp at which this mark price was relevant, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| openInterest | [uint64](#uint64) |  | The sum of the size of all positions greater than 0 on the market. |
| auctionEnd | [int64](#int64) |  | Time in seconds until the end of the auction (0 if currently not in auction period). |
| auctionStart | [int64](#int64) |  | Time until next auction (used in FBA&#39;s) - currently always 0. |






<a name="vega.MarketDepth"></a>

### MarketDepth
Represents market depth or order book data for the specified market on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  | Market identifier. |
| buy | [PriceLevel](#vega.PriceLevel) | repeated | Collection of price levels for the buy side of the book. |
| sell | [PriceLevel](#vega.PriceLevel) | repeated | Collection of price levels for the sell side of the book. |






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
| reference | [string](#string) |  | Reference given for the order, this is typically used to retrieve an order submitted through consensus. Currently set internally by the node to return a unique reference identifier for the order submission. TODO(cdm): Section on how order references work on Vega in docs.vega.xyz |
| reason | [OrderError](#vega.OrderError) |  | If the Order `status` is `STATUS_REJECTED` then an [`OrderError`](#vega.OrderError) reason will be specified. The default for this field is `ORDER_ERROR_NONE` which signifies that there were no errors. |
| updatedAt | [int64](#int64) |  | Timestamp for when the Order was last updated, in nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| version | [uint64](#uint64) |  | The version for the order, initial value is version 1 and is incremented after each successful amend |
| batchID | [uint64](#uint64) |  | Batch identifier for the order, used internally for orders submitted during auctions to keep track of the auction batch this order falls under (required for fees calculation). |






<a name="vega.OrderAmendment"></a>

### OrderAmendment
An order amendment is a request to amend or update an existing order on Vega.

The following three fields are used for lookup of the order only, cannot be amended by this command:


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  | Order identifier, this is required to find the order and will not be updated. Required field. |
| partyID | [string](#string) |  | Party identifier, this is required to find the order and will not be updated. Required field. |
| marketID | [string](#string) |  | Market identifier, this is required to find the order and will not be updated. |
| price | [Price](#vega.Price) |  | Amend the price for the order, if the Price value is set, otherwise price will remain unchanged. See [`Price`](#vega.Price). |
| sizeDelta | [int64](#int64) |  | Amend the size for the order by the delta specified. To reduce the size from the current value set a negative integer value. To increase the size from the current value, set a positive integer value. To leave the size unchanged set a value of zero. |
| expiresAt | [Timestamp](#vega.Timestamp) |  | Amend the expiry time for the order, if the Timestamp value is set, otherwise expiry time will remain unchanged. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| timeInForce | [Order.TimeInForce](#vega.Order.TimeInForce) |  | Amend the time in force for the order, set to TIF_UNSPECIFIED to remain unchanged. See [`TimeInForce`](#api.VegaTimeResponse).`timestamp`. |






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
| reference | [string](#string) |  | Reference given for the order, this is typically used to retrieve an order submitted through consensus. Currently set internally by the node to return a unique reference identifier for the order submission. TODO(cdm): Section on how order references work on Vega in docs.vega.xyz |






<a name="vega.Party"></a>

### Party
A party represents an entity who wishes to trade on or query a Vega network.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | A unique identifier for the party, typically represented by a public key. |






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
| cumulativeVolume | [uint64](#uint64) |  | Cumulative volume at the price level. |






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
A set of one or more trades.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trades | [Trade](#vega.Trade) | repeated |  |






<a name="vega.Transaction"></a>

### Transaction
Represents a transaction to be sent to Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| inputData | [bytes](#bytes) |  | One of the set of Vega commands (proto marshalled). |
| nonce | [uint64](#uint64) |  | A random number used to provided uniqueness and prevents against replay attack. |
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






<a name="vega.Withdraw"></a>

### Withdraw
Represents a withdrawal of an asset by a party on Vega.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  | Unique party identifier affecting the withdrawal. |
| amount | [uint64](#uint64) |  | The total amount withdrawn, the amount is an integer, for example `123456` is a correctly formatted price of `1.23456` assuming market configured to 5 decimal places. |
| asset | [string](#string) |  | Asset identifier. |








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



<a name="vega.ChainStatus"></a>

### ChainStatus
The Vega blockchain status as reported by the node the caller is connected to.

| Name | Number | Description |
| ---- | ------ | ----------- |
| CHAIN_STATUS_UNSPECIFIED | 0 | Default value, always invalid. |
| CHAIN_STATUS_DISCONNECTED | 1 | Blockchain is disconnected. |
| CHAIN_STATUS_REPLAYING | 2 | Blockchain is replaying historic transactions. |
| CHAIN_STATUS_CONNECTED | 3 | Blockchain is connected and receiving transactions. |



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



<a name="vega.MarketState"></a>

### MarketState
What mode is the market currently running, also known as market state.

| Name | Number | Description |
| ---- | ------ | ----------- |
| MARKET_STATE_UNSPECIFIED | 0 | Default value, this is invalid |
| MARKET_STATE_CONTINUOUS | 1 | Normal trading |
| MARKET_STATE_AUCTION | 2 | Auction trading |



<a name="vega.NodeSignatureKind"></a>

### NodeSignatureKind
The kind of the signature created by a node, for example, whitelisting a new asset, withdrawal etc.

| Name | Number | Description |
| ---- | ------ | ----------- |
| NODE_SIGNATURE_KIND_UNSPECIFIED | 0 | represents a unspecified / missing value from the input |
| NODE_SIGNATURE_KIND_ASSET_NEW | 1 | represents a signature for a new asset whitelisting |
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
If there is an issue with an order during it&#39;s life-cycle, it will be marked with `status.ORDER_STATUS_REJECTED`
and be given an error code in the `reason` field.

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
| TYPE_NETWORK_CLOSE_OUT_BAD | 3 | Trading initiated by the network with another party off the book, with a distressed party in order to zero-out the position of the party. todo(cdm): chat with Jeremy on zoom to sanity check/improve. |



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

