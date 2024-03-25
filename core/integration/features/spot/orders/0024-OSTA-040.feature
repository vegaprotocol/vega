Feature: 0037-OPEG-020, 0037-OPEG-021

  Background:
    Given the following network parameters are set:
      | name                                                | value |
      | market.liquidity.bondPenaltyParameter               | 1     |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | limits.markets.maxPeggedOrders                      | 6     |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.stakeToCcyVolume                   | 1.0   |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.19  |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 1     |

    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.1              | 24h         | 1              |

    And the average block duration is "1"
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.01      | 0.03               |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600    | 0.999       | 1                 |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.01        | 0.5                          | 1                             | 1.0                    |

  Scenario: check invalid order types in spot market
    #while a sell order pegged to the mid price should have it's price rounded down to the nearest market tick size
    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | liquidity monitoring | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | lqm-params           | 2              | 2                       | default-basic |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | ETH   | 100000000 |
      | party3 | ETH   | 2000      |
      | party4 | ETH   | 100000000 |
      | party4 | BTC   | 10000     |
    And the average block duration is "1"
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | party1 | BTC/ETH   | 10000             | 0.001 | submission |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | BTC/ETH   | buy  | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b1      |
      | party3 | BTC/ETH   | buy  | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | p3b2      |
      | party4 | BTC/ETH   | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | p4s2      |
      | party4 | BTC/ETH   | sell | 1000   | 1900  | 0                | TYPE_LIMIT | TIF_GTC | p4s1      |

    Then the network moves ahead "2" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1100       | TRADING_MODE_CONTINUOUS | 3600    | 1055      | 1147      | 10000        | 10000          | 0             |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party3 | 1100  | 10   | party4 |

    #0024-OSTA-047: Order reason of `ORDER_ERROR_INSUFFICIENT_ASSET_BALANCE` user does not have enough of the asset or does not have an account at all
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error                                                              |
      | party3 | BTC/ETH   | buy  | 100    | 1160  | 0                | TYPE_LIMIT | TIF_IOC | party does not have sufficient balance to cover the trade and fees |

    #0024-OSTA-048:self trade should be stopped
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error | reference |
      | party3 | BTC/ETH   | buy  | 1      | 1120  | 0                | TYPE_LIMIT | TIF_GFN |       | p3-b      |
      | party3 | BTC/ETH   | sell | 1      | 1120  | 0                | TYPE_LIMIT | TIF_GFN |       | p3-s      |

    And the orders should have the following status:
      | party  | reference | status         |
      | party3 | p3-b      | STATUS_ACTIVE  |
      | party3 | p3-s      | STATUS_STOPPED |

    #trigger auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 1      | 1160  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | BTC/ETH   | sell | 1      | 1160  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    |
      | 1100       | TRADING_MODE_MONITORING_AUCTION |

    #0024-OSTA-040:IOC; 0024-OSTA-041:FOK; 0024-OSTA-042:GFN; 0024-OSTA-046:order with invalid market ID,
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error                                     |
      | party3 | BTC/ETH   | buy  | 1      | 1160  | 0                | TYPE_LIMIT | TIF_IOC | ioc order received during auction trading |
      | party3 | BTC/ETH   | buy  | 1      | 1160  | 0                | TYPE_LIMIT | TIF_FOK | fok order received during auction trading |
      | party3 | BTC/ETH   | buy  | 1      | 1160  | 0                | TYPE_LIMIT | TIF_GFN | gfn order received during auction trading |
      | party3 | ETH/DEC20 | buy  | 1      | 1160  | 0                | TYPE_LIMIT | TIF_GFN | OrderError: Invalid Market ID             |
