Feature: Spot market SLA

  Scenario: 0044-LIME-045, when target stake > total stake, and LP reduces LP commitment
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
      | 0.001       | 0.1                          | 2                             | 0.2                    |

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

    Given time is updated to "2023-07-20T00:00:00Z"

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | SLA-1      |
    And the following network parameters are set:
      | name                                             | value |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |

    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount |
      | party1  | ETH   | 10000  |
      | party2  | BTC   | 50     |
      | lpprov1 | ETH   | 4000   |
      | lpprov1 | BTC   | 60     |
      | lpprov2 | ETH   | 4000   |
      | lpprov2 | BTC   | 60     |

    And the average block duration is "1"

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.2              | 20s         | 0.8            |

    When the spot markets are updated:
      | id      | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | updated-lqm-params   | 0.5                    | 0.5                       |

    When the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov1 | BTC/ETH   | 2000              | 0.1 | submission |
      | lp2 | lpprov2 | BTC/ETH   | 2000              | 0.1 | submission |

    And the parties should have the following account balances:
      | party   | asset | market id | general |
      | lpprov1 | BTC   | BTC/ETH   | 60      |
      | lpprov1 | ETH   | BTC/ETH   | 2000    |

    Then the network moves ahead "1" blocks
    And the network treasury balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "BTC"
    And the party "lpprov1" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
    Then the party "lpprov1" lp liquidity bond account balance should be "2000" for the market "BTC/ETH"

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 3200         | 4000           | 0             |

    # place orders and generate trades
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference    | only |
      | party1  | BTC/ETH   | buy  | 1      | 12    | 0                | TYPE_LIMIT | TIF_GTC | party-order1 |      |
      | party2  | BTC/ETH   | sell | 1      | 19    | 0                | TYPE_LIMIT | TIF_GTC | party-order2 |      |
      | party1  | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |      |
      | party2  | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order4 |      |
      | lpprov1 | BTC/ETH   | buy  | 6      | 8     | 0                | TYPE_LIMIT | TIF_GTC | lp-order1    |      |
      | lpprov1 | BTC/ETH   | sell | 6      | 24    | 0                | TYPE_LIMIT | TIF_GTC | lp-order2    |      |

    When the network moves ahead "3" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3200         | 4000           | 0             |

    And the parties should have the following account balances:
      | party   | asset | market id | general |
      | lpprov1 | BTC   | BTC/ETH   | 0       |
      | lpprov1 | ETH   | BTC/ETH   | 1520    |

    Then the network moves ahead "9" blocks
    And the network treasury balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "BTC"
    And the party "lpprov1" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
    Then "lpprov1" should have holding account balance of "480" for asset "ETH"
    Then "lpprov1" should have holding account balance of "60" for asset "BTC"
    Then the party "lpprov1" lp liquidity bond account balance should be "2000" for the market "BTC/ETH"

    Then the network moves ahead "7" blocks
    When the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type   |
      | lp1 | lpprov1 | BTC/ETH   | 20                | 0.1 | amendment |

    Then the network moves ahead "7" blocks
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3200         | 2020           |

    # place orders and generate trades to trigger liquidity fee
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | only |
      | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC |           |      |
      | party2 | BTC/ETH   | sell | 1      | 15    | 1                | TYPE_LIMIT | TIF_GTC |           |      |

    Then the network moves ahead "2" blocks
    And the network treasury balance should be "295" for the asset "ETH"

    And the party "lpprov1" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
    And the party "lpprov2" lp liquidity fee account balance should be "14" for the market "BTC/ETH"

    Then "lpprov1" should have holding account balance of "480" for asset "ETH"
    Then "lpprov1" should have holding account balance of "60" for asset "BTC"

    Then the party "lpprov1" lp liquidity bond account balance should be "20" for the market "BTC/ETH"

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3200         | 2020           |

    When the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type   |
      | lp1 | lpprov1 | BTC/ETH   | 1                 | 0.1 | amendment |

    Then the network moves ahead "7" blocks

    And the network treasury balance should be "299" for the asset "ETH"

    And the party "lpprov1" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
    And the party "lpprov2" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
    Then "lpprov1" should have holding account balance of "480" for asset "ETH"
    Then "lpprov1" should have holding account balance of "60" for asset "BTC"

    Then the party "lpprov1" lp liquidity bond account balance should be "2" for the market "BTC/ETH"




