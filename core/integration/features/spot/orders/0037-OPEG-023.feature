
Feature: 0001 tick size should be using market decimal, A pegged order specifying an offset which is not an integer multiple of the markets tick size should be rejected.

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

        And the following assets are registered:
            | id  | decimal places |
            | ETH | 1              |
            | BTC | 1              |

        And the average block duration is "1"
        And the simple risk model named "simple-risk-model-1":
            | long | short | max move up | min move down | probability of trading |
            | 0.1  | 0.1   | 60          | 50            | 0.2                    |
        And the fees configuration named "fees-config-1":
            | maker fee | infrastructure fee |
            | 0         | 0                  |
        And the price monitoring named "price-monitoring-1":
            | horizon | probability | auction extension |
            | 1       | 0.99        | 5                 |
        And the liquidity sla params named "SLA":
            | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
            | 0.01        | 0.5                          | 1                             | 1.0                    |

    Scenario:
        #0037-OPEG-023:Given a market with non-zero market and asset decimals where the asset decimals are strictly less than the market decimals (yielding a negative price factor). A pegged order specifying an offset which is not an integer multiple of the markets tick size should be rejected.
        #0037-OPEG-024:Given a market with non-zero market and asset decimals where the asset decimals are equal to the market decimals (yielding a negative price factor). A pegged order specifying an offset which is not an integer multiple of the markets tick size should be rejected.
        #0037-OPEG-027:Given a market with non-zero market and asset decimals where the asset decimals are strictly more than the market decimals (yielding a negative price factor). A pegged order specifying an offset which is not an integer multiple of the markets tick size should be rejected.
        And the spot markets:
            | id        | name    | quote asset | base asset | liquidity monitoring | risk model          | auction duration | fees          | price monitoring   | sla params    | decimal places | tick size |
            | ETH/DEC21 | BTC/ETH | BTC         | ETH        | lqm-params           | simple-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | default-basic | 2              | 10        |
            | ETH/DEC22 | BTC/ETH | BTC         | ETH        | lqm-params           | simple-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | default-basic | 2              | 5         |
            | ETH/DEC23 | BTC/ETH | BTC         | ETH        | lqm-params           | simple-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | default-basic | 1              | 5         |
            | ETH/DEC24 | BTC/ETH | BTC         | ETH        | lqm-params           | simple-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | default-basic | 0              | 10        |
        And the parties deposit on asset's general account the following amount:
            | party  | asset | amount        |
            | party1 | ETH   | 1000000000000 |
            | party1 | BTC   | 1000          |
            | party3 | ETH   | 1000000       |
            | party4 | ETH   | 1000000       |
        And the average block duration is "1"

        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                              |
            | party3 | ETH/DEC21 | buy  | 100    | 101   | 0                | TYPE_LIMIT | TIF_GTC | p3b1-1    | OrderError: price not in tick size |
            | party3 | ETH/DEC21 | buy  | 10     | 111   | 0                | TYPE_LIMIT | TIF_GTC | p3b2-1    | OrderError: price not in tick size |
            | party4 | ETH/DEC21 | sell | 10     | 111   | 0                | TYPE_LIMIT | TIF_GTC | p4s2-1    | OrderError: price not in tick size |
            | party4 | ETH/DEC21 | sell | 1000   | 191   | 0                | TYPE_LIMIT | TIF_GTC | p4s1-1    | OrderError: price not in tick size |
            | party3 | ETH/DEC22 | buy  | 100    | 101   | 0                | TYPE_LIMIT | TIF_GTC | p3b1-2    | OrderError: price not in tick size |
            | party3 | ETH/DEC22 | buy  | 10     | 111   | 0                | TYPE_LIMIT | TIF_GTC | p3b2-2    | OrderError: price not in tick size |
            | party4 | ETH/DEC22 | sell | 10     | 111   | 0                | TYPE_LIMIT | TIF_GTC | p4s2-2    | OrderError: price not in tick size |
            | party4 | ETH/DEC22 | sell | 1000   | 191   | 0                | TYPE_LIMIT | TIF_GTC | p4s1-2    | OrderError: price not in tick size |
            | party3 | ETH/DEC23 | buy  | 100    | 101   | 0                | TYPE_LIMIT | TIF_GTC | p3b1-3    | OrderError: price not in tick size |
            | party3 | ETH/DEC23 | buy  | 10     | 111   | 0                | TYPE_LIMIT | TIF_GTC | p3b2-3    | OrderError: price not in tick size |
            | party4 | ETH/DEC23 | sell | 10     | 111   | 0                | TYPE_LIMIT | TIF_GTC | p4s2-3    | OrderError: price not in tick size |
            | party4 | ETH/DEC23 | sell | 1000   | 191   | 0                | TYPE_LIMIT | TIF_GTC | p4s1-3    | OrderError: price not in tick size |
            | party3 | ETH/DEC24 | buy  | 100    | 101   | 0                | TYPE_LIMIT | TIF_GTC | p3b1-4    | OrderError: price not in tick size |
            | party3 | ETH/DEC24 | buy  | 10     | 111   | 0                | TYPE_LIMIT | TIF_GTC | p3b2-4    | OrderError: price not in tick size |
            | party4 | ETH/DEC24 | sell | 10     | 111   | 0                | TYPE_LIMIT | TIF_GTC | p4s2-4    | OrderError: price not in tick size |
            | party4 | ETH/DEC24 | sell | 1000   | 191   | 0                | TYPE_LIMIT | TIF_GTC | p4s1-4    | OrderError: price not in tick size |

        Then the network moves ahead "2" blocks
        And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC21"
        And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC22"
        And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC23"
        And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC24"

        And the parties place the following pegged iceberg orders:
            | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference | error                              |
            | party1 | ETH/DEC21 | 10        | 5                    | buy  | MID              | 10     | 10     | peg-buy-1 |                                    |
            | party1 | ETH/DEC21 | 10        | 5                    | sell | MID              | 20     | 20     | peg-buy-2 |                                    |
            | party1 | ETH/DEC21 | 10        | 5                    | buy  | MID              | 20     | 100    | peg-buy-3 |                                    |
            | party1 | ETH/DEC21 | 10        | 5                    | sell | MID              | 20     | 2      | peg-buy-4 | OrderError: price not in tick size |
            | party1 | ETH/DEC21 | 10        | 5                    | buy  | MID              | 20     | 5      | peg-buy-5 | OrderError: price not in tick size |
            | party1 | ETH/DEC22 | 10        | 5                    | buy  | MID              | 20     | 15     | peg-buy-6 |                                    |
            | party1 | ETH/DEC23 | 10        | 5                    | sell | MID              | 20     | 6      | peg-buy-7 | OrderError: price not in tick size |
            | party1 | ETH/DEC24 | 10        | 5                    | buy  | MID              | 20     | 17     | peg-buy-8 | OrderError: price not in tick size |
