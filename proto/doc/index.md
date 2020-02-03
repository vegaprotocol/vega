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
    - [CheckTokenRequest](#api.CheckTokenRequest)
    - [CheckTokenResponse](#api.CheckTokenResponse)
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
    - [NotifyTraderAccountRequest](#api.NotifyTraderAccountRequest)
    - [NotifyTraderAccountResponse](#api.NotifyTraderAccountResponse)
    - [OrderByMarketAndIdRequest](#api.OrderByMarketAndIdRequest)
    - [OrderByMarketAndIdResponse](#api.OrderByMarketAndIdResponse)
    - [OrderByReferenceRequest](#api.OrderByReferenceRequest)
    - [OrderByReferenceResponse](#api.OrderByReferenceResponse)
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
    - [SignInRequest](#api.SignInRequest)
    - [SignInResponse](#api.SignInResponse)
    - [SubmitOrderRequest](#api.SubmitOrderRequest)
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
    - [FinancialAmount](#vega.FinancialAmount)
    - [LedgerEntry](#vega.LedgerEntry)
    - [MarginLevels](#vega.MarginLevels)
    - [MarketData](#vega.MarketData)
    - [MarketDepth](#vega.MarketDepth)
    - [NotifyTraderAccount](#vega.NotifyTraderAccount)
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
| token | [string](#string) |  |  |






<a name="api.CancelOrderRequest"></a>

### CancelOrderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cancellation | [vega.OrderCancellation](#vega.OrderCancellation) |  |  |
| token | [string](#string) |  |  |






<a name="api.CandlesRequest"></a>

### CandlesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| marketID | [string](#string) |  |  |
| sinceTimestamp | [int64](#int64) |  |  |
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






<a name="api.CheckTokenRequest"></a>

### CheckTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| partyID | [string](#string) |  |  |
| token | [string](#string) |  |  |






<a name="api.CheckTokenResponse"></a>

### CheckTokenResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ok | [bool](#bool) |  |  |






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
| markets | [vega.Market](#vega.Market) | repeated |  |






<a name="api.NotifyTraderAccountRequest"></a>

### NotifyTraderAccountRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| notif | [vega.NotifyTraderAccount](#vega.NotifyTraderAccount) |  |  |






<a name="api.NotifyTraderAccountResponse"></a>

### NotifyTraderAccountResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| submitted | [bool](#bool) |  |  |






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






<a name="api.SignInRequest"></a>

### SignInRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| password | [string](#string) |  |  |






<a name="api.SignInResponse"></a>

### SignInResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |






<a name="api.SubmitOrderRequest"></a>

### SubmitOrderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| submission | [vega.OrderSubmission](#vega.OrderSubmission) |  |  |
| token | [string](#string) |  |  |






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
| timestamp | [int64](#int64) |  |  |






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
| SubmitOrder | [SubmitOrderRequest](#api.SubmitOrderRequest) | [.vega.PendingOrder](#vega.PendingOrder) | Submit an Order |
| CancelOrder | [CancelOrderRequest](#api.CancelOrderRequest) | [.vega.PendingOrder](#vega.PendingOrder) | Cancel an Order |
| AmendOrder | [AmendOrderRequest](#api.AmendOrderRequest) | [.vega.PendingOrder](#vega.PendingOrder) | Amend an Order |
| SignIn | [SignInRequest](#api.SignInRequest) | [SignInResponse](#api.SignInResponse) | Sign In |
| NotifyTraderAccount | [NotifyTraderAccountRequest](#api.NotifyTraderAccountRequest) | [NotifyTraderAccountResponse](#api.NotifyTraderAccountResponse) | Request balance increase |
| Withdraw | [WithdrawRequest](#api.WithdrawRequest) | [WithdrawResponse](#api.WithdrawResponse) | Request withdrawal |
| CheckToken | [CheckTokenRequest](#api.CheckTokenRequest) | [CheckTokenResponse](#api.CheckTokenResponse) | Check an API token |


<a name="api.trading_data"></a>

### trading_data


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| OrdersByMarket | [OrdersByMarketRequest](#api.OrdersByMarketRequest) | [OrdersByMarketResponse](#api.OrdersByMarketResponse) | Get Market Orders |
| OrdersByParty | [OrdersByPartyRequest](#api.OrdersByPartyRequest) | [OrdersByPartyResponse](#api.OrdersByPartyResponse) | Get Party Orders |
| OrderByMarketAndID | [OrderByMarketAndIdRequest](#api.OrderByMarketAndIdRequest) | [OrderByMarketAndIdResponse](#api.OrderByMarketAndIdResponse) | Get Market Order by OrderID |
| OrderByReference | [OrderByReferenceRequest](#api.OrderByReferenceRequest) | [OrderByReferenceResponse](#api.OrderByReferenceResponse) | Get an Order by Pending Order reference (UUID) |
| MarketByID | [MarketByIDRequest](#api.MarketByIDRequest) | [MarketByIDResponse](#api.MarketByIDResponse) | Get Market by ID |
| Markets | [.google.protobuf.Empty](#google.protobuf.Empty) | [MarketsResponse](#api.MarketsResponse) | Get a list of Markets |
| MarketDepth | [MarketDepthRequest](#api.MarketDepthRequest) | [MarketDepthResponse](#api.MarketDepthResponse) | Get Market Depth |
| LastTrade | [LastTradeRequest](#api.LastTradeRequest) | [LastTradeResponse](#api.LastTradeResponse) | Get latest Market Trade |
| PartyByID | [PartyByIDRequest](#api.PartyByIDRequest) | [PartyByIDResponse](#api.PartyByIDResponse) | Get Party by ID |
| Parties | [.google.protobuf.Empty](#google.protobuf.Empty) | [PartiesResponse](#api.PartiesResponse) | Get a list of Parties |
| TradesByMarket | [TradesByMarketRequest](#api.TradesByMarketRequest) | [TradesByMarketResponse](#api.TradesByMarketResponse) | Get Market Trades |
| TradesByParty | [TradesByPartyRequest](#api.TradesByPartyRequest) | [TradesByPartyResponse](#api.TradesByPartyResponse) | Get Party Trades |
| TradesByOrder | [TradesByOrderRequest](#api.TradesByOrderRequest) | [TradesByOrderResponse](#api.TradesByOrderResponse) | Get Order Trades |
| PositionsByParty | [PositionsByPartyRequest](#api.PositionsByPartyRequest) | [PositionsByPartyResponse](#api.PositionsByPartyResponse) | Get Party Positions |
| Candles | [CandlesRequest](#api.CandlesRequest) | [CandlesResponse](#api.CandlesResponse) | Get Market Candles |
| Statistics | [.google.protobuf.Empty](#google.protobuf.Empty) | [.vega.Statistics](#vega.Statistics) | Get Statistics |
| GetVegaTime | [.google.protobuf.Empty](#google.protobuf.Empty) | [VegaTimeResponse](#api.VegaTimeResponse) | Get Time |
| MarketDataByID | [MarketDataByIDRequest](#api.MarketDataByIDRequest) | [MarketDataByIDResponse](#api.MarketDataByIDResponse) | Get Market Data by ID |
| MarketsData | [.google.protobuf.Empty](#google.protobuf.Empty) | [MarketsDataResponse](#api.MarketsDataResponse) | Get a list of Market Data |
| MarginLevels | [MarginLevelsRequest](#api.MarginLevelsRequest) | [MarginLevelsResponse](#api.MarginLevelsResponse) | Get Party Margin Levels |
| OrdersSubscribe | [OrdersSubscribeRequest](#api.OrdersSubscribeRequest) | [OrdersStream](#api.OrdersStream) stream | streams |
| TradesSubscribe | [TradesSubscribeRequest](#api.TradesSubscribeRequest) | [TradesStream](#api.TradesStream) stream |  |
| CandlesSubscribe | [CandlesSubscribeRequest](#api.CandlesSubscribeRequest) | [.vega.Candle](#vega.Candle) stream |  |
| MarketDepthSubscribe | [MarketDepthSubscribeRequest](#api.MarketDepthSubscribeRequest) | [.vega.MarketDepth](#vega.MarketDepth) stream |  |
| PositionsSubscribe | [PositionsSubscribeRequest](#api.PositionsSubscribeRequest) | [.vega.Position](#vega.Position) stream |  |
| AccountsSubscribe | [AccountsSubscribeRequest](#api.AccountsSubscribeRequest) | [.vega.Account](#vega.Account) stream |  |
| TransferResponsesSubscribe | [.google.protobuf.Empty](#google.protobuf.Empty) | [.vega.TransferResponse](#vega.TransferResponse) stream |  |
| MarketsDataSubscribe | [MarketsDataSubscribeRequest](#api.MarketsDataSubscribeRequest) | [.vega.MarketData](#vega.MarketData) stream |  |
| MarginLevelsSubscribe | [MarginLevelsSubscribeRequest](#api.MarginLevelsSubscribeRequest) | [.vega.MarginLevels](#vega.MarginLevels) stream |  |
| PartyAccounts | [PartyAccountsRequest](#api.PartyAccountsRequest) | [PartyAccountsResponse](#api.PartyAccountsResponse) | Get Party accounts |
| MarketAccounts | [MarketAccountsRequest](#api.MarketAccountsRequest) | [MarketAccountsResponse](#api.MarketAccountsResponse) | Get Market accounts |





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
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| tradableInstrument | [TradableInstrument](#vega.TradableInstrument) |  |  |
| decimalPlaces | [uint64](#uint64) |  |  |
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
| balance | [int64](#int64) |  |  |
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
| timestamp | [int64](#int64) |  |  |
| datetime | [string](#string) |  |  |
| high | [uint64](#uint64) |  |  |
| low | [uint64](#uint64) |  |  |
| open | [uint64](#uint64) |  |  |
| close | [uint64](#uint64) |  |  |
| volume | [uint64](#uint64) |  |  |
| interval | [Interval](#vega.Interval) |  |  |






<a name="vega.FinancialAmount"></a>

### FinancialAmount



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| amount | [int64](#int64) |  |  |
| asset | [string](#string) |  |  |
| minAmount | [int64](#int64) |  |  |






<a name="vega.LedgerEntry"></a>

### LedgerEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fromAccount | [string](#string) |  |  |
| toAccount | [string](#string) |  |  |
| amount | [int64](#int64) |  |  |
| reference | [string](#string) |  |  |
| type | [string](#string) |  |  |
| timestamp | [int64](#int64) |  |  |






<a name="vega.MarginLevels"></a>

### MarginLevels



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maintenanceMargin | [int64](#int64) |  |  |
| searchLevel | [int64](#int64) |  |  |
| initialMargin | [int64](#int64) |  |  |
| collateralReleaseLevel | [int64](#int64) |  |  |
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






<a name="vega.NotifyTraderAccount"></a>

### NotifyTraderAccount



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| traderID | [string](#string) |  |  |
| amount | [uint64](#uint64) |  |  |






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
| status | [Order.Status](#vega.Order.Status) |  |  |
| expiresAt | [int64](#int64) |  |  |
| reference | [string](#string) |  |  |
| reason | [OrderError](#vega.OrderError) |  |  |






<a name="vega.OrderAmendment"></a>

### OrderAmendment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| orderID | [string](#string) |  |  |
| partyID | [string](#string) |  |  |
| marketID | [string](#string) |  |  |
| price | [uint64](#uint64) |  |  |
| size | [uint64](#uint64) |  |  |
| expiresAt | [int64](#int64) |  |  |
| side | [Side](#vega.Side) |  |  |






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
| price | [uint64](#uint64) |  | do not enforce that price, as Market Order will not have price specified |
| size | [uint64](#uint64) |  |  |
| side | [Side](#vega.Side) |  | make sur for both that they are non nil and the value is part of the respective enums. |
| TimeInForce | [Order.TimeInForce](#vega.Order.TimeInForce) |  |  |
| expiresAt | [int64](#int64) |  | do not enforce as not always required althouth at least check it&#39;s not a negative integer, would be not that very handy to create a time.Time with it |
| type | [Order.Type](#vega.Order.Type) |  |  |






<a name="vega.Party"></a>

### Party



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| positions | [Position](#vega.Position) | repeated |  |






<a name="vega.PendingOrder"></a>

### PendingOrder



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reference | [string](#string) |  |  |
| price | [uint64](#uint64) |  |  |
| TimeInForce | [Order.TimeInForce](#vega.Order.TimeInForce) |  |  |
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






<a name="vega.Statistics"></a>

### Statistics



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blockHeight | [uint64](#uint64) |  |  |
| backlogLength | [uint64](#uint64) |  |  |
| totalPeers | [uint64](#uint64) |  |  |
| genesisTime | [string](#string) |  |  |
| currentTime | [string](#string) |  |  |
| vegaTime | [string](#string) |  |  |
| status | [ChainStatus](#vega.ChainStatus) |  |  |
| txPerBlock | [uint64](#uint64) |  |  |
| averageTxBytes | [uint64](#uint64) |  |  |
| averageOrdersPerBlock | [uint64](#uint64) |  |  |
| tradesPerSecond | [uint64](#uint64) |  |  |
| ordersPerSecond | [uint64](#uint64) |  |  |
| totalMarkets | [uint64](#uint64) |  |  |
| totalParties | [uint64](#uint64) |  |  |
| parties | [string](#string) | repeated |  |
| totalAmendOrder | [uint64](#uint64) |  |  |
| totalCancelOrder | [uint64](#uint64) |  |  |
| totalCreateOrder | [uint64](#uint64) |  |  |
| totalOrders | [uint64](#uint64) |  |  |
| totalTrades | [uint64](#uint64) |  |  |
| orderSubscriptions | [int32](#int32) |  |  |
| tradeSubscriptions | [int32](#int32) |  |  |
| candleSubscriptions | [int32](#int32) |  |  |
| marketDepthSubscriptions | [int32](#int32) |  |  |
| positionsSubscriptions | [int32](#int32) |  |  |
| accountSubscriptions | [int32](#int32) |  |  |
| marketDataSubscriptions | [int32](#int32) |  |  |
| appVersionHash | [string](#string) |  |  |
| appVersion | [string](#string) |  |  |
| chainVersion | [string](#string) |  |  |
| blockDuration | [uint64](#uint64) |  | nanoseconds |
| uptime | [string](#string) |  |  |






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
| timestamp | [int64](#int64) |  |  |






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
| size | [uint64](#uint64) |  |  |
| amount | [FinancialAmount](#vega.FinancialAmount) |  |  |
| type | [TransferType](#vega.TransferType) |  |  |






<a name="vega.TransferBalance"></a>

### TransferBalance



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| account | [Account](#vega.Account) |  |  |
| balance | [int64](#int64) |  |  |






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
| I1M | 0 |  |
| I5M | 1 |  |
| I15M | 2 |  |
| I1H | 3 |  |
| I6H | 4 |  |
| I1D | 5 |  |



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



<a name="vega.Order.TimeInForce"></a>

### Order.TimeInForce


| Name | Number | Description |
| ---- | ------ | ----------- |
| GTC | 0 |  |
| GTT | 1 |  |
| IOC | 2 |  |
| FOK | 3 |  |



<a name="vega.Order.Type"></a>

### Order.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| LIMIT | 0 | Limit order |
| MARKET | 1 | Market order type |
| NETWORK | 2 | order where the initiating party is the network (used for distressed traders) |



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
| INVALID_PRICE | 17 |  |
| INVALID_SIZE | 18 |  |
| INVALID_PERSISTENCE | 19 |  |



<a name="vega.Side"></a>

### Side


| Name | Number | Description |
| ---- | ------ | ----------- |
| Buy | 0 |  |
| Sell | 1 |  |



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

