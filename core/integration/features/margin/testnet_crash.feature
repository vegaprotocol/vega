Feature: Test order amendment such that the full order is matched but the party doesn't have sufficient cover and their orders should be cancelled and funds released. This is testing the fix for https://github.com/vegaprotocol/vega/issues/10493

    Background:
        Given the following network parameters are set:
            | name                                    | value |
            | network.markPriceUpdateMaximumFrequency | 0s    |
        And the liquidity monitoring parameters:
            | name       | triggering ratio | time window | scaling factor |
            | lqm-params | 0.00             | 24h         | 1e-9           |
        And the fees configuration named "fees-config-1":
            | maker fee | infrastructure fee |
            | 0.0004    | 0.001              |
        And the simple risk model named "simple-risk-model":
            | long | short | max move up | min move down | probability of trading |
            | 0.1  | 0.1   | 100         | -100          | 0.2                    |
        And the markets:
            | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
            | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | fees-config-1 | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures |

    Scenario: 001 The party tried to amend an order which is fully matched after the price change but they don't have sufficient cover.
        Given the parties deposit on asset's general account the following amount:
            | party            | asset | amount       |
            | buySideProvider  | USD   | 100000000000 |
            | sellSideProvider | USD   | 100000000000 |
            | party1           | USD   | 320000       |

        And the parties place the following orders:
            | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900 | 0                | TYPE_LIMIT | TIF_GTC |           |
            | buySideProvider  | ETH/FEB23 | buy  | 6      | 15800 | 0                | TYPE_LIMIT | TIF_GTC |           |
            | buySideProvider  | ETH/FEB23 | buy  | 10     | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
            | party1           | ETH/FEB23 | buy  | 100    | 15850 | 0                | TYPE_LIMIT | TIF_GTC | buy-1     |
            | sellSideProvider | ETH/FEB23 | sell | 10     | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
            | sellSideProvider | ETH/FEB23 | sell | 100    | 20100 | 0                | TYPE_LIMIT | TIF_GTC |           |

        When the network moves ahead "2" blocks
        Then the mark price should be "15900" for the market "ETH/FEB23"

        And the parties should have the following margin levels:
            | party  | market id | maintenance | initial |
            | party1 | ETH/FEB23 | 159000      | 190800  |

        Then the parties should have the following account balances:
            | party  | asset | market id | margin | general |
            | party1 | USD   | ETH/FEB23 | 190200 | 129800  |

        And the parties submit update margin mode:
            | party  | market    | margin_mode     | margin_factor |
            | party1 | ETH/FEB23 | isolated margin | 0.2           |

        And the parties should have the following margin levels:
            | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order  |
            | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.2           | 317000 |

        When the parties amend the following orders:
            | party  | reference | price | size delta | tif     | error               |
            | party1 | buy-1     | 20100 | 0          | TIF_GTC | margin check failed |

        And the orders should have the following status:
            | party  | reference | status         |
            | party1 | buy-1     | STATUS_STOPPED |

        Then the parties should have the following account balances:
            | party  | asset | market id | margin | general |
            | party1 | USD   | ETH/FEB23 | 0      | 320000  |


    @Fail
    Scenario: 002 The party tried to amend an order which is partially matched after the price change but they don't have sufficient cover.
        Given the parties deposit on asset's general account the following amount:
            | party            | asset | amount       |
            | buySideProvider  | USD   | 100000000000 |
            | sellSideProvider | USD   | 100000000000 |
            | party1           | USD   | 320000       |

        And the parties place the following orders:
            | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900 | 0                | TYPE_LIMIT | TIF_GTC |           |
            | buySideProvider  | ETH/FEB23 | buy  | 6      | 15800 | 0                | TYPE_LIMIT | TIF_GTC |           |
            | buySideProvider  | ETH/FEB23 | buy  | 10     | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
            | party1           | ETH/FEB23 | buy  | 100    | 15850 | 0                | TYPE_LIMIT | TIF_GTC | buy-1     |
            | sellSideProvider | ETH/FEB23 | sell | 10     | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
            | sellSideProvider | ETH/FEB23 | sell | 10     | 20100 | 0                | TYPE_LIMIT | TIF_GTC |           |

        When the network moves ahead "2" blocks
        Then the mark price should be "15900" for the market "ETH/FEB23"

        And the parties should have the following margin levels:
            | party  | market id | maintenance | initial |
            | party1 | ETH/FEB23 | 159000      | 190800  |

        Then the parties should have the following account balances:
            | party  | asset | market id | margin | general |
            | party1 | USD   | ETH/FEB23 | 190200 | 129800  |

        And the parties submit update margin mode:
            | party  | market    | margin_mode     | margin_factor |
            | party1 | ETH/FEB23 | isolated margin | 0.2           |

        And the parties should have the following margin levels:
            | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order  |
            | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.2           | 317000 |

        When the parties amend the following orders:
            | party  | reference | price | size delta | tif     | error               |
            | party1 | buy-1     | 20100 | 0          | TIF_GTC | margin check failed |

        And the orders should have the following status:
            | party  | reference | status         |
            | party1 | buy-1     | STATUS_STOPPED |

        Then the parties should have the following account balances:
            | party  | asset | market id | margin | general | order margin |
            | party1 | USD   | ETH/FEB23 | 40200  | 279518  | 0            |

