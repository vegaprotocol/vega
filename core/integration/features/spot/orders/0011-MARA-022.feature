Feature: Spot market

  Scenario: 0011-MARA-020,0011-MARA-021,0011-MARA-022,0011-MARA-023,0011-MARA-024, GTC, GTT, GFA order in spot market

    Given time is updated to "2023-07-20T00:00:00Z"

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 36000   | 0.999       | 300               |

    And the liquidity sla params named "SLA-1":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.6                          | 2                             | 0.2                    |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 1              |
      | BTC | 1              |

    And the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 2s    |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.7   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.6   |
      | market.liquidity.maximumLiquidityFeeFactorLevel     | 0.4   |
      | validators.epoch.length                             | 10s   |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | SLA-1      |
    And the following network parameters are set:
      | name                                             | value |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | market.liquidity.stakeToCcyVolume                | 1     |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | BTC   | 500    |
      | lp1    | ETH   | 40000  |
      | lp1    | BTC   | 2000   |
      | lp2    | ETH   | 4000   |
      | lp2    | BTC   | 100    |
      | lp3    | ETH   | 4000   |
      | lp3    | BTC   | 100    |
      | lp4    | ETH   | 4000   |
      | lp4    | BTC   | 100    |
      | lp5    | ETH   | 4000   |
      | lp5    | BTC   | 100    |

    And the average block duration is "1"

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.2              | 20s         | 0.8            |

    When the spot markets are updated:
      | id      | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | updated-lqm-params   | 0.5                    | 0.5                       |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 1000              | 0.1 | submission |

    Then the network moves ahead "4" blocks
    And the network treasury balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "BTC"
    And the party "lp1" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
    Then the party "lp1" lp liquidity bond account balance should be "1000" for the market "BTC/ETH"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   | expires in | error                                     |
      | lp1    | BTC/ETH   | buy  | 50     | 4     | 0                | TYPE_LIMIT | TIF_GTC | lp1-b1      |            |                                           |
      | lp1    | BTC/ETH   | buy  | 50     | 5     | 0                | TYPE_LIMIT | TIF_GTT | lp1-b2      | 10         |                                           |
      | lp1    | BTC/ETH   | buy  | 100    | 5     | 0                | TYPE_LIMIT | TIF_GFA | lp1-b3      |            |                                           |
      | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party1-buy  |            |                                           |
      | party2 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |            |                                           |
      | lp2    | BTC/ETH   | sell | 2      | 20    | 0                | TYPE_LIMIT | TIF_GTC | GTC-1       |            |                                           |
      | lp3    | BTC/ETH   | sell | 2      | 22    | 0                | TYPE_LIMIT | TIF_GTT | GTT-1       | 10         |                                           |
      | lp4    | BTC/ETH   | sell | 2      | 24    | 0                | TYPE_LIMIT | TIF_GFN | GFN-1       |            | gfn order received during auction trading |
      | lp5    | BTC/ETH   | sell | 4      | 25    | 0                | TYPE_LIMIT | TIF_GTC | lp2-s       |            |                                           |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    # Then the opening auction period ends for market "BTC/ETH"
    Then "lp1" should have holding account balance of "4500" for asset "ETH"
    Then "lp1" should have general account balance of "34500" for asset "ETH"
    Then "lp3" should have general account balance of "4000" for asset "ETH"
    Then "lp4" should have general account balance of "4000" for asset "ETH"
    Then "lp5" should have general account balance of "4000" for asset "ETH"

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 15    | 1    | party2 |
    Then the parties cancel the following orders:
      | party | reference |
      | lp1   | lp1-b1    |
    Then the orders should have the following status:
      | party | reference | status           |
      | lp1   | lp1-b1    | STATUS_CANCELLED |

    Then "lp1" should have holding account balance of "2500" for asset "ETH"
    Then "lp1" should have general account balance of "36500" for asset "ETH"
    Then the party "lp1" lp liquidity bond account balance should be "1000" for the market "BTC/ETH"

      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | only |
      | party2 | BTC/ETH   | sell | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTT | lp1-b     |      |
      | party1 | BTC/ETH   | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTT | lp1-s     |      |
    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    Then "lp1" should have holding account balance of "2500" for asset "ETH"
    Then "lp1" should have general account balance of "36500" for asset "ETH"
    Then "lp5" should have holding account balance of "40" for asset "BTC"
    Then "lp5" should have general account balance of "4000" for asset "ETH"
    Then "lp5" should have general account balance of "60" for asset "BTC"

    # Then debug trades

