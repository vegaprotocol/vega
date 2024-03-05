Feature: Simple tests for perpetual market mark price.

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.01             | 10s         | 5              |

    And the following network parameters are set:
      | name                                             | value |
      | network.markPriceUpdateMaximumFrequency          | 1s    |
      | network.internalCompositePriceUpdateFrequency    | 1s    |
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

  Scenario: Use last trade for internal TWAP
    When the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals | price type | cash amount | source weights | source staleness tolerance |
      | perp-oracle | ETH   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0                     | 0             | 0                 | 0                 | ETH        | 5                   | weight     | 1000        | 1,0,0,0        | 0s,0s,0s,0s                |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | market type | sla params | price type |
      | ETH/DEC19 | ETH        | ETH   | lqm-params           | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.1                    | 0                         | 5              | 5                       | perp        | SLA        | last trade |
    Given the parties submit the following liquidity provision:
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
      | mark price | trading mode            |
      | 976        | TRADING_MODE_CONTINUOUS |

    And the network moves ahead "2" blocks
    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name           | value | time offset |
      | perp.ETH.value | 140   | -1s         |

    Then the product data for the market "ETH/DEC19" should be:
      | internal twap | external twap |
      | 976           | 140           |

  Scenario: 0053-PERP-033
    When the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals | price type | cash amount | source weights | source staleness tolerance |
      | perp-oracle | ETH   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0                     | 0             | 0                 | 0                 | ETH        | 5                   | weight     | 1000        | 0,1,0,0        | 0s,100s,0s,0s              |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | market type | sla params | price type |
      | ETH/DEC19 | ETH        | ETH   | lqm-params           | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.1                    | 0                         | 5              | 5                       | perp        | SLA        | last trade |
    Given the parties submit the following liquidity provision:
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
      | mark price | trading mode            |
      | 976        | TRADING_MODE_CONTINUOUS |

    And the network moves ahead "2" blocks
    
    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name           | value | time offset |
      | perp.ETH.value | 140   | -1s         |

    Then the product data for the market "ETH/DEC19" should be:
      | internal twap | external twap |
      | 1050          | 140           |

  Scenario: 0053-PERP-034
    When the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals | price type | cash amount | source weights | source staleness tolerance |
      | perp-oracle | ETH   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0                     | 0             | 0                 | 0                 | ETH        | 5                   | weight     | 1000        | 0,1,0,0        | 0s,100s,0s,0s              |
    And the composite price oracles from "0xCAFECAFE2":
      | name    | price property   | price type   | price decimals |
      | oracle1 | prices.ETH.value | TYPE_INTEGER | 5              |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | market type | sla params | price type | source weights | source staleness tolerance | oracle1 |
      | ETH/DEC19 | ETH        | ETH   | lqm-params           | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.1                    | 0                         | 5              | 5                       | perp        | SLA        | weight     | 0,0,1,0        | 0s,0s,24h0m0s,0s           | oracle1 |
    Given the parties submit the following liquidity provision:
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
      | mark price | trading mode            |
      | 976        | TRADING_MODE_CONTINUOUS |

    And the network moves ahead "2" blocks

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name           | value | time offset |
      | perp.ETH.value | 140   | -1s         |

    And the oracles broadcast data with block time signed with "0xCAFECAFE2":
      | name             | value | time offset |
      | prices.ETH.value | 500   | -1s         |

    And the product data for the market "ETH/DEC19" should be:
      | internal twap     | external twap |
      | 1050              | 140           |

    And the network moves ahead "15" blocks

    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            |
      | 500        | TRADING_MODE_CONTINUOUS |

    And the product data for the market "ETH/DEC19" should be:
      | internal twap | external twap |
      | 1050          | 140           |

  Scenario: 0053-PERP-035
    When the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals | price type | cash amount | source weights | source staleness tolerance |
      | perp-oracle | ETH   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0                     | 0             | 0                 | 0                 | ETH        | 5                   | weight     | 0           | 1,1,0,0        | 30s,100s,0s,0s             |
    And the composite price oracles from "0xCAFECAFE2":
      | name    | price property   | price type   | price decimals |
      | oracle1 | prices.ETH.value | TYPE_INTEGER | 5              |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | market type | sla params | price type | source weights | source staleness tolerance | oracle1 |
      | ETH/DEC19 | ETH        | ETH   | lqm-params           | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.1                    | 0                         | 5              | 5                       | perp        | SLA        | weight     | 0,0,1,0        | 0s,0s,24h0m0s,0s           | oracle1 |
    Given the parties submit the following liquidity provision:
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
      | mark price | trading mode            |
      | 976        | TRADING_MODE_CONTINUOUS |

    And the network moves ahead "2" blocks

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name           | value | time offset |
      | perp.ETH.value | 140   | -1s         |

    And the oracles broadcast data with block time signed with "0xCAFECAFE2":
      | name             | value | time offset |
      | prices.ETH.value | 500   | -1s         |

    And the product data for the market "ETH/DEC19" should be:
      | internal twap     | external twap |
      | 1013              | 140           |

    And the network moves ahead "15" blocks

    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            |
      | 500        | TRADING_MODE_CONTINUOUS |

    And the product data for the market "ETH/DEC19" should be:
      | internal twap | external twap |
      | 1013          | 140           |
