Feature: Test for parked/unparked pegged orders
    Background: Background name:
        Given the following network parameters are set:
            | name                                    | value |
            | network.markPriceUpdateMaximumFrequency | 0s    |
            | limits.markets.maxPeggedOrders          | 6     |
        And the average block duration is "1"
        And the liquidity monitoring parameters:
            | name       | triggering ratio | time window | scaling factor |
            | lqm-params | 0.00             | 24h         | 1e-9           |
        And the simple risk model named "simple-risk-model":
            | long | short | max move up | min move down | probability of trading |
            | 0.1  | 0.1   | 100         | -100          | 0.2                    |
        And the price monitoring named "price-monitoring":
            | horizon | probability | auction extension |
            | 3600    | 0.95        | 3                 |
        And the markets:
            | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
            | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures |
        And the parties deposit on asset's general account the following amount:
            | party      | asset | amount       |
            | lpprov     | USD   | 100000000000 |
            | aux_buys   | USD   | 100000000000 |
            | aux_sells  | USD   | 100000000000 |
            | test_party | USD   | 100000       |

        When the parties place the following orders:
            | party     | market id | side | volume | price | resulting trades | type       | tif     | reference    |
            | aux_buys  | ETH/FEB23 | buy  | 10     | 995   | 0                | TYPE_LIMIT | TIF_GTC | buy-1        |
            | aux_buys  | ETH/FEB23 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-trade-1  |
            | aux_sells | ETH/FEB23 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-trade-1 |
            | aux_sells | ETH/FEB23 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-1       |
            | aux_sells | ETH/FEB23 | sell | 10     | 1505  | 0                | TYPE_LIMIT | TIF_GTC | sell-2       |
            | lpprov    | ETH/FEB23 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | lp-buy-1     |
            | lpprov    | ETH/FEB23 | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC | lp-sell-1    |
        And the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee | lp type    |
            | lp1 | lpprov | ETH/FEB23 | 900000            | 0.1 | submission |
            | lp1 | lpprov | ETH/FEB23 | 900000            | 0.1 | submission |

        Then the opening auction period ends for market "ETH/FEB23"
        And the mark price should be "1000" for the market "ETH/FEB23"
        And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/FEB23"
        And the network moves ahead "1" blocks

    Scenario: In cross margin mode, party park pegged order, switch to isolated margin and the pegged order is cancelled 0019-MCAL-143
        # enter pegged order here
        Given the parties submit update margin mode:
            | party      | market    | margin_mode  | margin_factor | error |
            | test_party | ETH/FEB23 | cross margin |               |       |
        When the parties place the following pegged orders:
            | party      | market id | side | volume | pegged reference | offset | reference |
            | test_party | ETH/FEB23 | buy  | 5      | MID              | 1      | buy_peg   |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 500         | 550    | 600     | 700     | cross margin |               | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 600    | 99400   |

        Given the orders should have the following status:
            | party      | reference | status        |
            | test_party | buy_peg   | STATUS_ACTIVE |
        When the parties place the following orders:
            | party    | market id | side | volume | price | resulting trades | type       | tif     | reference         |
            | aux_buys | ETH/FEB23 | buy  | 1000   | 10000 | 0                | TYPE_LIMIT | TIF_GTC | auction-trigger-1 |
        Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/FEB23"
        And the orders should have the following status:
            | party      | reference | status        |
            | test_party | buy_peg   | STATUS_PARKED |
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | cross margin |               | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  |

        Given the parties cancel the following orders:
            | party    | reference         |
            | aux_buys | auction-trigger-1 |
        When the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.50          |       |
        Then the orders should have the following status:
            | party      | reference | status           |
            | test_party | buy_peg   | STATUS_CANCELLED |

        Given the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/FEB23"
        When the network moves ahead "4" blocks
        Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/FEB23"
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.50          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  | 0            |

    Scenario: In cross margin mode, party park pegged order, switch to isolated margin and the pegged order is cancelled 0019-MCAL-144
        # enter pegged iceberg order here
        Given the parties submit update margin mode:
            | party      | market    | margin_mode  | margin_factor | error |
            | test_party | ETH/FEB23 | cross margin |               |       |
        When the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference   | pegged reference | volume | offset |
            | test_party | ETH/FEB23 | 5         | 2                    | buy  | buy_ice_peg | MID              | 5      | 1      |
        # (1000 - 1) * 5 * 0.5 = 2492
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 500         | 550    | 600     | 700     | cross margin |               | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 600    | 99400   |

        Given the orders should have the following status:
            | party      | reference   | status        |
            | test_party | buy_ice_peg | STATUS_ACTIVE |
        When the parties place the following orders:
            | party    | market id | side | volume | price | resulting trades | type       | tif     | reference         |
            | aux_buys | ETH/FEB23 | buy  | 1000   | 10000 | 0                | TYPE_LIMIT | TIF_GTC | auction-trigger-1 |
        Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/FEB23"
        And the orders should have the following status:
            | party      | reference   | status        |
            | test_party | buy_ice_peg | STATUS_PARKED |
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | cross margin |               | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  |

        Given the parties cancel the following orders:
            | party    | reference         |
            | aux_buys | auction-trigger-1 |
        When the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.50          |       |
        Then the orders should have the following status:
            | party      | reference   | status           |
            | test_party | buy_ice_peg | STATUS_CANCELLED |

        Given the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/FEB23"
        When the network moves ahead "4" blocks
        Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/FEB23"
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.50          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 0      | 100000  | 0            |

    Scenario: In cross margin mode, partially fill pegged order, park then switch to isolated margin. Pegged order is cancelled 0019-MCAL-147
        # enter pegged order here
        Given the parties submit update margin mode:
            | party      | market    | margin_mode  | margin_factor | error |
            | test_party | ETH/FEB23 | cross margin |               |       |
        And the parties place the following pegged orders:
            | party      | market id | side | volume | pegged reference | offset | reference |
            | test_party | ETH/FEB23 | buy  | 5      | MID              | 1      | buy_peg   |
        And the parties place the following orders:
            | party     | market id | side | volume | price | resulting trades | type       | tif     | reference        |
            | aux_sells | ETH/FEB23 | sell | 2      | 995   | 1                | TYPE_LIMIT | TIF_GTC | pfill_test_order |
        And the following trades should be executed:
            | buyer      | price | size | seller    |
            | test_party | 997   | 2    | aux_sells |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 500         | 550    | 600     | 700     | cross margin |               | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 600    | 99400   |
        Given the orders should have the following status:
            | party      | reference | status        |
            | test_party | buy_peg   | STATUS_ACTIVE |
        When the parties place the following orders:
            | party    | market id | side | volume | price | resulting trades | type       | tif     | reference         |
            | aux_buys | ETH/FEB23 | buy  | 1000   | 10000 | 0                | TYPE_LIMIT | TIF_GTC | auction-trigger-1 |
        Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/FEB23"
        And the orders should have the following status:
            | party      | reference | status        |
            | test_party | buy_peg   | STATUS_PARKED |
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 500         | 550    | 600     | 700     | cross margin |               | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 600    | 99400   |

        Given the parties cancel the following orders:
            | party    | reference         |
            | aux_buys | auction-trigger-1 |
        When the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.50          |       |
        Then the orders should have the following status:
            | party      | reference | status           |
            | test_party | buy_peg   | STATUS_CANCELLED |

        Given the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/FEB23"
        When the network moves ahead "4" blocks
        Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/FEB23"
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 204         | 0      | 244     | 0       | isolated margin | 0.50          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 997    | 99003   | 0            |

    Scenario: In cross margin mode, partially fill pegged order, park then switch to isolated margin. Pegged order is cancelled 0019-MCAL-148
        # enter pegged iceberg order here
        Given the parties submit update margin mode:
            | party      | market    | margin_mode  | margin_factor | error |
            | test_party | ETH/FEB23 | cross margin |               |       |
        When the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference   | pegged reference | volume | offset |
            | test_party | ETH/FEB23 | 5         | 2                    | buy  | buy_ice_peg | MID              | 5      | 1      |
        And the parties place the following orders:
            | party     | market id | side | volume | price | resulting trades | type       | tif     | reference        |
            | aux_sells | ETH/FEB23 | sell | 2      | 995   | 1                | TYPE_LIMIT | TIF_GTC | pfill_test_order |
        And the following trades should be executed:
            | buyer      | price | size | seller    |
            | test_party | 997   | 2    | aux_sells |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 500         | 550    | 600     | 700     | cross margin |               | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 600    | 99400   |
        Given the orders should have the following status:
            | party      | reference   | status        |
            | test_party | buy_ice_peg | STATUS_ACTIVE |
        When the parties place the following orders:
            | party    | market id | side | volume | price | resulting trades | type       | tif     | reference         |
            | aux_buys | ETH/FEB23 | buy  | 1000   | 10000 | 0                | TYPE_LIMIT | TIF_GTC | auction-trigger-1 |
        Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/FEB23"
        And the orders should have the following status:
            | party      | reference   | status        |
            | test_party | buy_ice_peg | STATUS_PARKED |
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
            | test_party | ETH/FEB23 | 500         | 550    | 600     | 700     | cross margin |               | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general |
            | test_party | USD   | ETH/FEB23 | 600    | 99400   |

        Given the parties cancel the following orders:
            | party    | reference         |
            | aux_buys | auction-trigger-1 |
        When the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.50          |       |
        Then the orders should have the following status:
            | party      | reference   | status           |
            | test_party | buy_ice_peg | STATUS_CANCELLED |

        Given the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/FEB23"
        When the network moves ahead "4" blocks
        Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/FEB23"
        And the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 204         | 0      | 244     | 0       | isolated margin | 0.50          | 0     |
        And the parties should have the following account balances:
            | party      | asset | market id | margin | general | order margin |
            | test_party | USD   | ETH/FEB23 | 997    | 99003   | 0            |

    Scenario: In auction a party in isolated margin mode enter pegged order that is rejected 0019-MCAL-049
        Given the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/FEB23"
        When the parties place the following orders:
            | party    | market id | side | volume | price | resulting trades | type       | tif     | reference         |
            | aux_buys | ETH/FEB23 | buy  | 1000   | 10000 | 0                | TYPE_LIMIT | TIF_GTC | auction-trigger-1 |
        Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/FEB23"

        Given the parties submit update margin mode:
            | party      | market    | margin_mode     | margin_factor | error |
            | test_party | ETH/FEB23 | isolated margin | 0.5           |       |
        When the parties place the following pegged orders:
            | party      | market id | side | volume | pegged reference | offset | reference | error              |
            | test_party | ETH/FEB23 | buy  | 5      | MID              | 1      | buy_peg   | invalid OrderError |
        And the parties place the following pegged iceberg orders:
            | party      | market id | peak size | minimum visible size | side | reference   | pegged reference | volume | offset | error              |
            | test_party | ETH/FEB23 | 5         | 2                    | buy  | buy_ice_peg | MID              | 5      | 1      | invalid OrderError |
        Then the parties should have the following margin levels:
            | party      | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
            | test_party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin |               | 0     |