Feature: Test how offsets are applied with decimals

    Scenario:

        Given the following network parameters are set:
            | name                                                | value |
            | market.value.windowLength                           | 1h    |
            | market.stake.target.timeWindow                      | 24h   |
            | market.stake.target.scalingFactor                   | 1     |
            | market.liquidity.targetstake.triggering.ratio       | 0     |
            | network.markPriceUpdateMaximumFrequency             | 0s    |
            | limits.markets.maxPeggedOrders                      | 8     |
        And the following assets are registered:
            | id  | decimal places |
            | ETH | 5              |
            | USD | 2              |
        And the average block duration is "2"

        And the log normal risk model named "log-normal-risk-model-1":
            | risk aversion | tau | mu | r | sigma |
            | 0.000001      | 0.1 | 0  | 0 | 1.0   |
        And the fees configuration named "fees-config-1":
            | maker fee | infrastructure fee |
            | 0.0004    | 0.001              |
        And the price monitoring named "price-monitoring-1":
            | horizon | probability | auction extension |
            | 100000  | 0.99        | 3                 |

        And the liquidity sla params named "SLA":
            | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
            | 1.0         | 0.5                          | 1                             | 1.0                    |
        And the following network parameters are set:
            | name                                               | value |
            | market.liquidity.providersFeeCalculationTimeStep | 660s  |

        And the markets:
            | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees         | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor | sla params |
            | USD/DEC19 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none | price-monitoring-1 | default-usd-for-future | 3              | 3                       | 1e6                    | 1e6                       | SLA        |

        Given the parties deposit on asset's general account the following amount:
            | party  | asset | amount          |
            | lp1    | ETH   | 100000000000000 |
            | party1 | ETH   | 10000000000000  |
            | party2 | ETH   | 10000000000000  |
            | party3 | ETH   | 10000000000000  |

        And the parties submit the following liquidity provision:
            | id  | party | market id | commitment amount | fee   | lp type    |
            | lp1 | lp1   | USD/DEC19 | 10000000000       | 0.001 | submission |
            | lp1 | lp1   | USD/DEC19 | 10000000000       | 0.001 | submission |
            | lp1 | lp1   | USD/DEC19 | 10000000000       | 0.001 | submission |
            | lp1 | lp1   | USD/DEC19 | 10000000000       | 0.001 | submission |
        And the parties place the following pegged iceberg orders:
            | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
            | lp1    | USD/DEC19 | 2         | 1                    | buy  | BID              | 66667  | 0      |
            | lp1    | USD/DEC19 | 2         | 1                    | buy  | MID              | 33334  | 1      |
            | lp1    | USD/DEC19 | 2         | 1                    | sell | MID              | 33334  | 1      |
            | lp1    | USD/DEC19 | 2         | 1                    | sell | ASK              | 66667  | 0      |
    
        Then the parties place the following orders:
            | party  | market id | side | volume | price   | resulting trades | type       | tif     |
            | party1 | USD/DEC19 | buy  | 10000  | 999999  | 0                | TYPE_LIMIT | TIF_GTC |
            | party1 | USD/DEC19 | buy  | 10000  | 1000000 | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | USD/DEC19 | sell | 10000  | 1000000 | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | USD/DEC19 | sell | 10000  | 1000001 | 0                | TYPE_LIMIT | TIF_GTC |

        Then the opening auction period ends for market "USD/DEC19"
        And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "USD/DEC19"
        And the market data for the market "USD/DEC19" should be:
            | mark price | trading mode            | best static bid price | static mid price | best static offer price |
            | 1000000    | TRADING_MODE_CONTINUOUS | 999999                | 1000000          | 1000001                 |
        Then the orders should have the following states:
            | party | market id | side | volume | price   | status        |
            | lp1   | USD/DEC19 | buy  | 66667  | 999999  | STATUS_ACTIVE |
            | lp1   | USD/DEC19 | buy  | 33334  | 999999  | STATUS_ACTIVE |
            | lp1   | USD/DEC19 | sell | 33334  | 1000001 | STATUS_ACTIVE |
            | lp1   | USD/DEC19 | sell | 66667  | 1000001 | STATUS_ACTIVE |

        Then the parties place the following pegged orders:
            | party  | market id | side | volume | pegged reference | offset |
            | party3 | USD/DEC19 | buy  | 5      | BID              | 0      |
            | party3 | USD/DEC19 | buy  | 4      | MID              | 1      |
            | party3 | USD/DEC19 | sell | 3      | ASK              | 0      |
            | party3 | USD/DEC19 | sell | 2      | MID              | 1      |

        Then the orders should have the following states:
            | party  | market id | side | volume | price   | status        |
            | lp1    | USD/DEC19 | buy  | 66667  | 999999  | STATUS_ACTIVE |
            | lp1    | USD/DEC19 | buy  | 33334  | 999999  | STATUS_ACTIVE |
            | lp1    | USD/DEC19 | sell | 33334  | 1000001 | STATUS_ACTIVE |
            | lp1    | USD/DEC19 | sell | 66667  | 1000001 | STATUS_ACTIVE |
            | party3 | USD/DEC19 | buy  | 5      | 999999  | STATUS_ACTIVE |
            | party3 | USD/DEC19 | buy  | 4      | 999999  | STATUS_ACTIVE |
            | party3 | USD/DEC19 | sell | 3      | 1000001 | STATUS_ACTIVE |
            | party3 | USD/DEC19 | sell | 2      | 1000001 | STATUS_ACTIVE |