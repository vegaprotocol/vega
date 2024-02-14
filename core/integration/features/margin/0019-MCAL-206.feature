Feature: Amending an order during auction for a party in isolated margin such that they can't cover the order and have all of their orders cancelled
  Background:
    Given the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 6     |
      | market.auction.minimumDuration          | 1     |
    And the price monitoring named "my-price-monitoring-1":
      | horizon | probability | auction extension |
      | 5       | 0.99        | 6                 |

    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.2   | 100         | -100          | 0.2                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring      | data source config     | linear slippage factor | quadratic slippage factor | position decimal places | sla params      |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring-1 | default-eth-for-future | 0.25                   | 0                         | 2                       | default-futures |

  @MCAL206
  Scenario: replicated panic when amending an order during auction
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount       |
      | trader1 | USD   | 100000000000 |
      | trader2 | USD   | 100000000000 |
      | trader3 | USD   | 29340        |
      | trader4 | USD   | 100000000000 |
      | trader5 | USD   | 100000000000 |
      | lprov1  | USD   | 100000000000 |

    And the parties place the following orders with ticks:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/FEB23 | buy  | 1000   | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | trader1 | ETH/FEB23 | buy  | 300    | 15600  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | trader1 | ETH/FEB23 | buy  | 100    | 15700  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | trader1 | ETH/FEB23 | buy  | 300    | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | trader3 | ETH/FEB23 | sell | 300    | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | trader2 | ETH/FEB23 | sell | 300    | 16200  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | trader3 | ETH/FEB23 | sell | 300    | 16800  | 0                | TYPE_LIMIT | TIF_GTC | t3-sell-1 |
      | trader2 | ETH/FEB23 | sell | 1000   | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15800      | TRADING_MODE_CONTINUOUS | 5       | 15701     | 15899     | 0            | 0              | 300           |

    When the parties submit update margin mode:
      | party   | market    | margin_mode     | margin_factor | error |
      | trader3 | ETH/FEB23 | isolated margin | 0.3           |       |
    Then the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | trader3 | ETH/FEB23 | 10680       | 0      | 12816   | 0       | isolated margin | 0.3           | 15120 |

    #order margin: 16800*3*0.3=15120
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general | order margin |
      | trader3 | USD   | ETH/FEB23 | 14220  | 0       | 15120        |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/FEB23 | buy  | 100    | 16200 | 0                | TYPE_LIMIT | TIF_GTC | t3-first  |

    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    |
      | 15800      | TRADING_MODE_MONITORING_AUCTION |

    When the parties amend the following orders:
      | party   | reference | price | tif     | error               |
      | trader3 | t3-sell-1 | 16900 | TIF_GTC | margin check failed |

    When the network moves ahead "7" blockss

    And the orders should have the following status:
      | party   | reference | status        |
      | trader3 | t3-sell-1 | STATUS_ACTIVE |

    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 16200      | TRADING_MODE_CONTINUOUS |

# When the parties amend the following orders:
#   | party   | reference | price | size delta | tif     | error               |
#   | trader3 | t3-sell-1 | 16800 | 200        | TIF_GTC | margin check failed |




