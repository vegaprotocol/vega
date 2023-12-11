Feature: Test pegged order amend under isolated margin mode
    Background:
        Given the following network parameters are set:
            | name                                    | value |
            | network.markPriceUpdateMaximumFrequency | 0s    |
            | limits.markets.maxPeggedOrders          | 6     |
        And the liquidity monitoring parameters:
            | name       | triggering ratio | time window | scaling factor |
            | lqm-params | 0.00             | 24h         | 1e-9           |
        And the simple risk model named "simple-risk-model":
            | long | short | max move up | min move down | probability of trading |
            | 0.1  | 0.1   | 100         | -100          | 0.2                    |
        And the markets:
            | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
            | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures |
        And the parties deposit on asset's general account the following amount:
            | party      | asset | amount       |
            | lpprov     | USD   | 100000000000 |
            | aux_buys   | USD   | 100000000000 |
            | aux_sells  | USD   | 100000000000 |
            | test_party | USD   | 100000       |

        When the parties place the following orders:
            | party     | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | aux_buys  | ETH/FEB23 | buy  | 10     | 995   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
            | aux_buys  | ETH/FEB23 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
            | aux_sells | ETH/FEB23 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
            | aux_sells | ETH/FEB23 | sell | 10     | 1005  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
            | lpprov    | ETH/FEB23 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
            | lpprov    | ETH/FEB23 | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
        And the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee | lp type    |
            | lp1 | lpprov | ETH/FEB23 | 900000            | 0.1 | submission |
            | lp1 | lpprov | ETH/FEB23 | 900000            | 0.1 | submission |

        Then the opening auction period ends for market "ETH/FEB23"
        And the mark price should be "1000" for the market "ETH/FEB23"
        And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/FEB23"

    Scenario: When the party cancels a pegged order, which was their only order, the order margin should be 0 (0019-MCAL-049)
        # pegged order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        When the parties place the following pegged orders:
            | party      | market id | side | volume | pegged reference | offset | reference |
            | test_party | ETH/FEB23 | sell | 10     | ASK              | 9      | sell_peg  |
            | test_party | ETH/FEB23 | buy  | 5      | BID              | 9      | buy_peg   |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 9227  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 90773   | 9227         |

        Given the parties cancel the following orders:
            | party      | reference |
            | test_party | buy_peg   |
            | test_party | sell_peg  |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  | 0            |

    Scenario: When the party cancels a pegged order, which was their only order, the order margin should be 0 (0019-MCAL-049)
        # pegged iceberg order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        When the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference    | pegged reference | volume | offset |
            | test_party | ETH/FEB23 | 100       | 10                   | buy  | buy_ice_peg  | BID              | 100    | 9      |
            | test_party | ETH/FEB23 | 100       | 10                   | sell | sell_ice_peg | ASK              | 100    | 9      |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 92274 |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 7726    | 92274        |

        Given the parties cancel the following orders:
            | party      | reference    |
            | test_party | buy_ice_peg  |
            | test_party | sell_ice_peg |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  | 0            |

    Scenario: When the party reduces the pegged order size only, the order margin should be reduced (0019-MCAL-050)
        # pegged order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        When the parties place the following pegged orders:
            | party      | market id | side | volume | pegged reference | offset | reference |
            | test_party | ETH/FEB23 | buy  | 5      | BID              | 9      | buy_peg   |
        # (995 - 9) * 5 * 0.91 = 4486
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 4486  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 95514   | 4486         |

        # amend with size delta
        Given the parties amend the following orders:
            | party      | reference | size delta | size | tif     |
            | test_party | buy_peg   | -2         |      | TIF_GTC |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 2691  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 97309   | 2691         |

        # amend with size
        Given the parties amend the following orders:
            | party      | reference | size delta | size | tif     |
            | test_party | buy_peg   |            | 1    | TIF_GTC |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 897   |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 99103   | 897          |

    Scenario: When the party reduces the pegged order size only, the order margin should be reduced (0019-MCAL-050)
        # pegged iceberg
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        When the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference   | pegged reference | volume | offset |
            | test_party | ETH/FEB23 | 100       | 10                   | buy  | buy_ice_peg | BID              | 100    | 9      |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 89726 |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 10274   | 89726        |

        # amend with size delta
        Given the parties amend the following pegged iceberg orders:
            | party      | reference   | size delta | size | tif     |
            | test_party | buy_ice_peg | -50        |      | TIF_GTC |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 44863 |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 55137   | 44863        |

        # amend with size
        Given the parties amend the following pegged iceberg orders:
            | party      | reference   | size delta | size | tif     |
            | test_party | buy_ice_peg |            | 1    | TIF_GTC |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 897   |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 99103   | 897          |

    Scenario: When the party reduces the pegged buy order offset price, the order margin should be reduced (0019-MCAL-051)
        # pegged order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        When the parties place the following pegged orders:
            | party      | market id | side | volume | pegged reference | offset | reference |
            | test_party | ETH/FEB23 | buy  | 5      | BID              | 1      | buy_peg   |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 4522  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 95478   | 4522         |

        # reduce pegged order price by increasing offset
        Given the parties amend the following orders:
            | party      | reference | pegged offset | tif     |
            | test_party | buy_peg   | 11            | TIF_GTC |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 4477  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 95523   | 4477         |

    Scenario: When the party reduces the pegged buy order offset price, the order margin should be reduced (0019-MCAL-051)
        # pegged iceberg order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        When the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference   | pegged reference | volume | offset |
            | test_party | ETH/FEB23 | 10        | 1                    | buy  | buy_ice_peg | BID              | 10     | 1      |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 9045  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 90955   | 9045         |

        # reduce pegged iceberg order price by increasing offset
        Given the parties amend the following pegged iceberg orders:
            | party      | reference   | offset | tif     |
            | test_party | buy_ice_peg | 11     | TIF_GTC |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 8954  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 91046   | 8954         |

    Scenario: When the party increases the pegged sell order offset price, the order margin should be reduced (0019-MCAL-057)
        # pegged order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        When the parties place the following pegged orders:
            | party      | market id | side | volume | pegged reference | offset | reference |
            | test_party | ETH/FEB23 | sell | 5      | ASK              | 11     | sell_peg  |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 4622  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 95378   | 4622         |

        # reduce pegged order price by increasing offset
        Given the parties amend the following orders:
            | party      | reference | pegged offset | tif     |
            | test_party | sell_peg  | 1             | TIF_GTC |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 4577  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 95423   | 4577         |

    Scenario: When the party increases the pegged sell order offset price, the order margin should be reduced (0019-MCAL-057)
        # pegged iceberg order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        When the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference    | pegged reference | volume | offset |
            | test_party | ETH/FEB23 | 10        | 4                    | sell | sell_ice_peg | ASK              | 10     | 11     |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 9245  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 90755   | 9245         |

        # reduce pegged iceberg order price by increasing offset
        Given the parties amend the following pegged iceberg orders:
            | party      | reference    | offset | tif     |
            | test_party | sell_ice_peg | 1      | TIF_GTC |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 9154  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 90846   | 9154         |

    Scenario: When the party increases the pegged order size and the party's general account does not contain sufficient funds to cover any increases to the order margin account to be equal to side margin then the order should be stopped (0019-MCAL-052)
        # pegged order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.50          |       |
        When the parties place the following pegged orders:
            | party      | market id | side | volume | pegged reference | offset | reference |
            | test_party | ETH/FEB23 | sell | 5      | ASK              | -5     | sell_peg  |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.5           | 2500  |

        Given the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 97500   | 2500         |
        When the parties amend the following orders:
            | party      | reference | size | tif     | error               |
            | test_party | sell_peg  | 201  | TIF_GTC | margin check failed |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.5           | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  | 0            |
        And the orders should have the following status:
            | party      | reference | status         |
            | test_party | sell_peg  | STATUS_STOPPED |

    Scenario: When the party increases the pegged order size and the party's general account does not contain sufficient funds to cover any increases to the order margin account to be equal to side margin then the order should be stopped (0019-MCAL-052)
        # pegged iceberg order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.50          |       |
        When the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference    | pegged reference | volume | offset |
            | test_party | ETH/FEB23 | 5         | 4                    | sell | sell_ice_peg | ASK              | 5      | -5     |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.50          | 2500  |

        Given the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 97500   | 2500         |
        When the parties amend the following pegged iceberg orders:
            | party      | reference    | size | tif     | error               |
            | test_party | sell_ice_peg | 2001 | TIF_GTC | margin check failed |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.50          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  | 0            |
        And the orders should have the following status:
            | party      | reference    | status         |
            | test_party | sell_ice_peg | STATUS_STOPPED |