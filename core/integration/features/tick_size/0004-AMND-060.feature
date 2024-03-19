Feature: Price change amends specifying a new price which is not an integer multiple of the markets tick size should be rejected and the original order should be left in place

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 18             |
      | USD | 0              |
    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals | source weights | source staleness tolerance |
      | perp-oracle | ETH   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0                     | 0             | 0                 | 0                 | ETH        | 18                  | 1,0,0,0        | 100s,0s,0s,0s              |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.01             | 10s         | 5              |  

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | market type | sla params | tick size |
      | ETH/DEC19 | ETH        | ETH   | lqm-params           | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.1                    | 0                         | 5              | 5                       | perp        | SLA        |     2     |
    And the following network parameters are set:
      | name                                             | value |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | market.auction.minimumDuration                   | 1     |
      | market.fee.factors.infrastructureFee             | 0.001 |
      | market.fee.factors.makerFee                      | 0.004 |
      | market.value.windowLength                        | 60s   |
      | market.liquidity.bondPenaltyParameter            | 0.1   |
      | validators.epoch.length                          | 5s    |
      | limits.markets.maxPeggedOrders                   | 2     |
      | market.liquidity.providersFeeCalculationTimeStep | 5s    |

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

  Scenario: 001 Create a new market, leave opening auction, attempt to amend an order to an invalid price (0004-AMND-060)
    # the amount ought to be 390,500.000,000,000,000,000,000
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size        | minimum visible size | side | pegged reference | volume           | offset | reference   |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | buy  | BID              | 4000000000000000 | 2      | lp-ice-buy  |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | sell | ASK              | 4000000000000000 | 2      | lp-ice-sell |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 902    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC19 | buy  | 5      | 976    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 5      | 976    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 990    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 3905000000000000 | 5             |

    When the parties amend the following orders:
      | party   | reference | price | size delta | tif     | error              |
      | trader1 | t1-b-1    | 903  | 0           | TIF_GTC | invalid OrderError |
    And the network moves ahead "1" blocks

    Then the orders should have the following states:
      | party | market id   | reference   | side | volume | remaining | price | status        |
      | trader1 | ETH/DEC19 | t1-b-1      | buy  | 5       | 5         | 902   | STATUS_ACTIVE |