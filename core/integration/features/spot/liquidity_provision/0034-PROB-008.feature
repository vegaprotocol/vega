Feature: Probability of trading decreases away from the mid-price in Spot market

  Scenario: 001
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 360000  | 0.999       | 300               |

    And the liquidity sla params named "SLA-1":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.9         | 0.1                          | 2                             | 0.2                    |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 1              |
      | BTC | 1              |

    And the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 2s    |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.bondPenaltyParameter               | 0     |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.5   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.2   |
      | market.liquidity.maximumLiquidityFeeFactorLevel     | 0.4   |
      | validators.epoch.length                             | 2s    |
      | limits.markets.maxPeggedOrders                      | 10    |

    Given time is updated to "2023-07-20T00:00:00Z"

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | SLA-1      |
    And the following network parameters are set:
      | name                                             | value |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 100000 |
      | party2 | BTC   | 5000   |
      | lp1    | ETH   | 4000   |
      | lp1    | BTC   | 600    |
      | lp2    | ETH   | 4000   |
      | lp2    | BTC   | 600    |
      | lp3    | ETH   | 4000   |
      | lp3    | BTC   | 600    |

    And the average block duration is "1"

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.2              | 20s         | 0.8            |

    When the spot markets are updated:
      | id      | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | updated-lqm-params   | 0.5                    | 0.5                       |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 2000              | 0.2 | submission |
      | lp2 | lp2   | BTC/ETH   | 2000              | 0.2 | submission |
      | lp3 | lp3   | BTC/ETH   | 2000              | 0.2 | submission |

    And the parties should have the following account balances:
      | party | asset | market id | general |
      | lp1   | BTC   | BTC/ETH   | 600     |
      | lp1   | ETH   | BTC/ETH   | 2000    |

    Then the network moves ahead "1" blocks
    And the network treasury balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "BTC"
    And the party "lp1" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
    Then the party "lp1" lp liquidity bond account balance should be "2000" for the market "BTC/ETH"

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 4800         | 6000           | 0             |

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | only |
      | party1 | BTC/ETH   | buy  | 1      | 12    | 0                | TYPE_LIMIT | TIF_GTC | party-order1 |      |
      | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |      |
      | party2 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order4 |      |
      | party2 | BTC/ETH   | sell | 1      | 19    | 0                | TYPE_LIMIT | TIF_GTC | party-order2 |      |

    When the network moves ahead "1" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 4800         | 6000           | 0             |

    And the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset | reference |
      | lp1   | BTC/ETH   | buy  | 10     | BID              | 10     | lp1-b     |
      | lp1   | BTC/ETH   | sell | 10     | ASK              | 10     | lp1-s     |
      | lp2   | BTC/ETH   | buy  | 10     | BID              | 6      | lp2-b     |
      | lp2   | BTC/ETH   | sell | 10     | ASK              | 6      | lp2-s     |
      | lp3   | BTC/ETH   | buy  | 10     | BID              | 3      | lp3-b     |
      | lp3   | BTC/ETH   | sell | 10     | ASK              | 3      | lp3-s     |

    And the parties should have the following account balances:
      | party | asset | market id | general |
      | lp1   | BTC   | BTC/ETH   | 500     |
      | lp1   | ETH   | BTC/ETH   | 1800    |
      | lp2   | BTC   | BTC/ETH   | 500     |
      | lp2   | ETH   | BTC/ETH   | 1400    |
      | lp3   | BTC   | BTC/ETH   | 500     |
      | lp3   | ETH   | BTC/ETH   | 1100    |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | only |
      | party1 | BTC/ETH   | buy  | 100    | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |      |
      | party2 | BTC/ETH   | sell | 100    | 15    | 1                | TYPE_LIMIT | TIF_GTC | party-order4 |      |

    Then the accumulated liquidity fees should be "3000" for the market "BTC/ETH"

    When the network moves ahead "4" blocks

    #general account with LP fee received
    And the parties should have the following account balances:
      | party | asset | market id | general |
      | lp1   | BTC   | BTC/ETH   | 500     |
      | lp1   | ETH   | BTC/ETH   | 2799    |
      | lp2   | BTC   | BTC/ETH   | 500     |
      | lp2   | ETH   | BTC/ETH   | 2399    |
      | lp3   | BTC   | BTC/ETH   | 500     |
      | lp3   | ETH   | BTC/ETH   | 2099    |

