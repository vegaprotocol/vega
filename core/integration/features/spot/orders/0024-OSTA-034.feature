Feature: Spot market

  Scenario: wash trade on spot market
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 360000  | 0.999       | 3                 |

    And the liquidity sla params named "SLA-1":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.99        | 0.1                          | 2                             | 0.2                    |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 1              |
      | BTC | 1              |

    And the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.liquidity.earlyExitPenalty                   | 0.02  |
      | market.liquidity.earlyExitPenalty                   | 0.5   |
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
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | ETH   | 10000  |
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
      | lpprov | BTC/ETH   | buy  | 12     | 10    | 0                | TYPE_LIMIT | TIF_GTC | lp-order1    |      |
      | party1 | BTC/ETH   | buy  | 1      | 12    | 0                | TYPE_LIMIT | TIF_GTC | party-order1 |      |
      | party1 | BTC/ETH   | buy  | 2      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |      |
      | party2 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order4 |      |
      | party2 | BTC/ETH   | sell | 1      | 19    | 0                | TYPE_LIMIT | TIF_GTC | party-order2 |      |
      | lpprov | BTC/ETH   | sell | 6      | 24    | 0                | TYPE_LIMIT | TIF_GTC | lp-order2    |      |

    When the network moves ahead "3" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 360000  | 10        | 22        | 10           | 1000           | 0             |

    #GTC order partially filled
    And the orders should have the following status:
      | party  | reference    | status        |
      | party1 | party-order3 | STATUS_ACTIVE |

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 15    | 1      |

    #0024-OSTA-034, 0024-OSTA-035
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   | error |
      | party2 | BTC/ETH   | buy  | 1      | 19    | 0                | TYPE_LIMIT | TIF_GTC | self-trade2 |       |

    And the orders should have the following status:
      | party  | reference   | status         |
      | party2 | self-trade2 | STATUS_STOPPED |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   | error |
      | party1 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | self-trade1 |       |

    And the orders should have the following status:
      | party  | reference   | status         |
      | party1 | self-trade1 | STATUS_STOPPED |

    Then the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party2 | party-order2 | 19    | -1         | TIF_GTC |
    #GTT partially filled is canclled by trader
    And the orders should have the following status:
      | party  | reference    | status           |
      | party2 | party-order2 | STATUS_CANCELLED |

    #0024-OSTA-036
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   | error                                                       |
      | party1 | BTC/ETH   | buy  | 1      | 24    | 0                | TYPE_LIMIT | TIF_FOK | FOK-OUTSIDE | OrderError: non-persistent order trades out of price bounds |

    And the orders should have the following status:
      | party  | reference   | status         |
      | party1 | FOK-OUTSIDE | STATUS_STOPPED |

    #0024-OSTA-038
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   | error                                                       |
      | party1 | BTC/ETH   | buy  | 1      | 24    | 0                | TYPE_LIMIT | TIF_IOC | IOC-OUTSIDE | OrderError: non-persistent order trades out of price bounds |

    And the orders should have the following status:
      | party  | reference   | status         |
      | party1 | IOC-OUTSIDE | STATUS_STOPPED |

    #trigger auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   | error |
      | party1 | BTC/ETH   | buy  | 1      | 24    | 0                | TYPE_LIMIT | TIF_GTC | GTC-OUTSIDE |       |
    And the orders should have the following status:
      | party  | reference   | status        |
      | party1 | GTC-OUTSIDE | STATUS_ACTIVE |

    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    #0024-OSTA-037,  Wash trading is allowed on auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   | error |
      | party2 | BTC/ETH   | buy  | 1      | 19    | 0                | TYPE_LIMIT | TIF_GTC | self-trade3 |       |

    And the orders should have the following status:
      | party  | reference   | status        |
      | party2 | self-trade3 | STATUS_ACTIVE |



