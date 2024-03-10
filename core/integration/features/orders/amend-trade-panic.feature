Feature: Amending orders with isolated margins should never panic
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
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring-1 | default-eth-for-future | 0.1                    | 0                         | 2                       | default-futures |

  @AmendEdge
  Scenario: Edge-case 1: Amending an order will cause it to trade in full, but leaves the party with insufficient margin to maintain its position. The old order should be restored, without any trades beign made
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount       |
      | trader1 | USD   | 100000000000 |
      | trader2 | USD   | 100000000000 |
      | trader3 | USD   | 100000000000 |
      | trader4 | USD   | 100000000000 |
      | trader5 | USD   | 100000000000 |
      | lprov1  | USD   | 100000000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lprov1 | ETH/FEB23 | 1000              | 0.1 | submission |

    And the parties place the following orders with ticks:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/FEB23 | buy  | 1000   | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | trader1 | ETH/FEB23 | buy  | 300    | 15600  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lprov1  | ETH/FEB23 | buy  | 100    | 15700  | 0                | TYPE_LIMIT | TIF_GTC | lp-buy-1  |
      | trader3 | ETH/FEB23 | buy  | 300    | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lprov1  | ETH/FEB23 | sell | 300    | 15800  | 0                | TYPE_LIMIT | TIF_GTC | lp-sell-1 |
      | trader2 | ETH/FEB23 | sell | 600    | 15802  | 0                | TYPE_LIMIT | TIF_GTC | t2-sell-1 |
      | trader2 | ETH/FEB23 | sell | 300    | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | trader2 | ETH/FEB23 | sell | 1000   | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15800      | TRADING_MODE_CONTINUOUS | 5       | 15701     | 15899     | 0            | 1000           | 300           |

    When the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset | reference |
      | lprov1 | ETH/FEB23 | buy  | 100    | BID              | 10     | buy_peg_1 |
      | lprov1 | ETH/FEB23 | buy  | 200    | BID              | 20     | buy_peg_2 |

    Then the parties should have the following margin levels:
      | party   | market id | maintenance | margin mode  | margin factor | order |
      | lprov1  | ETH/FEB23 | 14220       | cross margin | 0             | 0     |
      | trader1 | ETH/FEB23 | 20540       | cross margin | 0             | 0     |
      | trader3 | ETH/FEB23 | 9480        | cross margin | 0             | 0     |
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general     | bond |
      | lprov1  | USD   | ETH/FEB23 | 17064  | 99999981936 | 1000 |
      | trader1 | USD   | ETH/FEB23 | 23496  | 99999976504 |      |
      | trader3 | USD   | ETH/FEB23 | 11376  | 99999988624 |      |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader3 | ETH/FEB23 | buy  | 100    | 15500 | 0                | TYPE_LIMIT | TIF_GTC | t3-to-amend |
    Then the parties should have the following margin levels:
      | party   | market id | maintenance | margin mode  | margin factor | order |
      | trader3 | ETH/FEB23 | 11060       | cross margin | 0             | 0     |
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general     |
      | trader3 | USD   | ETH/FEB23 | 13272  | 99999986728 |

    When the parties submit update margin mode:
      | party   | market    | margin_mode     | margin_factor | error |
      | trader3 | ETH/FEB23 | isolated margin | 0.3           |       |
    Then the parties should have the following margin levels:
      | party   | market id | maintenance | margin mode     | margin factor | order |
      | trader3 | ETH/FEB23 | 9480        | isolated margin | 0.3           | 4650  |
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general     |
      | trader3 | USD   | ETH/FEB23 | 14220  | 99999981130 |

    When the parties withdraw the following assets:
      | party   | asset | amount      | error |
      | trader3 | USD   | 99999981130 |       |
    Then the parties should have the following margin levels:
      | party   | market id | maintenance | release | margin mode     | margin factor | order |
      | trader3 | ETH/FEB23 | 9480        | 0       | isolated margin | 0.3           | 4650  |
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general | order margin |
      | trader3 | USD   | ETH/FEB23 | 14220  | 0       | 4650         |

    When the parties amend the following orders:
      | party   | reference   | price | size delta | tif     | error               |
      | trader3 | t3-to-amend | 15805 | 400        | TIF_GTC | margin check failed |
    Then the parties should have the following margin levels:
      | party   | market id | maintenance | margin mode     | margin factor | order |
      | trader3 | ETH/FEB23 | 9480        | isolated margin | 0.3           | 0     |
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | USD   | ETH/FEB23 | 14220  | 4650    |

    And debug detailed orderbook volumes for market "ETH/FEB23"
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader5 | ETH/FEB23 | buy  | 90     | 15802 | 1                | TYPE_LIMIT | TIF_GTC |           |
    Then the following trades should be executed:
      | buyer   | seller  | price | size |
      | trader5 | trader2 | 15802 | 90   |
    And debug detailed orderbook volumes for market "ETH/FEB23"
