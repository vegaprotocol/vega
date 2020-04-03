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
    - [GetProposalByIDRequest](#api.GetProposalByIDRequest)
    - [GetProposalByReferenceRequest](#api.GetProposalByReferenceRequest)
    - [GetProposalsResponse](#api.GetProposalsResponse)
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
    - [OrderByIDRequest](#api.OrderByIDRequest)
    - [OrderByMarketAndIdRequest](#api.OrderByMarketAndIdRequest)
    - [OrderByMarketAndIdResponse](#api.OrderByMarketAndIdResponse)
    - [OrderByReferenceIDRequest](#api.OrderByReferenceIDRequest)
    - [OrderByReferenceRequest](#api.OrderByReferenceRequest)
    - [OrderByReferenceResponse](#api.OrderByReferenceResponse)
    - [OrderVersionsByIDRequest](#api.OrderVersionsByIDRequest)
    - [OrderVersionsByReferenceIDRequest](#api.OrderVersionsByReferenceIDRequest)
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
    - [WithdrawRequest](#api.WithdrawRequest)
    - [WithdrawResponse](#api.WithdrawResponse)



    - [trading](#api.trading)
    - [trading_data](#api.trading_data)


- [proto/governance.proto](#proto/governance.proto)
    - [NetworkConfiguration](#vega.NetworkConfiguration)
    - [NewMarket](#vega.NewMarket)
    - [Proposal](#vega.Proposal)
    - [ProposalTerms](#vega.ProposalTerms)
    - [ProposalVote](#vega.ProposalVote)
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
    - [Order](#vega.Order)
    - [OrderAmendment](#vega.OrderAmendment)
    - [OrderCancellation](#vega.OrderCancellation)
    - [OrderCancellationConfirmation](#vega.OrderCancellationConfirmation)
    - [OrderConfirmation](#vega.OrderConfirmation)
    - [OrderSubmission](#vega.OrderSubmission)
    - [Party](#vega.Party)
    - [PendingOrder](#vega.PendingOrder)
    - [Position](#vega.Position)
    - [PositionTrade](#vega.PositionTrade)
    - [PriceLevel](#vega.PriceLevel)
    - [RiskFactor](#vega.RiskFactor)
    - [RiskResult](#vega.RiskResult)
    - [RiskResult.PredictedNextRiskFactorsEntry](#vega.RiskResult.PredictedNextRiskFactorsEntry)
    - [RiskResult.RiskFactorsEntry](#vega.RiskResult.RiskFactorsEntry)
    - [SignedBundle](#vega.SignedBundle)
    - [Statistics](#vega.Statistics)
    - [Trade](#vega.Trade)
    - [TradeSet](#vega.TradeSet)
    - [Transfer](#vega.Transfer)
    - [TransferBalance](#vega.TransferBalance)
    - [TransferRequest](#vega.TransferRequest)
    - [TransferResponse](#vega.TransferResponse)
    - [Withdraw](#vega.Withdraw)

    - [AccountType](#vega.AccountType)
    - [ChainStatus](#vega.ChainStatus)
    - [Interval](#vega.Interval)
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






<a name="api.GetProposalByIDRequest"></a>

### GetProposalByIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  |  |






<a name="api.GetProposalByReferenceRequest"></a>

### GetProposalByReferenceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| Reference | [string](#string) |  |  |






<a name="api.GetProposalsResponse"></a>

### GetProposalsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| proposals | [vega.ProposalVote](#vega.ProposalVote) | repeated |  |






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
| version | [uint64](#uint64) |  | version of the order (0 for most recent; 1 for original; 2 for first amendment, etc) |






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
| version | [uint64](#uint64) |  | version of the order (0 for most recent; 1 for original; 2 for first amendment, etc) |






<a name="api.OrderByReferenceRequest"></a>

### OrderByReferenceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reference | [string](#string) |  |  |
| version | [uint64](#uint64) |  | version of the order (0 for most recent; 1 for original; 2 for first amendment, etc) |






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






<a name="api.OrderVersionsByReferenceIDRequest"></a>

### OrderVersionsByReferenceIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| referenceID | [string](#string) |  |  |






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
| pendingOrder | [vega.PendingOrder](#vega.PendingOrder) |  |  |






<a name="api.PrepareCancelOrderResponse"></a>

### PrepareCancelOrderResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blob | [bytes](#bytes) |  |  |
| pendingOrder | [vega.PendingOrder](#vega.PendingOrder) |  |  |






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
| pendingOrder | [vega.PendingOrder](#vega.PendingOrder) |  |  |






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
| OrderVersionsByReferenceID | [OrderVersionsByReferenceIDRequest](#api.OrderVersionsByReferenceIDRequest) | [OrderVersionsResponse](#api.OrderVersionsResponse) | Get all versions of the order by its referenceID |
| MarginLevels | [MarginLevelsRequest](#api.MarginLevelsRequest) | [MarginLevelsResponse](#api.MarginLevelsResponse) | Get Margin Levels by PartyID |
| Parties | [.google.protobuf.Empty](#google.protobuf.Empty) | [PartiesResponse](#api.PartiesResponse) | Get a list of Parties |
| PartyByID | [PartyByIDRequest](#api.PartyByIDRequest) | [PartyByIDResponse](#api.PartyByIDResponse) | Get a Party by ID |
| PositionsByParty | [PositionsByPartyRequest](#api.PositionsByPartyRequest) | [PositionsByPartyResponse](#api.PositionsByPartyResponse) | Get a list of Positions by Party |
| LastTrade | [LastTradeRequest](#api.LastTradeRequest) | [LastTradeResponse](#api.LastTradeResponse) | Get latest Trade |
| TradesByMarket | [TradesByMarketRequest](#api.TradesByMarketRequest) | [TradesByMarketResponse](#api.TradesByMarketResponse) | Get a list of Trades by Market |
| TradesByOrder | [TradesByOrderRequest](#api.TradesByOrderRequest) | [TradesByOrderResponse](#api.TradesByOrderResponse) | Get a list of Trades by Order |
| TradesByParty | [TradesByPartyRequest](#api.TradesByPartyRequest) | [TradesByPartyResponse](#api.TradesByPartyResponse) | Get a list of Trades by Party |
| GetProposals | [.google.protobuf.Empty](#google.protobuf.Empty) | [GetProposalsResponse](#api.GetProposalsResponse) | Get all proposals |
| GetOpenProposals | [.google.protobuf.Empty](#google.protobuf.Empty) | [GetProposalsResponse](#api.GetProposalsResponse) | Get all OPEN proposals |
| GetProposalByID | [GetProposalByIDRequest](#api.GetProposalByIDRequest) | [.vega.ProposalVote](#vega.ProposalVote) | Get a proposal by ID |
| GetProposalByReference | [GetProposalByReferenceRequest](#api.GetProposalByReferenceRequest) | [.vega.ProposalVote](#vega.ProposalVote) | Get a proposal by reference |
| ObserveProposals | [.google.protobuf.Empty](#google.protobuf.Empty) | [.vega.ProposalVote](#vega.ProposalVote) stream | Subscribe to a stream of updates to proposal data |
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





<a name="proto/governance.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/governance.proto



<a name="vega.NetworkConfiguration"></a>

### NetworkConfiguration



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| minCloseInSeconds | [int64](#int64) |  |  |
| maxCloseInSeconds | [int64](#int64) |  |  |
| minEnactInSeconds | [int64](#int64) |  |  |
| maxEnactInSeconds | [int64](#int64) |  |  |
| minParticipationStake | [uint64](#uint64) |  |  |






<a name="vega.NewMarket"></a>

### NewMarket



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changes | [Market](#vega.Market) |  |  |






<a name="vega.Proposal"></a>

### Proposal



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ID | [string](#string) |  |  |
| reference | [string](#string) |  |  |
| partyID | [string](#string) |  |  |
| state | [Proposal.State](#vega.Proposal.State) |  |  |
| timestamp | [int64](#int64) |  |  |
| terms | [ProposalTerms](#vega.ProposalTerms) |  |  |






<a name="vega.ProposalTerms"></a>

### ProposalTerms



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| closingTimestamp | [int64](#int64) |  |  |
| enactmentTimestamp | [int64](#int64) |  |  |
| minParticipationStake | [uint64](#uint64) |  |  |
| updateMarket | [UpdateMarket](#vega.UpdateMarket) |  |  |
| newMarket | [NewMarket](#vega.NewMarket) |  |  |
| updateNetwork | [UpdateNetwork](#vega.UpdateNetwork) |  |  |






<a name="vega.ProposalVote"></a>

### ProposalVote



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| proposal | [Proposal](#vega.Proposal) |  |  |
| yes | [Vote](#vega.Vote) | repeated |  |
| no | [Vote](#vega.Vote) | repeated |  |






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
| partyID | [string](#string) |  |  |
| value | [Vote.Value](#vega.Vote.Value) |  |  |
| proposalID | [string](#string) |  |  |








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
| FAILED | 0 | Proposal could not be enacted after being accepted by the network |
| OPEN | 1 | Proposal is open for voting. |
| PASSED | 2 | Proposal has gained enough support to be executed. |
| REJECTED | 3 | Proposal wasn&#39;t accepted (validation failed, author not allowed to submit proposals) |
| DECLINED | 4 | Proposal didn&#39;t get enough votes |
| ENACTED | 5 | Proposal has been executed and the changes under this proposal have now been applied. |



<a name="vega.Vote.Value"></a>

### Vote.Value


| Name | Number | Description |
| ---- | ------ | ----------- |
| NO | 0 |  |
| YES | 1 |  |










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
| createdAt | [int64](#int64) |  |  |
| status | [Order.Status](#vega.Order.Status) |  | If `status` is `Rejected`, check `reason`. |
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
| price | [uint64](#uint64) |  | these can be amended |
| sizeDelta | [int64](#int64) |  |  |
| expiresAt | [int64](#int64) |  |  |
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






<a name="vega.PendingOrder"></a>

### PendingOrder



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reference | [string](#string) |  |  |
| price | [uint64](#uint64) |  |  |
| timeInForce | [Order.TimeInForce](#vega.Order.TimeInForce) |  |  |
| side | [Side](#vega.Side) |  |  |
| marketID | [string](#string) |  |  |
| size | [uint64](#uint64) |  |  |
| partyID | [string](#string) |  |  |
| status | [Order.Status](#vega.Order.Status) |  |  |
| id | [string](#string) |  |  |
| type | [Order.Type](#vega.Order.Type) |  |  |






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






<a name="vega.Withdraw"></a>

### Withdraw



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |
| amount | [uint64](#uint64) |  |  |
| asset | [string](#string) |  |  |








<a name="vega.AccountType"></a>

### AccountType


| Name | Number | Description |
| ---- | ------ | ----------- |
| ALL | 0 |  |
| INSURANCE | 1 |  |
| SETTLEMENT | 2 |  |
| MARGIN | 3 |  |
| GENERAL | 4 |  |



<a name="vega.ChainStatus"></a>

### ChainStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| DISCONNECTED | 0 |  |
| REPLAYING | 1 |  |
| CONNECTED | 2 |  |



<a name="vega.Interval"></a>

### Interval


| Name | Number | Description |
| ---- | ------ | ----------- |
| I1M | 0 | 1 minute |
| I5M | 1 | 5 minutes |
| I15M | 2 | 15 minutes |
| I1H | 3 | 1 hour |
| I6H | 4 | 6 hours |
| I1D | 5 | 1 day |



<a name="vega.Order.Status"></a>

### Order.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| Active | 0 |  |
| Expired | 1 |  |
| Cancelled | 2 |  |
| Stopped | 3 |  |
| Filled | 4 |  |
| Rejected | 5 |  |
| PartiallyFilled | 6 |  |



<a name="vega.Order.TimeInForce"></a>

### Order.TimeInForce
Order Time in Force

| Name | Number | Description |
| ---- | ------ | ----------- |
| GTC | 0 | good til cancelled |
| GTT | 1 | good til time |
| IOC | 2 | immediate or cancel |
| FOK | 3 | fill or kill |



<a name="vega.Order.Type"></a>

### Order.Type
Order Type

| Name | Number | Description |
| ---- | ------ | ----------- |
| LIMIT | 0 | used for Limit orders |
| MARKET | 1 | used for Market orders |
| NETWORK | 2 | used for orders where the initiating party is the network (used for distressed traders) |



<a name="vega.OrderError"></a>

### OrderError


| Name | Number | Description |
| ---- | ------ | ----------- |
| NONE | 0 |  |
| INVALID_MARKET_ID | 1 |  |
| INVALID_ORDER_ID | 2 |  |
| ORDER_OUT_OF_SEQUENCE | 3 |  |
| INVALID_REMAINING_SIZE | 4 |  |
| TIME_FAILURE | 5 |  |
| ORDER_REMOVAL_FAILURE | 6 |  |
| INVALID_EXPIRATION_DATETIME | 7 |  |
| INVALID_ORDER_REFERENCE | 8 |  |
| EDIT_NOT_ALLOWED | 9 |  |
| ORDER_AMEND_FAILURE | 10 |  |
| ORDER_NOT_FOUND | 11 |  |
| INVALID_PARTY_ID | 12 |  |
| MARKET_CLOSED | 13 |  |
| MARGIN_CHECK_FAILED | 14 |  |
| MISSING_GENERAL_ACCOUNT | 15 |  |
| INTERNAL_ERROR | 16 |  |
| INVALID_SIZE | 17 |  |
| INVALID_PERSISTENCE | 18 |  |



<a name="vega.Side"></a>

### Side


| Name | Number | Description |
| ---- | ------ | ----------- |
| Buy | 0 |  |
| Sell | 1 |  |



<a name="vega.Trade.Type"></a>

### Trade.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| DEFAULT | 0 |  |
| NETWORK_CLOSE_OUT_GOOD | 1 |  |
| NETWORK_CLOSE_OUT_BAD | 2 |  |



<a name="vega.TransferType"></a>

### TransferType


| Name | Number | Description |
| ---- | ------ | ----------- |
| LOSS | 0 |  |
| WIN | 1 |  |
| CLOSE | 2 |  |
| MTM_LOSS | 3 |  |
| MTM_WIN | 4 |  |
| MARGIN_LOW | 5 |  |
| MARGIN_HIGH | 6 |  |
| MARGIN_CONFISCATED | 7 |  |










## Scalar Value Types

| .proto Type | Notes | C++ Type | Java Type | Python Type |
| ----------- | ----- | -------- | --------- | ----------- |
| <a name="double" /> double |  | double | double | float |
| <a name="float" /> float |  | float | float | float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long |
| <a name="bool" /> bool |  | bool | boolean | boolean |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str |

