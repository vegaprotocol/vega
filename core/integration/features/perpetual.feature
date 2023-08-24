Feature: Simple test creating a perpetual market.

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 18             |
      | USD | 0              |

    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle | ETH   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0                     | 0             | 0                 | 0                 | ETH        | 18                  |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | providers fee calculation time step | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 2592000                             | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | market type | sla params |
      | ETH/DEC19 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.1                    | 0                         | 5              | 5                       | perp        | SLA        |
    And the following network parameters are set:
      | name                                          | value |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
      | market.liquidity.targetstake.triggering.ratio | 0     |
      | market.stake.target.timeWindow                | 10s   |
      | market.stake.target.scalingFactor             | 5     |
      | market.auction.minimumDuration                | 1     |
      | market.fee.factors.infrastructureFee          | 0.001 |
      | market.fee.factors.makerFee                   | 0.004 |
      | market.value.windowLength                     | 60s   |
      | market.liquidityV2.bondPenaltyParameter       | 0.1   |
      | validators.epoch.length                       | 5s    |
      | limits.markets.maxPeggedOrders                | 2     |

    And the average block duration is "1"

    # All parties have 1,000,000.000,000,000,000,000,000 
    # Add as many parties as needed here
    And the parties deposit on asset's general account the following amount:
      | party   | asset | amount                     |
      | lpprov  | ETH   | 10000000000000000000000000 |
      | trader1 | ETH   | 10000000000000000000000000 |
      | trader2 | ETH   | 10000000000000000000000000 |
      | trader3 | ETH   | 10000000000000000000000000 |
      | trader4 | ETH   | 10000000000000000000000000 |
      | trader5 | ETH   | 10000000000000000000000000 |


  @Perpetual
  Scenario: 001 Create a new perp market and leave opening auction in the same way the system tests do
    # the amount ought to be 390,500.000,000,000,000,000,000
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.01  |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size        | minimum visible size | side | pegged reference | volume           | offset | reference   |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | buy  | BID              | 4000000000000000 | 1      | lp-ice-buy  |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | sell | ASK              | 4000000000000000 | 1      | lp-ice-sell |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 269815200000 | 3905000000000000 | 5             |
    And the parties should have the following account balances:
      | party   | asset | market id | margin       | general                   |
      | trader1 | ETH   | ETH/DEC19 | 113402285504 | 9999999999999886597714496 |
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search       | initial      | release      |
      | trader1 | ETH/DEC19 | 94501904587 | 103952095045 | 113402285504 | 132302666421 |
    And debug orders
    And debug detailed orderbook volumes for market "ETH/DEC19"
    # example of how to use the oracle
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.funding.cue | 1511924180 | -100s       |
      | perp.ETH.value   | 975        | -2s         |
      | perp.ETH.value   | 977        | -1s         |


  @Perpetual
  Scenario: 002 Create a new perp market and leave opening auction, then terminate the market through governance
    # the amount ought to be 390,500.000,000,000,000,000,000
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.01  |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size        | minimum visible size | side | pegged reference | volume           | offset | reference   |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | buy  | BID              | 4000000000000000 | 1      | lp-ice-buy  |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | sell | ASK              | 4000000000000000 | 1      | lp-ice-sell |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 269815200000 | 3905000000000000 | 5             |
    And the parties should have the following account balances:
      | party   | asset | market id | margin       | general                   |
      | trader1 | ETH   | ETH/DEC19 | 113402285504 | 9999999999999886597714496 |
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search       | initial      | release      |
      | trader1 | ETH/DEC19 | 94501904587 | 103952095045 | 113402285504 | 132302666421 |
    And debug orders
    And debug detailed orderbook volumes for market "ETH/DEC19"
    # example of how to use the oracle
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.funding.cue | 1511924180 | -100s       |
      | perp.ETH.value   | 975        | -2s         |
      | perp.ETH.value   | 977        | -1s         |

    When the market states are updated through governance:
      | market id | state                              | settlement price |
      | ETH/DEC19 | MARKET_STATE_UPDATE_TYPE_TERMINATE | 976              |
    Then the market state should be "STATE_CLOSED" for the market "ETH/DEC19"

