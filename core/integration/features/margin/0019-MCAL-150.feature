Feature:  pegged order in isoalted margin is not supported
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
            | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring-1 | default-eth-for-future | 0.000125               | 0                         | 2                       | default-futures |

        And the following network parameters are set:
            | name                           | value |
            | market.auction.minimumDuration | 1     |
        Given the average block duration is "1"

    Scenario: When the party has pegged orders and switches from cross margin mode to isolated margin mode, all the pegged orders will be stopped. (0019-MCAL-050,0019-MCAL-090)
        Given the parties deposit on asset's general account the following amount:
            | party            | asset | amount       |
            | buySideProvider  | USD   | 100000000000 |
            | sellSideProvider | USD   | 100000000000 |
            | party1           | USD   | 158550       |

        When the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee | lp type    |
            | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

        And the parties place the following orders:
            | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
            | buySideProvider  | ETH/FEB23 | buy  | 1000   | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
            | buySideProvider  | ETH/FEB23 | buy  | 300    | 15600  | 0                | TYPE_LIMIT | TIF_GTC |           |
            | party1           | ETH/FEB23 | buy  | 100    | 15700  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy-1  |
            | buySideProvider  | ETH/FEB23 | buy  | 300    | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
            | party1           | ETH/FEB23 | sell | 300    | 15800  | 0                | TYPE_LIMIT | TIF_GTC | p1-sell-1 |
            | sellSideProvider | ETH/FEB23 | sell | 600    | 15802  | 0                | TYPE_LIMIT | TIF_GTC | sP-sell   |
            | sellSideProvider | ETH/FEB23 | sell | 300    | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
            | sellSideProvider | ETH/FEB23 | sell | 1000   | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

        When the network moves ahead "2" blocks
        And the market data for the market "ETH/FEB23" should be:
            | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
            | 15800      | TRADING_MODE_CONTINUOUS | 5       | 15701     | 15899     | 0            | 1000           | 300           |

        #now we try to place short pegged order which does offset the current short position, order margin should be 0
        When the parties place the following pegged orders:
            | party  | market id | side | volume | pegged reference | offset | reference |
            | party1 | ETH/FEB23 | buy  | 100    | BID              | 10     | buy_peg_1 |
            | party1 | ETH/FEB23 | buy  | 200    | BID              | 20     | buy_peg_2 |

        Then the parties should have the following margin levels:
            | party  | market id | maintenance | margin mode  | margin factor | order |
            | party1 | ETH/FEB23 | 9486        | cross margin | 0             | 0     |
        And the parties should have the following account balances:
            | party  | asset | market id | margin | general | bond |
            | party1 | USD   | ETH/FEB23 | 11383  | 146167  | 1000 |

        #switch to isolated margin
        And the parties submit update margin mode:
            | party  | market    | margin_mode     | margin_factor | error                                                                                     |
            | party1 | ETH/FEB23 | isolated margin | 0.1           | margin factor (0.1) must be greater than max(riskFactorLong (0.1), riskFactorShort (0.2)) + linearSlippageFactor (0.000125) |

        And the parties submit update margin mode:
            | party  | market    | margin_mode     | margin_factor | error                                                                                     |
            | party1 | ETH/FEB23 | isolated margin | 0.15           | margin factor (0.15) must be greater than max(riskFactorLong (0.1), riskFactorShort (0.2)) + linearSlippageFactor (0.000125) |

        And the parties submit update margin mode:
            | party  | market    | margin_mode     | margin_factor | error                                                        |
            | party1 | ETH/FEB23 | isolated margin | 0.21          | required position margin must be greater than initial margin |

        And the parties submit update margin mode:
            | party  | market    | margin_mode     | margin_factor | error |
            | party1 | ETH/FEB23 | isolated margin | 0.3           |       |

        Then the parties should have the following margin levels:
            | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | party1 | ETH/FEB23 | 9486        | 0      | 11383   | 0       | isolated margin | 0.3           | 0     |

        #0019-MCAL-149:A party with a parked pegged order switches from cross margin mode to isolated margin mode, the parked pegged order is cancelled
        Then the orders should have the following status:
            | party  | reference | status           |
            | party1 | buy_peg_1 | STATUS_CANCELLED |
            | party1 | buy_peg_2 | STATUS_CANCELLED |
            | party1 | p1-sell-1 | STATUS_FILLED    |
            | party1 | p1-buy-1  | STATUS_ACTIVE    |

        #0019-MCAL-049:When the party submit a pegged order, it should be rejected
        When the parties place the following pegged orders:
            | party  | market id | side | volume | pegged reference | offset | reference | error                                                         |
            | party1 | ETH/FEB23 | buy  | 100    | BID              | 10     | buy_peg_3 | OrderError: pegged orders not allowed in isolated margin mode |
            | party1 | ETH/FEB23 | buy  | 200    | BID              | 20     | buy_peg_4 | OrderError: pegged orders not allowed in isolated margin mode |

        Then the orders should have the following status:
            | party  | reference | status          |
            | party1 | buy_peg_3 | STATUS_REJECTED |
            | party1 | buy_peg_4 | STATUS_REJECTED |

        And the parties submit update margin mode:
            | party  | market    | margin_mode  | margin_factor | error |
            | party1 | ETH/FEB23 | cross margin | 0             |       |
        Then the parties should have the following margin levels:
            | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | party1 | ETH/FEB23 | 9486        | 10434  | 11383   | 13280   | cross margin | 0             | 0     |

        #0019-MCAL-051:When the party has iceberg pegged orders and switches from cross margin mode to isolated margin mode, all the iceberg pegged orders will be stopped.
        When the parties place the following pegged orders:
            | party  | market id | side | volume | pegged reference | offset | reference | error |
            | party1 | ETH/FEB23 | buy  | 100    | BID              | 10     | buy_peg_5 |       |
            | party1 | ETH/FEB23 | buy  | 200    | BID              | 20     | buy_peg_6 |       |

        Then the orders should have the following status:
            | party  | reference | status        |
            | party1 | buy_peg_5 | STATUS_ACTIVE |
            | party1 | buy_peg_6 | STATUS_ACTIVE |

        And the parties submit update margin mode:
            | party  | market    | margin_mode     | margin_factor | error |
            | party1 | ETH/FEB23 | isolated margin | 0.3           |       |

        Then the parties should have the following margin levels:
            | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | party1 | ETH/FEB23 | 9486        | 0      | 11383   | 0       | isolated margin | 0.3           | 0     |

        Then the orders should have the following status:
            | party  | reference | status           |
            | party1 | buy_peg_5 | STATUS_CANCELLED |
            | party1 | buy_peg_6 | STATUS_CANCELLED |



