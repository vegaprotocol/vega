Feature: Spot market

  Scenario: 0011-MARA-025,0011-MARA-026,0011-MARA-027,0011-MARA-031,0011-MARA-032, GTC, GTT, GFA, GFN pegged order in spot market

    Given time is updated to "2023-07-20T00:00:00Z"

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 72000   | 0.999       | 2                 |

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
      | limits.markets.maxPeggedOrders                   | 10    |

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

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   | expires in | error |
      | lp1    | BTC/ETH   | buy  | 50     | 4     | 0                | TYPE_LIMIT | TIF_GTC | lp1-b1      |            |       |
      | lp1    | BTC/ETH   | buy  | 20     | 12    | 0                | TYPE_LIMIT | TIF_GTT | lp1-b2      | 10         |       |
      | lp1    | BTC/ETH   | buy  | 10     | 12    | 0                | TYPE_LIMIT | TIF_GFA | lp1-b3      |            |       |
      | party1 | BTC/ETH   | buy  | 2      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party1-buy  |            |       |
      | party2 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |            |       |
      | lp2    | BTC/ETH   | sell | 2      | 16    | 0                | TYPE_LIMIT | TIF_GTT | GTT-1       | 3          |       |
      | lp2    | BTC/ETH   | sell | 2      | 17    | 0                | TYPE_LIMIT | TIF_GTC | GTC-1       |            |       |
      | lp5    | BTC/ETH   | sell | 4      | 25    | 0                | TYPE_LIMIT | TIF_GTC | lp2-s       |            |       |

    And the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset | reference |
      | lp3   | BTC/ETH   | sell | 6      | ASK              | 3      | lp3-peg-2 |

    #pegged GTT order is parked during auction
    Then the orders should have the following status:
      | party | reference | status        |
      | lp3   | lp3-peg-2 | STATUS_PARKED |
    Then the parties cancel the following orders:
      | party | reference |
      | lp3   | lp3-peg-2 |

    Then the network moves ahead "1" blocks
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake |
      | 15         | TRADING_MODE_CONTINUOUS | 72000   | 13        | 18        | 800          | 1000           |

    Then "lp1" should have holding account balance of "4400" for asset "ETH"
    Then "lp1" should have general account balance of "34600" for asset "ETH"
    Then "lp3" should have general account balance of "4000" for asset "ETH"
    Then "party1" should have holding account balance of "150" for asset "ETH"
    Then "party1" should have general account balance of "9700" for asset "ETH"
    Then "party1" should have general account balance of "10" for asset "BTC"
    Then "party2" should have general account balance of "490" for asset "BTC"

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 15    | 1    | party2 |

    And the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset | reference |
      | lp3   | BTC/ETH   | sell | 6      | ASK              | 3      | lp3-peg-1 |

    Then the orders should have the following status:
      | party | reference | status        |
      | lp3   | lp3-peg-1 | STATUS_ACTIVE |
    Then "lp3" should have general account balance of "4000" for asset "ETH"

    #pegged on GTT order GTC-1
    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 19    | 6      |

    Then the network moves ahead "2" blocks
    Then the orders should have the following status:
      | party | reference | status           |
      | lp1   | lp1-b3    | STATUS_CANCELLED |
      | lp2   | GTT-1     | STATUS_EXPIRED   |
      | lp3   | lp3-peg-1 | STATUS_ACTIVE    |

    Then "lp3" should have general account balance of "4000" for asset "ETH"

    #when GTT order expired then pegged on GTC order GTC-1
    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 20    | 6      |

    Then the network moves ahead "1" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | only | error |
      | party2 | BTC/ETH   | sell | 10     | 12    | 0                | TYPE_LIMIT | TIF_GTC | party2-s  |      |       |

    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    #pegged GTC order is parked during auction
    Then the orders should have the following status:
      | party | reference | status        |
      | lp3   | lp3-peg-1 | STATUS_PARKED |

    Then the network moves ahead "3" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference | expires in | error |
      | lp1   | BTC/ETH   | buy  | 10     | 14    | 0                | TYPE_LIMIT | TIF_GFN | lp1-b3    |            |       |

    And the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset | reference |
      | lp3   | BTC/ETH   | buy  | 6      | BID              | 1      | lp3-peg-2 |

    #pegged GFN order
    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 13    | 6      |

    Then "lp3" should have holding account balance of "780" for asset "ETH"
