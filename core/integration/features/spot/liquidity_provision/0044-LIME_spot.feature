Feature: Spot market

  @SLABug
  Scenario: party submit liquidity, and amend/cancel it
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
      | 0.99        | 0.1                          | 2                             | 0.2                    |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 1              |
      | BTC | 1              |

    And the following network parameters are set:
      | name                                                  | value |
      | network.markPriceUpdateMaximumFrequency               | 0s    |
      | market.liquidity.earlyExitPenalty                     | 0.02  |
      | market.liquidity.earlyExitPenalty                     | 0.5   |
      | market.liquidity.bondPenaltyParameter                 | 0     |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope   | 0.5   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax     | 0.2   |
      | market.liquidity.maximumLiquidityFeeFactorLevel       | 0.4   |
      | validators.epoch.length                               | 2s    |

    Given time is updated to "2023-07-20T00:00:00Z"

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | SLA-1      |
    And the following network parameters are set:
      | name                                               | value |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | BTC   | 50     |
      | lpprov | ETH   | 4000   |
      | lpprov | BTC   | 60     |

    And the average block duration is "1"

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.2              | 20s         | 0.01           |

    When the spot markets are updated:
      | id      | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | updated-lqm-params   | 0.5                    | 0.5                       |

    # Attempt to submit a liquidity request we do not have enough funds to cover (0080-SPOT-006)
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    | error                                    |
      | lp1 | lpprov | BTC/ETH   | 5000              | 0.1 | submission | not enough collateral in general account |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 1000              | 0.1 | submission |

    Then the network moves ahead "1" blocks
    And the network treasury balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "BTC"
    And the party "lpprov" lp liquidity fee account balance should be "0" for the market "BTC/ETH"

    Then "lpprov" should have general account balance of "3000" for asset "ETH"
    Then "lpprov" should have general account balance of "60" for asset "BTC"
    Then the party "lpprov" lp liquidity bond account balance should be "1000" for the market "BTC/ETH"

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 10           | 1000           | 0             |

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | only |
      | party1 | BTC/ETH   | buy  | 1      | 12    | 0                | TYPE_LIMIT | TIF_GTC | party-order1 |      |
      | party2 | BTC/ETH   | sell | 1      | 19    | 0                | TYPE_LIMIT | TIF_GTC | party-order2 |      |
      | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |      |
      | party2 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order4 |      |
      | lpprov | BTC/ETH   | buy  | 12     | 10    | 0                | TYPE_LIMIT | TIF_GTC | lp-order1    |      |
      | lpprov | BTC/ETH   | sell | 6      | 20    | 0                | TYPE_LIMIT | TIF_GTC | lp-order2    |      |

    When the network moves ahead "3" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 360000  | 10        | 22        | 10           | 1000            | 0             |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type   |
      | lp1 | lpprov | BTC/ETH   | 2000              | 0.1 | amendment |

    Then the network moves ahead "2" blocks
    And the network treasury balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "BTC"
    And the party "lpprov" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
    Then "lpprov" should have holding account balance of "1200" for asset "ETH"
    Then "lpprov" should have general account balance of "800" for asset "ETH"
    Then "lpprov" should have holding account balance of "60" for asset "BTC"
    Then "lpprov" should have general account balance of "0" for asset "BTC"
    Then the party "lpprov" lp liquidity bond account balance should be "2000" for the market "BTC/ETH"

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 20           | 2000           |

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 10    | 12     |
      | buy  | 12    | 1      |
      | sell | 19    | 1      |
      | sell | 20    | 6      |

    Then the network moves ahead "7" blocks
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type   |
      | lp1 | lpprov | BTC/ETH   | 20                | 0.1 | amendment |

    Then the network moves ahead "7" blocks
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 20           | 20             |

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 10    | 12     |
      | buy  | 12    | 1      |
      | sell | 19    | 1      |
      | sell | 20    | 6      |

    # place orders and generate trades to trigger liquidity fee
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | only |
      | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC |           |      |
      | party2 | BTC/ETH   | sell | 1      | 15    | 1                | TYPE_LIMIT | TIF_GTC |           |      |

    Then the network moves ahead "2" blocks
    And the network treasury balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "BTC"
    And the party "lpprov" lp liquidity fee account balance should be "15" for the market "BTC/ETH"
    Then "lpprov" should have holding account balance of "1200" for asset "ETH"
    Then "lpprov" should have general account balance of "2780" for asset "ETH"
    Then "lpprov" should have holding account balance of "60" for asset "BTC"
    Then "lpprov" should have general account balance of "0" for asset "BTC"
    Then the party "lpprov" lp liquidity bond account balance should be "20" for the market "BTC/ETH"

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 20           | 20             |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type   |
      | lp1 | lpprov | BTC/ETH   | 1                 | 0.1 | amendment |

    Then the network moves ahead "7" blocks
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake |
      | 15 | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 20 | 2 |

    And the network treasury balance should be "9" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "BTC"
    And the party "lpprov" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
    Then "lpprov" should have holding account balance of "1200" for asset "ETH"
    Then "lpprov" should have general account balance of "2804" for asset "ETH"

    Then "lpprov" should have holding account balance of "60" for asset "BTC"
    Then "lpprov" should have general account balance of "0" for asset "BTC"
    Then the party "lpprov" lp liquidity bond account balance should be "2" for the market "BTC/ETH"

