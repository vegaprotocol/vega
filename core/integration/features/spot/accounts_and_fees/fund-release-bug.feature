Feature: replicate the fund releasing bug when market is terminated

  Scenario: 001 terminate market with Oracle
    Given time is updated to "2023-07-20T00:00:00Z"

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 36000   | 0.999       | 23                |

    And the liquidity sla params named "SLA-1":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.1                          | 2                             | 0.2                    |

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
      | market.liquidity.probabilityOfTrading.tau.scaling   | 0.1   |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | SLA-1      |
    And the following network parameters are set:
      | name                                             | value |
      | market.liquidity.providersFeeCalculationTimeStep | 2s    |

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

    Then the network moves ahead "1" blocks
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 0            | 0              | 0             |

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference         | only |
      | party1 | BTC/ETH   | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | buy-party-order1  |      |
      | party1 | BTC/ETH   | buy  | 1      | 13    | 0                | TYPE_LIMIT | TIF_GTC | buy-party-order2  |      |
      | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | buy-party-order3  |      |
      | party2 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | sell-party-order1 |      |
    # | party2 | BTC/ETH   | sell | 1      | 18    | 0                | TYPE_LIMIT | TIF_GTC | sell-party-order2 |      |

    When the network moves ahead "4" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 36000   | 14        | 17        | 0            | 0              |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference         | only |
      | party2 | BTC/ETH   | sell | 1      | 13    | 0                | TYPE_LIMIT | TIF_GTC | sell-party-order3 |      |

    When the network moves ahead "4" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       |
      | 15         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    #cancel the orders that triggered price mon auction
    Then the parties cancel all their orders for the markets:
      | party  | market id |
      | party1 | BTC/ETH   |
      | party2 | BTC/ETH   |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 2000              | 0.4 | submission |

    #lp commits provisions and submits orders
    #other parties enter orders that will cross
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference         |
      | lp1    | BTC/ETH   | buy  | 1      | 13    | 0                | TYPE_LIMIT | TIF_GTC | lp-order1         |
      | party1 | BTC/ETH   | buy  | 1      | 14    | 0                | TYPE_LIMIT | TIF_GTC | buy-party-order4  |
      | party2 | BTC/ETH   | sell | 1      | 14    | 0                | TYPE_LIMIT | TIF_GTC | sell-party-order4 |
      | lp1    | BTC/ETH   | sell | 1      | 13    | 0                | TYPE_LIMIT | TIF_GTC | lp-order2         |

    When the network moves ahead "1" blocks
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       |
      | 15         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    When the market states are updated through governance:
      | market id | state                              |
      | BTC/ETH   | MARKET_STATE_UPDATE_TYPE_TERMINATE |


# When the network moves ahead "4" blocks

# Then the market data for the market "BTC/ETH" should be:
#   | mark price | trading mode            | auction trigger             |
#   | 13         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

# When the network moves ahead "1" blocks

# And the party "lp1" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
# Then the party "lp1" lp liquidity bond account balance should be "2000" for the market "BTC/ETH"
# And the parties should have the following account balances:
#   | party | asset | market id | general |
#   | lp1   | BTC   | BTC/ETH   | 610     |
#   | lp1   | ETH   | BTC/ETH   | 1865    |

# When the market states are updated through governance:
#   | market id | state                              |
#   | BTC/ETH   | MARKET_STATE_UPDATE_TYPE_TERMINATE |

# And the parties should have the following account balances:
#   | party | asset | market id | general |
#   | lp1   | BTC   | BTC/ETH   | 610     |
#   | lp1   | ETH   | BTC/ETH   | 3865    |

# Then "lp1" should have holding account balance of "0" for asset "ETH"
# And the party "lp1" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
# Then the party "lp1" lp liquidity bond account balance should be "0" for the market "BTC/ETH"

# And the network treasury balance should be "0" for the asset "ETH"
# And the global insurance pool balance should be "0" for the asset "ETH"
# And the global insurance pool balance should be "0" for the asset "BTC"