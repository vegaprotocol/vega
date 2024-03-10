Feature: Test the mark price update candidate gets rejected if it violates the price monitoring engine bounds.
  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 4s    |

    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 36000   | 0.95        | 1                 |

    And the composite price oracles from "0xCAFECAFE1":
      | name    | price property   | price type   | price decimals |
      | oracle1 | price1.USD.value | TYPE_INTEGER | 0              |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor | sla params      | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance | oracle1 | market type |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures | weight     | 0            | 1           | 20000       | 1,0,0,0        | 5s,20s,20s,1h25m0s         | oracle1 | future      |

  Scenario: 0032-PRIM-040
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 48050        |

    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party            | ETH/FEB23 | sell | 3      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 2      | 18000 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    # leaving opening auction
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15900      | TRADING_MODE_CONTINUOUS | 36000   | 15801     | 15999     | 0            | 0              | 3             |

    When the network moves ahead "1" blocks
    #create traded price 18000 which is outside price monitoring bounds
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 18000 | 0                | TYPE_LIMIT | TIF_GTC |           |

    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    |
      | 15900      | TRADING_MODE_MONITORING_AUCTION |
    When the network moves ahead "2" blocks

    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 18000      | TRADING_MODE_CONTINUOUS | 36000   | 17900     | 18100     | 0            | 0              | 4             |

    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |

    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    |
      | 18000      | TRADING_MODE_MONITORING_AUCTION |
    When the network moves ahead "4" blocks

    #mark price 18000 which is outside the price monitoring bounds is rejected in mark price calculation
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15900      | TRADING_MODE_CONTINUOUS | 36000   | 15801     | 15999     | 0            | 0              | 5             |

