Feature: Mark price calculation on auction exit

  Background:
    Given the average block duration is "1"
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 60      | 0.99        | 10                |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures | weight     | 0            | 0           | 100         | 0,1,0          | 1m0s,1m0s,1m0s             |

  Scenario: Order book price set to indicative price during auctions and mark price calcualted on auction exit (0009-MRKP-024)(0009-MRKP-025)(0009-MRKP-026)(0009-MRKP-027)
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | aux1             | USD   | 100000000000 |
      | aux2             | USD   | 100000000000 |
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14970 | 0                | TYPE_LIMIT | TIF_GTC | bestBid   |
      | aux1             | ETH/FEB23 | buy  | 1      | 15000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2             | ETH/FEB23 | sell | 1      | 15000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 15090 | 0                | TYPE_LIMIT | TIF_GTC | bestAsk   |
    When the opening auction period ends for market "ETH/FEB23"
    Then the mark price should be "15030" for the market "ETH/FEB23"

    When the network moves ahead "6" blocks
    Then the mark price should be "15030" for the market "ETH/FEB23"

    Given the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 15030      | TRADING_MODE_CONTINUOUS | 60      | 14900     | 15100     |
    And the parties amend the following orders:
      | party            | reference | price | size delta | tif     |
      | sellSideProvider | bestAsk   | 15190 | 0          | TIF_GTC |
      | buySideProvider  | bestBid   | 15070 | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | aux1  | ETH/FEB23 | buy  | 1      | 15110 | 0                | TYPE_LIMIT | TIF_GTC | auctionBid |
      | aux2  | ETH/FEB23 | sell | 1      | 15110 | 0                | TYPE_LIMIT | TIF_GTC | auctionAsk |
    When the network moves ahead "10" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    |
      | 15030      | TRADING_MODE_MONITORING_AUCTION |
    Given the parties amend the following orders:
      | party | reference  | price | size delta | tif     |
      | aux1  | auctionBid | 15100 | 0          | TIF_GTC |
      | aux2  | auctionAsk | 15100 | 0          | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 15108      | TRADING_MODE_CONTINUOUS |









