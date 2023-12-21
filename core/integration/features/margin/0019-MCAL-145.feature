Feature:  pegged order in isoalted margin is not supported during auction
    Background:
        Given the following network parameters are set:
            | name                                    | value |
            | network.markPriceUpdateMaximumFrequency | 0s    |
            | limits.markets.maxPeggedOrders          | 6     |
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

        And the following network parameters are set:
            | name                           | value |
            | market.auction.minimumDuration | 1     |
        Given the average block duration is "1"

    Scenario: 001 (0019-MCAL-145) A market in auction and party with a partially filled pegged order switches from cross margin mode to isolated margin mode the unfilled portion of the pegged order is cancelled
        Given the parties deposit on asset's general account the following amount:
            | party            | asset | amount       |
            | buySideProvider  | USD   | 100000000000 |
            | sellSideProvider | USD   | 100000000000 |
            | party1           | USD   | 158550       |
            | aux              | USD   | 1585510000   |

        When the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee | lp type    |
            | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

        And the parties place the following orders:
            | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
            | buySideProvider  | ETH/FEB23 | buy  | 1000   | 15600  | 0                | TYPE_LIMIT | TIF_GTC |           |
            | buySideProvider  | ETH/FEB23 | buy  | 100    | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
            | buySideProvider  | ETH/FEB23 | buy  | 100    | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
            | sellSideProvider | ETH/FEB23 | sell | 100    | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
            | sellSideProvider | ETH/FEB23 | sell | 100    | 15810  | 0                | TYPE_LIMIT | TIF_GTC |           |
            | sellSideProvider | ETH/FEB23 | sell | 1000   | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

        When the network moves ahead "2" blocks
        And the market data for the market "ETH/FEB23" should be:
            | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
            | 15800      | TRADING_MODE_CONTINUOUS | 5       | 15701     | 15899     | 0            | 1000           | 100           |

        #now we try to place short pegged order which does offset the current short position, order margin should be 0
        When the parties place the following pegged orders:
            | party  | market id | side | volume | pegged reference | offset | reference |
            | party1 | ETH/FEB23 | buy  | 200    | BID              | 10     | buy_peg_1 |
            | party1 | ETH/FEB23 | buy  | 200    | BID              | 100    | buy_peg_2 |
        And the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux   | ETH/FEB23 | sell | 200    | 15790 | 2                | TYPE_LIMIT | TIF_GTC |

        And the following trades should be executed:
            | buyer  | price | size | seller |
            | party1 | 15790 | 100  | aux    |

        Then the orders should have the following status:
            | party  | reference | status        |
            | party1 | buy_peg_1 | STATUS_ACTIVE |
            | party1 | buy_peg_2 | STATUS_ACTIVE |

        And the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux   | ETH/FEB23 | sell | 2      | 15600 | 0                | TYPE_LIMIT | TIF_GTC |

        And the market data for the market "ETH/FEB23" should be:
            | mark price | trading mode                    |
            | 15800      | TRADING_MODE_MONITORING_AUCTION |

        And the parties submit update margin mode:
            | party  | market    | margin_mode     | margin_factor | error |
            | party1 | ETH/FEB23 | isolated margin | 0.5           |       |

        Then the orders should have the following status:
            | party  | reference | status           |
            | party1 | buy_peg_1 | STATUS_CANCELLED |
            | party1 | buy_peg_2 | STATUS_CANCELLED |

    Scenario: 002 (0019-MCAL-146) A market in an auction and party with a partially filled iceberg pegged order switches from cross margin mode to isolated margin mode the unfilled portion of the iceberg pegged order is cancelled
        Given the parties deposit on asset's general account the following amount:
            | party            | asset | amount       |
            | buySideProvider  | USD   | 100000000000 |
            | sellSideProvider | USD   | 100000000000 |
            | party1           | USD   | 158550       |
            | aux              | USD   | 1585510000   |

        When the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee | lp type    |
            | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

        And the parties place the following orders:
            | party            | market id | side | volume | price  | resulting trades | type       | tif     |
            | buySideProvider  | ETH/FEB23 | buy  | 1000   | 15600  | 0                | TYPE_LIMIT | TIF_GTC |
            | buySideProvider  | ETH/FEB23 | buy  | 100    | 15800  | 0                | TYPE_LIMIT | TIF_GTC |
            | buySideProvider  | ETH/FEB23 | buy  | 100    | 15800  | 0                | TYPE_LIMIT | TIF_GTC |
            | sellSideProvider | ETH/FEB23 | sell | 100    | 15800  | 0                | TYPE_LIMIT | TIF_GTC |
            | sellSideProvider | ETH/FEB23 | sell | 100    | 15810  | 0                | TYPE_LIMIT | TIF_GTC |
            | sellSideProvider | ETH/FEB23 | sell | 1000   | 200100 | 0                | TYPE_LIMIT | TIF_GTC |

        When the network moves ahead "2" blocks
        And the market data for the market "ETH/FEB23" should be:
            | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
            | 15800      | TRADING_MODE_CONTINUOUS | 5       | 15701     | 15899     | 0            | 1000           | 100           |

        And the parties place the following pegged iceberg orders:
            | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
            | party1 | ETH/FEB23 | 200       | 200                  | buy  | BID              | 200     | 10     | buy_peg_1 |
            | party1 | ETH/FEB23 | 200       | 200                  | buy  | BID              | 200     | 100    | buy_peg_2 |

        And the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux   | ETH/FEB23 | sell | 200    | 15790 | 2                | TYPE_LIMIT | TIF_GTC |

        And the following trades should be executed:
            | buyer  | price | size | seller |
            | party1 | 15790 | 100  | aux    |

        Then the orders should have the following status:
            | party  | reference | status        |
            | party1 | buy_peg_1 | STATUS_ACTIVE |
            | party1 | buy_peg_2 | STATUS_ACTIVE |

        And the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux   | ETH/FEB23 | sell | 2      | 15600 | 0                | TYPE_LIMIT | TIF_GTC |

        And the market data for the market "ETH/FEB23" should be:
            | mark price | trading mode                    |
            | 15800      | TRADING_MODE_MONITORING_AUCTION |

        And the parties submit update margin mode:
            | party  | market    | margin_mode     | margin_factor | error |
            | party1 | ETH/FEB23 | isolated margin | 0.5           |       |

        Then the orders should have the following status:
            | party  | reference | status           |
            | party1 | buy_peg_1 | STATUS_CANCELLED |
            | party1 | buy_peg_2 | STATUS_CANCELLED |

#