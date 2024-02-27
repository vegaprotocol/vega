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
            | party       | asset | amount       |
            | lpprov      | USD   | 100000000000 |
            | aux_buys    | USD   | 100000000000 |
            | aux_sells   | USD   | 100000000000 |
            | test_party  | USD   | 100000       |
            | test_party2 | USD   | 100000       |

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

    Scenario: Party cannot enter a pegged order when in isolated margin mode (0019-MCAL-049)
        # pegged order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        When the parties place the following pegged orders:
            | party      | market id | side | volume | pegged reference | offset | reference | error              |
            | test_party | ETH/FEB23 | sell | 10     | ASK              | 9      | sell_peg  | invalid OrderError |
            | test_party | ETH/FEB23 | buy  | 5      | BID              | 9      | buy_peg   | invalid OrderError |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  | 0            |

    Scenario: Party cannot enter a pegged order when in isolated margin mode (0019-MCAL-052)
        # pegged iceberg order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        When the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference    | pegged reference | volume | offset | error              |
            | test_party | ETH/FEB23 | 100       | 10                   | buy  | buy_ice_peg  | BID              | 100    | 9      | invalid OrderError |
            | test_party | ETH/FEB23 | 100       | 10                   | sell | sell_ice_peg | ASK              | 100    | 9      | invalid OrderError |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  | 0            |

    Scenario: When the party has a pegged order in cross margin mode switches to isolated margin mode the pegged order is cancelled (0019-MCAL-050)
        # pegged order
        Given the parties submit update margin mode:
            | party      | market    | margin_mode  | margin_factor | error |
            | test_party | ETH/FEB23 | cross margin |               |       |
        When the parties place the following pegged orders:
            | party      | market id | side | volume | pegged reference | offset | reference |
            | test_party | ETH/FEB23 | buy  | 5      | BID              | 9      | buy_peg   |
            | test_party | ETH/FEB23 | sell | 10     | ASK              | 9      | sell_peg  |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 1000        | 1100   | 1200    | 1400    | cross margin |               |       |

        Given the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 1200   | 98800   |
        When the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        Then the orders should have the following status:
            | party      | reference | status           |
            | test_party | buy_peg   | STATUS_CANCELLED |
            | test_party | sell_peg  | STATUS_CANCELLED |
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  | 0            |

    Scenario: When the party has a pegged order in cross margin mode switches to isolated margin mode the pegged order is cancelled (0019-MCAL-051)
        # pegged iceberg
        Given the parties submit update margin mode:
            | party      | market    | margin_mode  | margin_factor | error |
            | test_party | ETH/FEB23 | cross margin |               |       |
        When the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference    | pegged reference | volume | offset |
            | test_party | ETH/FEB23 | 100       | 10                   | buy  | buy_ice_peg  | BID              | 100    | 9      |
            | test_party | ETH/FEB23 | 100       | 10                   | sell | sell_ice_peg | ASK              | 100    | 9      |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 10000       | 11000  | 12000   | 14000   | cross margin |               | 0     |

        Given the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 12000  | 88000   |
        When the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.5           |       |
        Then the orders should have the following status:
            | party      | reference    | status           |
            | test_party | buy_ice_peg  | STATUS_CANCELLED |
            | test_party | sell_ice_peg | STATUS_CANCELLED |
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.5           | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  | 0            |

    Scenario: When in cross margin mode and multiple parties enter multiple types of orders, when one party switches to isolated margin mode only their pegged orders are cancelled. (0019-MCAL-057)
        # pegged order
        Given the parties submit update margin mode:
            | party       | market    | margin_mode  | margin_factor | error |
            | test_party  | ETH/FEB23 | cross margin |               |       |
            | test_party2 | ETH/FEB23 | cross margin |               |       |
        When the parties place the following pegged orders:
            | party       | market id | side | volume | pegged reference | offset | reference |
            | test_party  | ETH/FEB23 | buy  | 5      | BID              | 1      | buy_peg   |
            | test_party  | ETH/FEB23 | sell | 5      | ASK              | 1      | sell_peg  |
            | test_party2 | ETH/FEB23 | buy  | 5      | BID              | 1      | buy_peg2  |
            | test_party2 | ETH/FEB23 | sell | 5      | MID              | 1      | sell_peg2 |
        And the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference    | pegged reference | volume | offset |
            | test_party | ETH/FEB23 | 10        | 5                    | buy  | buy_ice_peg  | BID              | 10     | 9      |
            | test_party | ETH/FEB23 | 10        | 5                    | sell | sell_ice_peg | ASK              | 10     | 9      |
        And the parties place the following orders:
            | party       | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | test_party  | ETH/FEB23 | buy  | 10     | 995   | 0                | TYPE_LIMIT | TIF_GTC | buy_lim   |
            | test_party2 | ETH/FEB23 | buy  | 10     | 996   | 0                | TYPE_LIMIT | TIF_GTC | buy_lim2  |
            | test_party  | ETH/FEB23 | sell | 10     | 1005  | 0                | TYPE_LIMIT | TIF_GTC | sell_lim  |
            | test_party2 | ETH/FEB23 | sell | 10     | 1006  | 0                | TYPE_LIMIT | TIF_GTC | sell_lim2 |
        Then the parties should have the following margin levels:
            | party       | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party  | ETH/FEB23 | 2500        | 2750   | 3000    | 3500    | cross margin |               | 0     |
            | test_party2 | ETH/FEB23 | 1500        | 1650   | 1800    | 2100    | cross margin |               | 0     |

        Given the parties should have the following account balances:
            | party       | asset | market id | margin | general |
            | test_party  | USD   | ETH/FEB23 | 3000   | 97000   |
            | test_party2 | USD   | ETH/FEB23 | 1800   | 98200   |
        When the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        Then the orders should have the following status:
            | party       | reference | status           |
            | test_party  | buy_peg   | STATUS_CANCELLED |
            | test_party  | sell_peg  | STATUS_CANCELLED |
            | test_party  | buy_lim   | STATUS_ACTIVE    |
            | test_party  | sell_lim  | STATUS_ACTIVE    |
            | test_party2 | buy_peg2  | STATUS_ACTIVE    |
            | test_party2 | sell_peg2 | STATUS_ACTIVE    |
            | test_party2 | buy_lim2  | STATUS_ACTIVE    |
            | test_party2 | sell_lim2 | STATUS_ACTIVE    |
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.91          | 9145  |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 90855   | 9145         |

    Scenario: A party in cross maring mode has a partially filled pegged order, when the party switches to isolated margin mode the pegged order is cancelled (0019-MCAL-075)
        Given the parties submit update margin mode:
            | party      | market    | margin_mode  | margin_factor | error |
            | test_party | ETH/FEB23 | cross margin |               |       |
        When the parties place the following pegged orders:
            | party      | market id | side | volume | pegged reference | offset | reference |
            | test_party | ETH/FEB23 | sell | 20     | ASK              | -5     | sell_peg  |
        And the parties place the following orders:
            | party    | market id | side | volume | price | resulting trades | type       | tif     | reference   |
            | aux_buys | ETH/FEB23 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | auc-trade-1 |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 2000        | 2200   | 2400    | 2800    | cross margin |               | 0     |

        Given the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 2400   | 97600   |
        When the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        Then the orders should have the following status:
            | party      | reference | status           |
            | test_party | sell_peg  | STATUS_CANCELLED |
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 1050        | 0      | 1260    | 0       | isolated margin | 0.91          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 9100   | 90900   | 0            |

    Scenario: A party in cross maring mode has a partially filled pegged order, when the party switches to isolated margin mode the pegged order is cancelled (0019-MCAL-078)
        # iceberg pegged
        Given the parties submit update margin mode:
            | party      | market    | margin_mode  | margin_factor | error |
            | test_party | ETH/FEB23 | cross margin |               |       |
        When the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference    | pegged reference | volume | offset |
            | test_party | ETH/FEB23 | 20        | 10                   | sell | sell_ice_peg | ASK              | 20     | -5     |
        And the parties place the following orders:
            | party    | market id | side | volume | price | resulting trades | type       | tif     | reference   |
            | aux_buys | ETH/FEB23 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | auc-trade-1 |

        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 2000        | 2200   | 2400    | 2800    | cross margin |               | 0     |

        Given the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 2400   | 97600   |
        When the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.91          |       |
        Then the orders should have the following status:
            | party      | reference    | status           |
            | test_party | sell_ice_peg | STATUS_CANCELLED |
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 1050        | 0      | 1260    | 0       | isolated margin | 0.91          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 9100   | 90900   | 0            |