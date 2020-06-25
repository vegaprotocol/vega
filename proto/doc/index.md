# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [proto/api/trading.proto](#proto/api/trading.proto)
    - [AccountsSubscribeRequest](#api.AccountsSubscribeRequest)
    - [AmendOrderRequest](#api.AmendOrderRequest)
    - [CancelOrderRequest](#api.CancelOrderRequest)
    - [CandlesRequest](#api.CandlesRequest)
    - [CandlesResponse](#api.CandlesResponse)
    - [CandlesSubscribeRequest](#api.CandlesSubscribeRequest)
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

    - [trading](#api.trading)
    - [trading_data](#api.trading_data)

- [proto/assets.proto](#proto/assets.proto)
    - [Asset](#vega.Asset)
    - [AssetSource](#vega.AssetSource)
    - [BuiltinAsset](#vega.BuiltinAsset)
    - [DevAssets](#vega.DevAssets)
    - [ERC20](#vega.ERC20)

- [proto/governance.proto](#proto/governance.proto)
    - [GovernanceData](#vega.GovernanceData)
    - [GovernanceData.NoPartyEntry](#vega.GovernanceData.NoPartyEntry)
    - [GovernanceData.YesPartyEntry](#vega.GovernanceData.YesPartyEntry)
    - [NetworkConfiguration](#vega.NetworkConfiguration)
    - [NewAsset](#vega.NewAsset)
    - [NewMarket](#vega.NewMarket)
    - [Proposal](#vega.Proposal)
    - [ProposalTerms](#vega.ProposalTerms)
    - [UpdateMarket](#vega.UpdateMarket)
    - [UpdateNetwork](#vega.UpdateNetwork)
    - [Vote](#vega.Vote)

    - [Proposal.State](#vega.Proposal.State)
    - [Vote.Value](#vega.Vote.Value)

- [proto/markets.proto](#proto/markets.proto)
    - [ContinuousTrading](#vega.ContinuousTrading)
    - [DiscreteTrading](#vega.DiscreteTrading)
    - [EthereumEvent](#vega.EthereumEvent)
    - [ExternalRiskModel](#vega.ExternalRiskModel)
    - [ExternalRiskModel.ConfigEntry](#vega.ExternalRiskModel.ConfigEntry)
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
    - [Amount](#vega.Amount)
    - [Candle](#vega.Candle)
    - [ErrorDetail](#vega.ErrorDetail)
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
    - [SignedBundle](#vega.SignedBundle)
    - [Statistics](#vega.Statistics)
    - [Timestamp](#vega.Timestamp)
    - [Trade](#vega.Trade)
    - [TradeSet](#vega.TradeSet)
    - [Transfer](#vega.Transfer)
    - [TransferBalance](#vega.TransferBalance)
    - [TransferRequest](#vega.TransferRequest)
    - [TransferResponse](#vega.TransferResponse)

    - [AccountType](#vega.AccountType)
    - [ChainStatus](#vega.ChainStatus)
    - [Interval](#vega.Interval)
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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  |  |






<a name="api.GetNodeSignaturesAggregateResponse"></a>

### GetNodeSignaturesAggregateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| signatures | [vega.NodeSignature](#vega.NodeSignature) | repeated |  |






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
| open | [bool](#bool) |  |  |






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
| open | [bool](#bool) |  |  |






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












<a name="api.trading"></a>

### trading


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| PrepareSubmitOrder | [SubmitOrderRequest](#api.SubmitOrderRequest) | [PrepareSubmitOrderResponse](#api.PrepareSubmitOrderResponse) | Prepare a submit order request |
| PrepareCancelOrder | [CancelOrderRequest](#api.CancelOrderRequest) | [PrepareCancelOrderResponse](#api.PrepareCancelOrderResponse) | Cancel an Order |
| PrepareAmendOrder | [AmendOrderRequest](#api.AmendOrderRequest) | [PrepareAmendOrderResponse](#api.PrepareAmendOrderResponse) | Amend an Order |
| SubmitTransaction | [SubmitTransactionRequest](#api.SubmitTransactionRequest) | [SubmitTransactionResponse](#api.SubmitTransactionResponse) | Submit a signed transaction |
| PrepareProposal | [PrepareProposalRequest](#api.PrepareProposalRequest) | [PrepareProposalResponse](#api.PrepareProposalResponse) | Prepare proposal that can be sent out to the chain (via SubmitTransaction) |
| PrepareVote | [PrepareVoteRequest](#api.PrepareVoteRequest) | [PrepareVoteResponse](#api.PrepareVoteResponse) | Prepare a vote to be put on the chain (via SubmitTransaction) |


<a name="api.trading_data"></a>

### trading_data


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| MarketAccounts | [MarketAccountsRequest](#api.MarketAccountsRequest) | [MarketAccountsResponse](#api.MarketAccountsResponse) | Get a list of Accounts by Market |
| PartyAccounts | [PartyAccountsRequest](#api.PartyAccountsRequest) | [PartyAccountsResponse](#api.PartyAccountsResponse) | Get a list of Accounts by Party |
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





<a name="proto/assets.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/assets.proto



<a name="vega.Asset"></a>

### Asset



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  |  |
| name | [string](#string) |  |  |
| symbol | [string](#string) |  |  |
| totalSupply | [string](#string) |  | this may very much likely be a big.Int |
| decimals | [uint64](#uint64) |  |  |
| builtinAsset | [BuiltinAsset](#vega.BuiltinAsset) |  |  |
| erc20 | [ERC20](#vega.ERC20) |  |  |






<a name="vega.AssetSource"></a>

### AssetSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| builtinAsset | [BuiltinAsset](#vega.BuiltinAsset) |  |  |
| erc20 | [ERC20](#vega.ERC20) |  |  |






<a name="vega.BuiltinAsset"></a>

### BuiltinAsset



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| symbol | [string](#string) |  |  |
| totalSupply | [string](#string) |  |  |
| decimals | [uint64](#uint64) |  |  |






<a name="vega.DevAssets"></a>

### DevAssets



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sources | [AssetSource](#vega.AssetSource) | repeated |  |






<a name="vega.ERC20"></a>

### ERC20



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contractAddress | [string](#string) |  |  |















<a name="proto/governance.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/governance.proto



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
| changes | [Market](#vega.Market) |  |  |






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



<a name="vega.ContinuousTrading"></a>

### ContinuousTrading



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tickSize | [uint64](#uint64) |  |  |






<a name="vega.DiscreteTrading"></a>

### DiscreteTrading



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| duration | [int64](#int64) |  |  |






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
| id | [string](#string) |  | 32 pseudo-random upper-case letters and digits |
| name | [string](#string) |  | a human-understandable name for the Market, perhaps including a currency pair and a maturity date |
| tradableInstrument | [TradableInstrument](#vega.TradableInstrument) |  |  |
| decimalPlaces | [uint64](#uint64) |  | the number of decimal places that a price must be shifted by in order to get a correct price denominated in the currency of the Market. ie `realPrice = price / 10^decimalPlaces` |
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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| owner | [string](#string) |  |  |
| balance | [uint64](#uint64) |  |  |
| asset | [string](#string) |  |  |
| marketID | [string](#string) |  |  |
| type | [AccountType](#vega.AccountType) |  |  |






<a name="vega.Amount"></a>

### Amount



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  |  |






<a name="vega.Candle"></a>

### Candle



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [int64](#int64) |  | nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| datetime | [string](#string) |  | ISO 8601 datetime |
| high | [uint64](#uint64) |  |  |
| low | [uint64](#uint64) |  |  |
| open | [uint64](#uint64) |  |  |
| close | [uint64](#uint64) |  |  |
| volume | [uint64](#uint64) |  |  |
| interval | [Interval](#vega.Interval) |  |  |






<a name="vega.ErrorDetail"></a>

### ErrorDetail



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [int32](#int32) |  | a Vega API domain specific unique error code, useful for client side mappings. e.g. 10004 |
| message | [string](#string) |  | a message that describes the error in more detail, should describe the problem encountered. |
| inner | [string](#string) |  | any inner error information that could add more context, or be helpful for error reporting. |






<a name="vega.FinancialAmount"></a>

### FinancialAmount



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| amount | [int64](#int64) |  |  |
| asset | [string](#string) |  |  |






<a name="vega.LedgerEntry"></a>

### LedgerEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fromAccount | [string](#string) |  |  |
| toAccount | [string](#string) |  |  |
| amount | [uint64](#uint64) |  |  |
| reference | [string](#string) |  |  |
| type | [string](#string) |  |  |
| timestamp | [int64](#int64) |  |  |






<a name="vega.MarginLevels"></a>

### MarginLevels



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maintenanceMargin | [uint64](#uint64) |  |  |
| searchLevel | [uint64](#uint64) |  |  |
| initialMargin | [uint64](#uint64) |  |  |
| collateralReleaseLevel | [uint64](#uint64) |  |  |
| partyID | [string](#string) |  |  |
| marketID | [string](#string) |  |  |
| asset | [string](#string) |  |  |
| timestamp | [int64](#int64) |  |  |






<a name="vega.MarketData"></a>

### MarketData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| markPrice | [uint64](#uint64) |  |  |
| bestBidPrice | [uint64](#uint64) |  |  |
| bestBidVolume | [uint64](#uint64) |  |  |
| bestOfferPrice | [uint64](#uint64) |  |  |
| bestOfferVolume | [uint64](#uint64) |  |  |
| midPrice | [uint64](#uint64) |  |  |
| market | [string](#string) |  |  |
| timestamp | [int64](#int64) |  |  |






<a name="vega.MarketDepth"></a>

### MarketDepth



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| buy | [PriceLevel](#vega.PriceLevel) | repeated |  |
| sell | [PriceLevel](#vega.PriceLevel) | repeated |  |






<a name="vega.NodeRegistration"></a>

### NodeRegistration



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pubKey | [bytes](#bytes) |  |  |
| chainPubKey | [bytes](#bytes) |  |  |






<a name="vega.NodeSignature"></a>

### NodeSignature



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  |  |
| sig | [bytes](#bytes) |  |  |
| kind | [NodeSignatureKind](#vega.NodeSignatureKind) |  |  |






<a name="vega.NodeVote"></a>

### NodeVote



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pubKey | [bytes](#bytes) |  |  |
| reference | [string](#string) |  |  |






<a name="vega.Order"></a>

### Order



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| marketID | [string](#string) |  |  |
| partyID | [string](#string) |  |  |
| side | [Side](#vega.Side) |  |  |
| price | [uint64](#uint64) |  |  |
| size | [uint64](#uint64) |  |  |
| remaining | [uint64](#uint64) |  |  |
| timeInForce | [Order.TimeInForce](#vega.Order.TimeInForce) |  |  |
| type | [Order.Type](#vega.Order.Type) |  |  |
| createdAt | [int64](#int64) |  | nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| status | [Order.Status](#vega.Order.Status) |  | If `status` is `STATUS_REJECTED`, check `reason`. |
| expiresAt | [int64](#int64) |  |  |
| reference | [string](#string) |  |  |
| reason | [OrderError](#vega.OrderError) |  |  |
| updatedAt | [int64](#int64) |  |  |
| version | [uint64](#uint64) |  | Versioning support for amends, orders start at version 1 and increment after each successful amend |






<a name="vega.OrderAmendment"></a>

### OrderAmendment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  | required to find the order, not being updated |
| partyID | [string](#string) |  |  |
| marketID | [string](#string) |  |  |
| price | [Price](#vega.Price) |  | these can be amended |
| sizeDelta | [int64](#int64) |  |  |
| expiresAt | [Timestamp](#vega.Timestamp) |  |  |
| timeInForce | [Order.TimeInForce](#vega.Order.TimeInForce) |  |  |






<a name="vega.OrderCancellation"></a>

### OrderCancellation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  |  |
| marketID | [string](#string) |  |  |
| partyID | [string](#string) |  |  |






<a name="vega.OrderCancellationConfirmation"></a>

### OrderCancellationConfirmation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| order | [Order](#vega.Order) |  |  |






<a name="vega.OrderConfirmation"></a>

### OrderConfirmation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| order | [Order](#vega.Order) |  |  |
| trades | [Trade](#vega.Trade) | repeated |  |
| passiveOrdersAffected | [Order](#vega.Order) | repeated |  |






<a name="vega.OrderSubmission"></a>

### OrderSubmission



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| marketID | [string](#string) |  |  |
| partyID | [string](#string) |  |  |
| price | [uint64](#uint64) |  | mandatory for Limit orders, not required for Market orders |
| size | [uint64](#uint64) |  |  |
| side | [Side](#vega.Side) |  |  |
| timeInForce | [Order.TimeInForce](#vega.Order.TimeInForce) |  |  |
| expiresAt | [int64](#int64) |  | mandatory for GTT orders, not required for GTC, IOC, FOK |
| type | [Order.Type](#vega.Order.Type) |  |  |
| reference | [string](#string) |  |  |






<a name="vega.Party"></a>

### Party



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="vega.Position"></a>

### Position



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| partyID | [string](#string) |  |  |
| openVolume | [int64](#int64) |  |  |
| realisedPNL | [int64](#int64) |  |  |
| unrealisedPNL | [int64](#int64) |  |  |
| averageEntryPrice | [uint64](#uint64) |  |  |






<a name="vega.PositionTrade"></a>

### PositionTrade



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| volume | [int64](#int64) |  |  |
| price | [uint64](#uint64) |  |  |






<a name="vega.Price"></a>

### Price



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [uint64](#uint64) |  |  |






<a name="vega.PriceLevel"></a>

### PriceLevel



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| price | [uint64](#uint64) |  |  |
| numberOfOrders | [uint64](#uint64) |  |  |
| volume | [uint64](#uint64) |  |  |
| cumulativeVolume | [uint64](#uint64) |  |  |






<a name="vega.RiskFactor"></a>

### RiskFactor



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| market | [string](#string) |  |  |
| short | [double](#double) |  |  |
| long | [double](#double) |  |  |






<a name="vega.RiskResult"></a>

### RiskResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| updatedTimestamp | [int64](#int64) |  | timestamp when these risk factors were generated |
| riskFactors | [RiskResult.RiskFactorsEntry](#vega.RiskResult.RiskFactorsEntry) | repeated | risk factors (long and short) for each marginable asset/currency (usually == settlement assets) in the market |
| nextUpdateTimestamp | [int64](#int64) |  | time when risk factors are expected to change (or empty if risk factors are continually updated) |
| predictedNextRiskFactors | [RiskResult.PredictedNextRiskFactorsEntry](#vega.RiskResult.PredictedNextRiskFactorsEntry) | repeated | predicted risk factors at next change (what they&#39;d be if the change occurred now) |






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






<a name="vega.SignedBundle"></a>

### SignedBundle



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [bytes](#bytes) |  |  |
| sig | [bytes](#bytes) |  |  |
| address | [bytes](#bytes) |  |  |
| pubKey | [bytes](#bytes) |  |  |






<a name="vega.Statistics"></a>

### Statistics



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blockHeight | [uint64](#uint64) |  |  |
| backlogLength | [uint64](#uint64) |  |  |
| totalPeers | [uint64](#uint64) |  |  |
| genesisTime | [string](#string) |  | ISO 8601 datetime, nanosecond precision |
| currentTime | [string](#string) |  | ISO 8601 datetime, nanosecond precision |
| vegaTime | [string](#string) |  | ISO 8601 datetime, nanosecond precision |
| status | [ChainStatus](#vega.ChainStatus) |  |  |
| txPerBlock | [uint64](#uint64) |  |  |
| averageTxBytes | [uint64](#uint64) |  |  |
| averageOrdersPerBlock | [uint64](#uint64) |  |  |
| tradesPerSecond | [uint64](#uint64) |  |  |
| ordersPerSecond | [uint64](#uint64) |  |  |
| totalMarkets | [uint64](#uint64) |  |  |
| totalAmendOrder | [uint64](#uint64) |  |  |
| totalCancelOrder | [uint64](#uint64) |  |  |
| totalCreateOrder | [uint64](#uint64) |  |  |
| totalOrders | [uint64](#uint64) |  |  |
| totalTrades | [uint64](#uint64) |  |  |
| orderSubscriptions | [uint32](#uint32) |  |  |
| tradeSubscriptions | [uint32](#uint32) |  |  |
| candleSubscriptions | [uint32](#uint32) |  |  |
| marketDepthSubscriptions | [uint32](#uint32) |  |  |
| positionsSubscriptions | [uint32](#uint32) |  |  |
| accountSubscriptions | [uint32](#uint32) |  |  |
| marketDataSubscriptions | [uint32](#uint32) |  |  |
| appVersionHash | [string](#string) |  |  |
| appVersion | [string](#string) |  |  |
| chainVersion | [string](#string) |  |  |
| blockDuration | [uint64](#uint64) |  | nanoseconds |
| uptime | [string](#string) |  | ISO 8601 datetime, nanosecond precision |
| chainID | [string](#string) |  | Unique ID of the blockchain |






<a name="vega.Timestamp"></a>

### Timestamp



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [int64](#int64) |  |  |






<a name="vega.Trade"></a>

### Trade



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| marketID | [string](#string) |  |  |
| price | [uint64](#uint64) |  |  |
| size | [uint64](#uint64) |  |  |
| buyer | [string](#string) |  |  |
| seller | [string](#string) |  |  |
| aggressor | [Side](#vega.Side) |  |  |
| buyOrder | [string](#string) |  |  |
| sellOrder | [string](#string) |  |  |
| timestamp | [int64](#int64) |  | nanoseconds since the epoch. See [`VegaTimeResponse`](#api.VegaTimeResponse).`timestamp`. |
| type | [Trade.Type](#vega.Trade.Type) |  |  |






<a name="vega.TradeSet"></a>

### TradeSet



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trades | [Trade](#vega.Trade) | repeated |  |






<a name="vega.Transfer"></a>

### Transfer



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | [string](#string) |  |  |
| amount | [FinancialAmount](#vega.FinancialAmount) |  |  |
| type | [TransferType](#vega.TransferType) |  |  |
| minAmount | [int64](#int64) |  |  |






<a name="vega.TransferBalance"></a>

### TransferBalance



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account | [Account](#vega.Account) |  |  |
| balance | [uint64](#uint64) |  |  |






<a name="vega.TransferRequest"></a>

### TransferRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fromAccount | [Account](#vega.Account) | repeated |  |
| toAccount | [Account](#vega.Account) | repeated |  |
| amount | [uint64](#uint64) |  |  |
| minAmount | [uint64](#uint64) |  |  |
| asset | [string](#string) |  |  |
| reference | [string](#string) |  |  |






<a name="vega.TransferResponse"></a>

### TransferResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| transfers | [LedgerEntry](#vega.LedgerEntry) | repeated |  |
| balances | [TransferBalance](#vega.TransferBalance) | repeated |  |








<a name="vega.AccountType"></a>

### AccountType


| Name | Number | Description |
| ---- | ------ | ----------- |
| ACCOUNT_TYPE_UNSPECIFIED | 0 |  |
| ACCOUNT_TYPE_INSURANCE | 1 |  |
| ACCOUNT_TYPE_SETTLEMENT | 2 |  |
| ACCOUNT_TYPE_MARGIN | 3 |  |
| ACCOUNT_TYPE_GENERAL | 4 |  |



<a name="vega.ChainStatus"></a>

### ChainStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| CHAIN_STATUS_UNSPECIFIED | 0 |  |
| CHAIN_STATUS_DISCONNECTED | 1 |  |
| CHAIN_STATUS_REPLAYING | 2 |  |
| CHAIN_STATUS_CONNECTED | 3 |  |



<a name="vega.Interval"></a>

### Interval


| Name | Number | Description |
| ---- | ------ | ----------- |
| INTERVAL_UNSPECIFIED | 0 | Default value, always invalid |
| INTERVAL_I1M | 60 | 1 minute |
| INTERVAL_I5M | 300 | 5 minutes |
| INTERVAL_I15M | 900 | 15 minutes |
| INTERVAL_I1H | 3600 | 1 hour |
| INTERVAL_I6H | 21600 | 6 hours |
| INTERVAL_I1D | 86400 | 1 day |



<a name="vega.NodeSignatureKind"></a>

### NodeSignatureKind


| Name | Number | Description |
| ---- | ------ | ----------- |
| NODE_SIGNATURE_KIND_UNSPECIFIED | 0 |  |
| NODE_SIGNATURE_KIND_ASSET_NEW | 1 |  |
| NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL | 2 |  |



<a name="vega.Order.Status"></a>

### Order.Status
Order Status

See [What order types are available to trade on Vega?](https://docs.vega.xyz/docs/50-trading-questions/#what-order-types-are-available-to-trade-on-vega) for details.

| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_INVALID | 0 | Default value, always invalid |
| STATUS_ACTIVE | 1 | used for active unfilled or partially filled orders |
| STATUS_EXPIRED | 2 | used for expired GTT orders |
| STATUS_CANCELLED | 3 | used for orders cancelled by the party that created the order |
| STATUS_STOPPED | 4 | used for unfilled FOK or IOC orders, and for orders that were stopped by the network |
| STATUS_FILLED | 5 | used for closed fully filled orders |
| STATUS_REJECTED | 6 | used for orders when not enough collateral was available to fill the margin requirements |
| STATUS_PARTIALLY_FILLED | 7 | used for closed partially filled IOC orders |



<a name="vega.Order.TimeInForce"></a>

### Order.TimeInForce
Order Time in Force

See [What order types are available to trade on Vega?](https://docs.vega.xyz/docs/50-trading-questions/#what-order-types-are-available-to-trade-on-vega) for details.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TIF_UNSPECIFIED | 0 | Default value, can be valid for an amend |
| TIF_GTC | 1 | good til cancelled |
| TIF_GTT | 2 | good til time |
| TIF_IOC | 3 | immediate or cancel |
| TIF_FOK | 4 | fill or kill |



<a name="vega.Order.Type"></a>

### Order.Type
Order Type

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Default value, always invalid |
| TYPE_LIMIT | 1 | used for Limit orders |
| TYPE_MARKET | 2 | used for Market orders |
| TYPE_NETWORK | 3 | used for orders where the initiating party is the network (used for distressed traders) |



<a name="vega.OrderError"></a>

### OrderError


| Name | Number | Description |
| ---- | ------ | ----------- |
| ORDER_ERROR_NONE | 0 |  |
| ORDER_ERROR_INVALID_MARKET_ID | 1 |  |
| ORDER_ERROR_INVALID_ORDER_ID | 2 |  |
| ORDER_ERROR_OUT_OF_SEQUENCE | 3 |  |
| ORDER_ERROR_INVALID_REMAINING_SIZE | 4 |  |
| ORDER_ERROR_TIME_FAILURE | 5 |  |
| ORDER_ERROR_REMOVAL_FAILURE | 6 |  |
| ORDER_ERROR_INVALID_EXPIRATION_DATETIME | 7 |  |
| ORDER_ERROR_INVALID_ORDER_REFERENCE | 8 |  |
| ORDER_ERROR_EDIT_NOT_ALLOWED | 9 |  |
| ORDER_ERROR_AMEND_FAILURE | 10 |  |
| ORDER_ERROR_NOT_FOUND | 11 |  |
| ORDER_ERROR_INVALID_PARTY_ID | 12 |  |
| ORDER_ERROR_MARKET_CLOSED | 13 |  |
| ORDER_ERROR_MARGIN_CHECK_FAILED | 14 |  |
| ORDER_ERROR_MISSING_GENERAL_ACCOUNT | 15 |  |
| ORDER_ERROR_INTERNAL_ERROR | 16 |  |
| ORDER_ERROR_INVALID_SIZE | 17 |  |
| ORDER_ERROR_INVALID_PERSISTENCE | 18 |  |
| ORDER_ERROR_INVALID_TYPE | 19 |  |



<a name="vega.Side"></a>

### Side


| Name | Number | Description |
| ---- | ------ | ----------- |
| SIDE_UNSPECIFIED | 0 | Default value, always invalid |
| SIDE_BUY | 1 | Buy |
| SIDE_SELL | 2 | Sell |



<a name="vega.Trade.Type"></a>

### Trade.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Default value, always invalid |
| TYPE_DEFAULT | 1 |  |
| TYPE_NETWORK_CLOSE_OUT_GOOD | 2 |  |
| TYPE_NETWORK_CLOSE_OUT_BAD | 3 |  |



<a name="vega.TransferType"></a>

### TransferType


| Name | Number | Description |
| ---- | ------ | ----------- |
| TRANSFER_TYPE_UNSPECIFIED | 0 |  |
| TRANSFER_TYPE_LOSS | 1 |  |
| TRANSFER_TYPE_WIN | 2 |  |
| TRANSFER_TYPE_CLOSE | 3 |  |
| TRANSFER_TYPE_MTM_LOSS | 4 |  |
| TRANSFER_TYPE_MTM_WIN | 5 |  |
| TRANSFER_TYPE_MARGIN_LOW | 6 |  |
| TRANSFER_TYPE_MARGIN_HIGH | 7 |  |
| TRANSFER_TYPE_MARGIN_CONFISCATED | 8 |  |










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

