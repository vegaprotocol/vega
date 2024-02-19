Feature: Reproduce BTC/USDT-PERP market configuration as updated on February 11th 2024

  Background:
    Given time is updated to "2024-02-04T16:00:00Z"
    And the following assets are registered:
      | id    | decimal places |
      | TUSDT | 6              |
    And the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau         | mu    | r | sigma |
      | 0.000001      | 0.000003995 | 0.016 | 0 | 1     |
    And the margin calculator named "margin-calculator":
      | search factor | initial factor | release factor |
      | 1.1           | 1.5            | 1.7            |
    And the fees configuration named "fees":
      | maker fee | infrastructure fee |
      | 0.0001    | 0.0003             |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 360     | 0.9999999   | 120               |
      | 1440    | 0.9999999   | 180               |
      | 4320    | 0.9999999   | 300               |
      | 21600   | 0.9999999   | 86400             |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor | auction extension |
      | lqm-params | 0.9              | 3600s       | 0.05           | 1                 |
    And the liquidity sla params named "sla":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.03        | 0.85                         | 1                             | 0.5                    |
    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle | TUSDT | eth.price           | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.9                   | 0.1095        | -0.0005           | 0.0005            | USDT       | 18                  |
    And the markets:
      | id            | quote name | asset | risk model            | margin calculator | auction duration | fees | price monitoring | data source config | decimal places | position decimal places | linear slippage factor | quadratic slippage factor | sla params | market type |
      | BTC/USDT-PERP | USDT       | TUSDT | log-normal-risk-model | margin-calculator | 3600             | fees | price-monitoring | perp-oracle        | 4              | 1                       | 0.001                  | 0                         | sla        | perp        |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 30    |
      | limits.markets.maxPeggedOrders | 20    |

  Scenario: Check the resulting bounds
    
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount          |
      | party1 | TUSDT | 10000000000000  |
      | party2 | TUSDT | 10000000000000  |
      | aux    | TUSDT | 100000000000000 |
      | aux2   | TUSDT | 100000000000000 |
      | lp     | TUSDT | 100000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id     | commitment amount | fee | lp type    |
      | lp1 | lp    | BTC/USDT-PERP | 90000000          | 0.1 | submission |
      | lp1 | lp    | BTC/USDT-PERP | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id     | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp    | BTC/USDT-PERP | 2         | 1                    | buy  | BID              | 50         | 100    |
      | lp    | BTC/USDT-PERP | 2         | 1                    | sell | ASK              | 50         | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    And the parties place the following orders:
      | party | market id     | side | volume | price      | resulting trades | type       | tif     |
      | aux   | BTC/USDT-PERP | buy  | 1      | 1          | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/USDT-PERP | sell | 1      | 2000000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT-PERP | buy  | 1      | 520000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/USDT-PERP | sell | 1      | 520000000  | 0                | TYPE_LIMIT | TIF_GTC |
    
    When the opening auction period ends for market "BTC/USDT-PERP"
    Then the market data for the market "BTC/USDT-PERP" should be:
      | horizon | ref price | min bound | max bound |
      | 360     | 520000000 | 510725426 | 529437150 |  
      | 1440    | 520000000 | 501610731 | 539039617 |
      | 4320    | 520000000 | 488548773 | 553402620 |
      | 21600   | 520000000 | 452206310 | 597561106 |
