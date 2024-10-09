## Integration test framework setting up markets.

Markets are the cornerstone of integration tests, and are rather complex entities to set up. As such, there are a number of parameters that in turn need to be configured through distinct steps.
These sub-components are:

* [Risk model](#Risk-models)
* [Fee configuration](#Fees-configuration)
* [Oracles for settlement](#Settlement-data)
    * [Settlement data oracles with specific decimal places.](#Oracle-decimal-places)
* [Oracles for termination.](#Trading-termination-oracle)
* [Oracles for perpetual markets](#Perpetual-oracles)
* [Oracles for composite price.](#Composite-price-oracles)
* [Price monitoring parameters](#Price-monitoring)
* [Liquidity SLA parameters](#Liquidity-SLA-parameters)
* [Liquidity monitoring parameters](#Liquidity-monitoring-parameters) [No longer used - DEPRECATED]
* [Margin calculators.](#Margin-calculator)
* [Liquidation strategies.](#Liquidation-strategies)

Arguably not a sub-component, but something that needs to be mentioned in this context:

* [Asset configuration](#Assets)

Before covering the actual configuration of a market, this document will first outline how these parameters can be configured. It's important to note that defaults are available for the following:

* [Fee configuration](#Fees-configuration)
* [Liquidation strats](#Liquidation-strategies)
* Liquidity monitoring [DEPRECATED]
* [Margin calculators](#Margin-calculator)
* [Oracles](#Data-source-configuration) (settlement data, perpetual markets, and market termination).
* [Price monintoring](#Price-monitoring)
* [Risk models](#Risk-models)
* [Liquidity SLA parameters](#Liquidity-SLA-parameters)

The available defaults will be mentioned under their respective sections, along with details on where the provided defaults can be found.

Once a market has been set up, the current market state should also be checked. This can be done through:

* [Market data](#Market-data)
* [Market state](#Market-state)
* [Last market state](#Last-market-state) may sometimes be needed to check markets that have reached a final state.
* [Mark price](#Mark-price)

The market lifecycle is largely determined through oracles. How to use oracles in integration tests is [covered here](oracles.md). Markets can, however be updated or closed through governance. The integration test framework essentially takes the chain and the governance engine out of the equation, but to test market changes through governance, some steps have been provided. These steps are [covered here](governance.md).

### Risk models

There are a number of pre-defined risk models, but if a custom risk model is required, there are steps provided to create one.

#### Pre-configured risk models

The pre-configured risk models can be found under `core/integration/steps/market/defaults/risk-model`. The models themselves are split into _simple_ and _log-normal_ models.
The simple risk models only exist to simplify testing, in practice real markets will only use the _log-normal_ risk models.
Risk models (both pre-configured or manually registered) can then be used in a market definition by name. Manually registered risk models are scoped to the feature file (if defined in the `Background` section), or the `Scenario` if defined there.

The _log-normal_ risk models currently defined are:

* closeout-st-risk-model
* default-log-normal-risk-model
* default-st-risk-model

The _simple_ risk models proveded are:

* system-test-risk-model
* default-simple-risk-model
* default-simple-risk-model-2
* default-simple-risk-model-3
* default-simple-risk-model-4

#### Creating a risk model.

If the pre-configured risk models are not sufficient, or you're testing the impact of changes to a risk model, one or more risk models can be configured using one of the following step:

```cucumber
Given the simple risk model named "my-custom-model":
  | long | short | max move up | min move down | probability of trading |
  | 0.2  | 0.1   | 100         | -100          | 0.1                    |
```

Where the fields are all required and have the following types:

```
| long                   | decimal           |
| short                  | decimal           |
| max move up            | uint              |
| min move down          | int (must be < 0) |
| probability of trading | decimal           |
```

This will register a new simple risk model alongside the pre-existing ones with the name `my-custom-model`. To add a _log-normal_ risk model, the following step should be used:

```cucumber
And the log normal risk model named "custom-logn-model":
  | risk aversion | tau  | mu | r   | sigma |
  | 0.0002        | 0.01 | 0  | 0.0 | 1.2   |
```

Where the fields are all required and have the following types:

```
| risk aversion | decimal |
| tau           | decimal |
| mu            | decimal |
| r             | decimal |
| sigma         | decimal |
```

### Fees configuration

Analogous to risk models, fees configuration are pre-defined, but can be configured for specific tests if needed.

#### Pre-configured fees configuration

 he pre-defined fees configurations can be found in `core/integration/steps/market/defaults/fees-config/`.
Pre-defined fees configurations available are:

* default-none

#### Creating custom fees configuration

To create a fees configuration specific to the feature or scenario, use the following step:

```cucumber
Given the fees configuration named "custom-fee-config":
  | maker fee | infrastructure fee | liquidity fee method | liquidity fee constant | buy back fee | treasury fee |
  | 0.0004    | 0.001              | METHOD_CONSTANT      | 0                      | 0.0001       | 0.00002      |
```

Where the fields are defined as:

```
| maker fee              | required | decimal/string                                  |
| infrastructure fee     | required | decimal/string                                  |
| buy back fee           | optional | decimal/string (default "0")                    |
| treasury fee           | optional | decimal/string (default "0")                    |
| liquidity fee constant | optional | decimal/string                                  |
| liquidity fee method   | optional | LiquidityFeeMethod (default METHOD_UNSPECIFIED) |
```

Details on the [`LiquidityFeeMethod` type](types.md#LiquidityFeeMethod).

### Price monitoring

Again, like risk models and fees config, some price monitoring settings are pre-defined, but custom configuration can be used.

#### Pre-configured price monitoring

The pre-configured price monitoring parameters can be found in `core/integration/steps/market/defaults/price-monitoring`
Available price monitoring settings are:

* default-none
* default-basic

#### Creating custom price monitoring

To create custom price monitoring config, use the following step:

```cucumber
Given the price monitoring named "custom-price-monitoring":
  | horizon | probability | auction extension |
  | 3600    | 0.99        | 3                 |
  | 7200    | 0.95        | 30                |
```

Where fields are required, and the types are:

```
| horizon           | integer (timestamp) |
| probability       | decimal             |
| auction extension | integer (seconds)   |
```

### Liquidity SLA parameters

Again, pre-defined SLA parameters can be used, or custom parameters can be defined.

#### Pre-configured liquidity SLA parameters

The pre-configured liquidity SLA parameters can be found in `core/integration/steps/market/defaults/liquidity-sla-params`
Existing SLA parameters are:

* default-basic
* default-futures
* default-st

#### Creating custom lquidity SLA parameters

To create custom liquidity SLA parameters, use the following step:

```cucumber
Given the liquidity sla params named "custom-sla-params":
  | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
  | 0.5         | 0.6                          | 1                             | 1.0                    |
```

Where the fields are all required and defined as:

```
| price range                   | decimal/string |
| commitment min time fraction  | decimal/string |
| performance hysteresis epochs | int64          |
| sla competition factor        | decimal        |
```

### Liquidity monitoring parameters

Liquidity auctions have been deprecated. The defaults and steps have not yet been removed as they are referenced by some tests, but shouldn't be used in new tests.

### Margin calculator

Margin calculators are provided as pre-configured settings, but can be customised like risk, SLA, price monitoring, etc...

#### Pre-configured margin calculators

The pre-configured margin calculaters can be found in `core/integration/steps/market/defaults/margin-calculator`
Existing margin calculators are:

* default-capped-margin-calculator
* default-margin-calculator
* default-overkill-margin-calculator

#### Creating custom margn calculator

To create a custom margin calculator, use the following step:

```cucumber
Given the margin calculator named "custom-margin-calculator":
  | search factor | initial factor | release factor |
  | 1.2           | 1.5            | 1.7            |
```

All fields are required, and defined as `decimal`

### Liquidation strategies

As any market sub-configuration type, there are pre-defined liquidation strategies, and custom strategies can be created:

#### Pre-configured liquidtaion strategies

The pre-configured liquidation strategies can be found in `core/integration/steps/market/defaults/liquidation-config`
Existing liquidation strategies are:

* AC-013-strat
* default-liquidation-strat
* legacy-liquidation-strategy
* slow-liquidation-strat

#### Creating custom liquidation strategies

Several liquidation strategies can be created using the following step:

```cucumber
Given the liquidation strategies:
  | name                  | disposal step | disposal fraction | full disposal size | max fraction consumed | disposal slippage range |
  | cumstom-liquidation-1 | 10            | 0.1               | 20                 | 0.05                  | 0.5                     |
  | cumstom-liquidation-2 | 20            | 0.2               | 50                 | 0.02                  | 0.4                     |
```

All fields are required and defined as follows:

```
| name                    | string                      |
| disposal step           | int64 (duration in seconds) |
| disposal fraction       | decimal                     |
| full disposal size      | uint64                      |
| max fraction consumed   | decimal                     |
| disposal slippage range | decimal                     |
```

### Data source configuration

Markets rely on oracles for things like settlement price data, or trading temrination (future markets specifically). To create a market, an oracle needs to be configured. quite often a default oracle can be used, but if the test needs to control the oracle, a custom oracle _must_ be configured.

#### Pre-configured oracles

Pre-configured oracles can be found in `core/integration/steps/market/defaults/oracle-config`
Existing oracles are:

* default-eth-for-future
* default-eth-for-perps
* default-dai-for-future
* default-dai-for-perps
* default-usd-for-future
* default-usd-for-perps

#### Creating custom oracles

Creating a custom oracle requires a bit more work:

##### Settlement data

To create an oracle for settlement data, use the following step:

```cucumber
Given the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "myOracle":
  | property         | type         | binding         | decimals | condition             | value |
  | prices.ETH.value | TYPE_INTEGER | settlement data | 0        | OPERATOR_GREATER_THAN | 0     |
```

Where the fields are defined as:

```
| property  | required                                | string                   |
| type      | required                                | PropertyKey_Type         |
| binding   | required                                | string                   |
| decimals  | optional                                | uint64                   |
| condition | optional                                | Condition_Operator       |
| value     | optional (required if condition is set) | string (must match type) |
```

Details on the [`PropertyKey_Type` type](types.md#PropertyKey_Type).
Details on the [`Condition_Operator` type](types.md#Condition_Operator).

##### Trading termination oracle

The same inputs are used for trading termination bindings, but the step looks like this:

```cucmber
And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "myOracle":
  | property           | type         | binding             |
  | trading.terminated | TYPE_BOOLEAN | trading termination |
```

Note that the `from` and `named` section of the step matches. This is required.

##### Oracle decimal places

Oracles feed data to the system from an external source. The asset used might have 18 decimal places (e.g. ETH), the market could be limited to 10 decimals, and the oracle in turn could supply price data with 12 decimal places. To mimic this, the number of decimal for the oracle can be set using the following step:

```cucumber
And the settlement data decimals for the oracle named "myOracle" is given in "1" decimal places
```

## Creating a basic futures market

With these components covered/set up, a basic market can now be set up using the step:

```cucumber
Given the markets:
  | id        | quote name | asset | risk model        | margin calculator        | auction duration | fees              | price monitoring        | data source config | linear slippage factor | quadratic slippage factor | sla params        | liquidation-strategy |
  | ETH/MAR24 | ETH        | ETH   | custom-logn-model | custom-margin-calculator | 1                | custom-fee-config | custom-price-monitoring | myOracle           | 1e0                    | 0                         | custom-sla-params | custom-liquidation-1 |
```

Note that, when using one of the pre-configured data sources that has a perps counterpart, the test is expected to pass both as a future or a perpetual market. Should this not be the case for whatever reason, the test should be tagged with the `@NoPerp` tag.

Market configuration is extensive, and can be configured with a myriad of additional (optional) settings. The full list of fields is as follows:

```
| field name                 | required | type                                          | deprecated |
|----------------------------|----------|-----------------------------------------------|------------|
| id                         | yes      | string                                        |            |
| quote name                 | yes      | string                                        |            |
| asset                      | yes      | string                                        |            |
| risk model                 | yes      | string (risk model name)                      |            |
| fees                       | yes      | string (fees-config name)                     |            |
| data source config         | yes      | string (oracle name)                          |            |
| price monitoring           | yes      | string (price monitoring name)                |            |
| margin calculator          | yes      | string (margin calculator name)               |            |
| auction duration           | yes      | int64 (opening auction duration in ticks)     |            |
| linear slippage factor     | yes      | float64                                       |            |
| quadratic slippage factor  | yes      | float64                                       | yes        |
| sla params                 | yes      | string (sla params name)                      |            |
| decimal places             | no       | integer (default 0)                           |            |
| position decimal places    | no       | integer (default 0)                           |            |
| liquidity monitoring       | no       | string (name of liquidity monitoring)         | yes        |
| parent market id           | no       | string (ID of other market)                   |            |
| insurance pool fraction    | no       | decimal (default 0)                           |            |
| successor auction          | no       | int64 (duration in seconds)                   |            |
| is passed                  | no       | boolean                                       | not used   |
| market type                | no       | string (perp for perpetual market)            |            |
| liquidation strategy       | no       | string (liquidation strategy name)            |            |
| price type                 | no       | Price_Type                                    |            |
| decay weight               | no       | decimal (default 0)                           |            |
| decay power                | no       | decimal (default 0)                           |            |
| cash amount                | no       | uint (default 0)                              |            |
| source weights             | no       | Source_Weights (default 0,0,0,0)              |            |
| source staleness tolerance | no       | Staleness_Tolerance (default 1us,1us,1us,1us) |            |
| oracle1                    | no       | string (composite price oracle name)          |            |
| oracle2                    | no       | string (composite price oracle name)          |            |
| oracle3                    | no       | string (composite price oracle name)          |            |
| oracle4                    | no       | string (composite price oracle name)          |            |
| oracle5                    | no       | string (composite price oracle name)          |            |
| tick size                  | no       | uint (default 1)                              |            |
| max price cap              | no       | uint                                          |            |
| binary                     | no       | boolean (if true, max price cap is required)  |            |
| fully collateralised       | no       | boolean (if true, max price cap is required)  |            |
|----------------------------|----------|-----------------------------------------------|------------|
```

Details on the [`Price_Type` type](types.md#Price-type).
Details on the [`Source_Weights` type](types.md#Source-weights)
Details on the [`Staleness_Tolerance` type](types.md#Staleness-tolerance)

## Optional market config components

As seen in the table above, there are certain optional parameters that haven't been covered yet. Most notably composite price oracles, oracles for perpetual markets, and assets.

### Composite price oracles

There are no default composite price oracles provided, the only way to create one is to define one or more oracles using the following step:

```cucumber
Given the composite price oracles from "0xCAFECAFE1":
  | name        | price property    | price type   | price decimals |
  | composite-1 | price.USD.value   | TYPE_INTEGER | 0              |
  | composite-2 | price.USD.value.2 | TYPE_INTEGER | 0              |
```

Where the fields are defined as follows:

```
| name           | required | string           |
| price property | required | string           |
| type           | required | PropertyKey_Type |
| price decimals | optional | int              |
```

Details on [`PropertyKey_Type` type](types.md#PropertyKey_Type).

### Perpetual oracles

The pre-existing oracles have been covered as part of the data source config section. To create a custom perpetual oracle, a custom oracle can be created using the following step:

```cucumber
Given the perpetual oracles from "0xCAFECAFE1":
  | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals | source weights | source staleness tolerance | price type |
  | perp-oracle | ETH   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0                     | 0             | 0                 | 0                 | ETH        | 18                  | 1,0,0,0        | 100s,0s,0s,0s              | weight     |
```

Where the fields are defined as follows:

```
| field                       | required | type                                                  |
|-----------------------------|----------|-------------------------------------------------------|
| name                        | yes      | string                                                |
| asset                       | yes      | string (asset ID)                                     |
| settlement property         | yes      | string                                                |
| settlement type             | yes      | PropertyKey_Type                                      |
| schedule property           | yes      | string                                                |
| schedule type               | yes      | PropertyKey_Type                                      |
| settlement decimals         | no       | uint64                                                |
| margin fundgin factor       | no       | decimal (default 0)                                   |
| interest rate               | no       | decimal (default 0)                                   |
| clamp lower bound           | no       | decimal (default 0)                                   |
| clamp upper bound           | no       | decimal (default 0)                                   |
| quote name                  | no       | string (asset ID)                                     |
| funding rate scaling factor | no       | decimal                                               |
| funding rate lower bound    | no       | decimal                                               |
| funding rate upper bound    | no       | decimal                                               |
| decay weight                | no       | decimal (default 0)                                   |
| decay power                 | no       | decimal (default 0)                                   |
| cash amount                 | no       | decimal                                               |
| source weights              | no       | Source_Weights                                        |
| source staleness tolerance  | no       | Staleness_Tolerance (default 1000s,1000s,1000s,1000s) |
| price type                  | no       | Price_Type                                            |
```

Details on the [`Price_Type` type](types.md#Price-type).
Details on the [`Source_Weights` type](types.md#Source-weights)
Details on the [`Staleness_Tolerance` type](types.md#Staleness-tolerance)

### Assets

It is not required to define an asset prior to using it in a market. If you create a market with an non-existing asset, the asset will be created ad-hoc, with the same number of decimal places as the market. As mentioned earlier, though, actual markets may have less decimal places than the asset they use. To make it possible to test whether or not the system handles these scenario's as expected, it's possible to configure an asset with a specific number of decimal places using the following step:

```cucumber
Given the following assets are registered:
  | id   | decimal places | quantum |
  | ETH  | 18             | 10      |
  | DAI  | 18             | 1       |
  | USDT | 10             | 2.3     |
```

Where the fields are defined as follows:

```
| id             | string  | required |
| decimal places | uint64  | required |
| quantum        | decimal | optional |
```

## Checking the market state

### Market data

The most comprehensive way to check the market state is by checking the last `MarketData` event. This is done using the following step

```cucumber
Then the market data for the market "ETH/MAR24" should be:
  | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
  | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 100000         | 1             |
```

All fields are treated as optional, and are defined as follows:

```
| market                   | string                      |
| best bid price           | Uint                        |
| best bid volume          | uint64                      |
| best offer price         | Uint                        |
| best offer volume        | uint64                      |
| best static bid price    | Uint                        |
| best static bid volume   | uint64                      |
| best static offer price  | Uint                        |
| best static offer volume | uint64                      |
| mid price                | Uint                        |
| static mid price         | Uint                        |
| mark price               | Uint                        |
| last traded price        | Uint                        |
| timestamp                | int64 (timestamp)           |
| open interest            | uint64                      |
| indicative price         | Uint                        |
| indicative volume        | uint64                      |
| auction start            | int64 (timestamp)           |
| auction end              | int64 (timestamp)           |
| trading mode             | Market_TradingMode          |
| auction trigger          | AuctionTrigger              |
| extension trigger        | AuctionTrigger              |
| target stake             | Uint                        |
| supplied stake           | Uint                        |
| horizon                  | int64 (duration in seconds) |
| ref price                | Uint (reference price)      |
| min bound                | Uint                        |
| max bound                | Uint                        |
| market value proxy       | Uint                        |
| average entry valuation  | Uint                        |
| party                    | string                      |
| equity share             | decimal                     |
```

Details on the [`Market_TradingMode` type](types.md#Trading-mode)
Details on the [`AuctionTrigger` type](types.md#Auction-trigger)

### Market state

To check what the current `Market_state` of a given market is, the following step should be used:

```cucumber
Then  the market state should be "STATE_ACTIVE" for the market "ETH/DEC22"
```

Details on the [`Market_State` type](types.md#Market-state)

### Last market state

Similarly to checking the market state for an active market, should a market have settled or succeeded, we can check the last market state event pertaining to that market using:

```cucumber
Then the last market state should be "STATE_CANCELLED" for the market "ETH/JAN23"
```

Details on the [`Market_State` type](types.md#Market-state)

### Mark price

To quickly check what the current mark price is:

```cucumber
Then the mark price should be "1000" for the market "ETH/FEB23"
```
