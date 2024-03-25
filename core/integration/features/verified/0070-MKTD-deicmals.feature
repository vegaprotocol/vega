Feature: Allow markets to be specified with a smaller number of decimal places than the underlying settlement asset

    Background:
        Given the following network parameters are set:
            | name                                    | value |
            | market.liquidity.bondPenaltyParameter   | 0.2   |
            | limits.markets.maxPeggedOrders          | 1500  |
            | network.markPriceUpdateMaximumFrequency | 0s    |
            | limits.markets.maxPeggedOrders          | 12    |
            | network.markPriceUpdateMaximumFrequency | 2s    |
        Given the liquidity monitoring parameters:
            | name       | triggering ratio | time window | scaling factor |
            | lqm-params | 0.1              | 24h         | 1.0            |

        And the following assets are registered:
            | id  | decimal places |
            | ETH | 1              |
            | USD | 1              |
        And the average block duration is "1"
        And the log normal risk model named "log-normal-risk-model-1":
            | risk aversion | tau | mu | r | sigma |
            | 0.000001      | 0.1 | 0  | 0 | 1.0   |
        And the fees configuration named "fees-config-1":
            | maker fee | infrastructure fee |
            | 0.004     | 0.001              |
        And the price monitoring named "price-monitoring-1":
            | horizon | probability | auction extension |
            | 360000  | 0.99        | 300               |
        And the markets:
            | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor | sla params      |
            | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0              | 2                       | 0.1                    | 0                         | default-futures |
        # | USD/DEC19 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 3              | 3                       | 1                      | 0                         | default-futures |
        # | USD/DEC20 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 5                       | 1                      | 0                         | default-futures |
        # | USD/DEC21 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 3                       | 1                      | 0                         | default-futures |
        And the parties deposit on asset's general account the following amount:
            | party  | asset | amount    |
            | party0 | USD   | 5000000   |
            | party0 | ETH   | 5000000   |
            | party1 | USD   | 100000000 |
            | party1 | ETH   | 100000000 |
            | party2 | USD   | 100000000 |
            | party2 | ETH   | 100000000 |
            | party3 | USD   | 100000000 |
            | party4 | USD   | 100000000 |
            | lpprov | ETH   | 100000000 |
            | lpprov | USD   | 100000000 |

    Scenario: 001: Users engage in a USD market auction
        Given the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee   | lp type    |
            | lp1 | party0 | ETH/MAR22 | 35569             | 0.001 | submission |
            | lp1 | party0 | ETH/MAR22 | 35569             | 0.001 | amendment  |
        And the parties place the following pegged iceberg orders:
            | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
            | party0 | ETH/MAR22 | 2         | 1                    | sell | ASK              | 50000  | 20     |
            | party0 | ETH/MAR22 | 2         | 1                    | buy  | BID              | 50000  | 20     |
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
            | party3 | ETH/MAR22 | buy  | 100    | 8     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party3 | ETH/MAR22 | buy  | 100    | 9     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party1 | ETH/MAR22 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
            | party2 | ETH/MAR22 | sell | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
            | party4 | ETH/MAR22 | sell | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
            | party4 | ETH/MAR22 | sell | 100    | 11    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

        And the markets are updated:
            | id        | linear slippage factor | quadratic slippage factor |
            | ETH/MAR22 | 1                      | 0                         |

        When the opening auction period ends for market "ETH/MAR22"

        And the market data for the market "ETH/MAR22" should be:
            | mark price | trading mode            | horizon | min bound | max bound |
            | 10         | TRADING_MODE_CONTINUOUS | 360000  | 8         | 13        |

        And the parties should have the following account balances:
            | party  | asset | market id | margin | general  | bond  |
            | party0 | USD   | ETH/MAR22 | 213415 | 4751016  | 35569 |
            | party1 | USD   | ETH/MAR22 | 2      | 99999998 |       |
            | party2 | USD   | ETH/MAR22 | 6      | 99999994 |       |

        And the following trades should be executed:
            | buyer  | price | size | seller |
            | party1 | 10    | 1    | party2 |

        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | party4 | ETH/MAR22 | sell | 100    | 9     | 1                | TYPE_LIMIT | TIF_GTC | buy-ref-1 |

        And the following trades should be executed:
            | buyer  | price | size | seller |
            | party3 | 9     | 100  | party4 |

        When the network moves ahead "3" blocks
        And the market data for the market "ETH/MAR22" should be:
            | mark price | trading mode            | horizon | min bound | max bound |
            | 9          | TRADING_MODE_CONTINUOUS | 360000  | 8         | 13        |

        And the parties should have the following account balances:
            | party  | asset | market id | margin | general  | bond  |
            | party0 | USD   | ETH/MAR22 | 213415 | 4751016  | 35569 |
            | party1 | USD   | ETH/MAR22 | 2      | 99999998 |       |
            | party2 | USD   | ETH/MAR22 | 6      | 99999994 |       |

        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | party4 | ETH/MAR22 | sell | 100    | 8     | 1                | TYPE_LIMIT | TIF_GTC | buy-ref-1 |

        And the following trades should be executed:
            | buyer  | price | size | seller |
            | party3 | 8     | 100  | party4 |

        When the network moves ahead "3" blocks
        And the market data for the market "ETH/MAR22" should be:
            | mark price | trading mode            | horizon | min bound | max bound |
            | 8          | TRADING_MODE_CONTINUOUS | 360000  | 8         | 13        |

        And the parties should have the following account balances:
            | party  | asset | market id | margin | general  | bond  |
            | party0 | USD   | ETH/MAR22 | 170732 | 4793699  | 35569 |
            | party1 | USD   | ETH/MAR22 | 2      | 99999998 |       |
            | party2 | USD   | ETH/MAR22 | 4      | 99999996 |       |

        Then the following transfers should happen:
            | from   | to     | from account        | to account           | market id | amount | asset |
            | party2 | party2 | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 2      | USD   |




