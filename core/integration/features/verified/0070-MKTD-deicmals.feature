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
            | 7200000 | 0.99        | 300               |
        And the markets:
            | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor | sla params      |
            | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0              | 3                       | 1                      | 0                         | default-futures |
        # | USD/DEC19 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 3              | 3                       | 1                      | 0                         | default-futures |
        # | USD/DEC20 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 5                       | 1                      | 0                         | default-futures |
        # | USD/DEC21 | USD        | ETH   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 3                       | 1                      | 0                         | default-futures |
        And the parties deposit on asset's general account the following amount:
            | party  | asset | amount    |
            | party0 | USD   | 5000000   |
            | party0 | ETH   | 5000000   |
            | party1 | USD   | 2         |
            | party1 | ETH   | 100000000 |
            | party2 | USD   | 6         |
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
            | party3 | ETH/MAR22 | buy  | 100    | 4     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party3 | ETH/MAR22 | buy  | 100    | 5     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party3 | ETH/MAR22 | buy  | 100    | 6     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party3 | ETH/MAR22 | buy  | 100    | 7     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party3 | ETH/MAR22 | buy  | 100    | 8     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party3 | ETH/MAR22 | buy  | 100    | 9     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party1 | ETH/MAR22 | buy  | 5      | 10    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
            | party2 | ETH/MAR22 | sell | 5      | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
            | party4 | ETH/MAR22 | sell | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
            | party4 | ETH/MAR22 | sell | 100    | 11    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
            | party4 | ETH/MAR22 | sell | 100    | 12    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
            | party4 | ETH/MAR22 | sell | 100    | 13    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
            | party4 | ETH/MAR22 | sell | 100    | 14    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
            | party4 | ETH/MAR22 | sell | 100    | 25    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
            | party4 | ETH/MAR22 | sell | 100    | 30    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

        When the opening auction period ends for market "ETH/MAR22"

        And the market data for the market "ETH/MAR22" should be:
            | mark price | trading mode            | horizon | min bound | max bound |
            | 10         | TRADING_MODE_CONTINUOUS | 7200000 | 3         | 30        |

        And the parties should have the following account balances:
            | party  | asset | market id | margin | general | bond  |
            | party0 | USD   | ETH/MAR22 | 21342  | 4943089 | 35569 |
            | party1 | USD   | ETH/MAR22 | 1      | 1       |       |
            | party2 | USD   | ETH/MAR22 | 3      | 3       |       |

        And the following trades should be executed:
            | buyer  | price | size | seller |
            | party1 | 10    | 5    | party2 |

        #set mark price to 9
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | party4 | ETH/MAR22 | sell | 100    | 9     | 1                | TYPE_LIMIT | TIF_GTC | buy-ref-1 |

        And the following trades should be executed:
            | buyer  | price | size | seller |
            | party3 | 9     | 100  | party4 |

        When the network moves ahead "3" blocks
        And the market data for the market "ETH/MAR22" should be:
            | mark price | trading mode            | horizon | min bound | max bound |
            | 9          | TRADING_MODE_CONTINUOUS | 7200000 | 3         | 30        |

        And the parties should have the following account balances:
            | party  | asset | market id | margin | general | bond  |
            | party0 | USD   | ETH/MAR22 | 21342  | 4943089 | 35569 |
            | party1 | USD   | ETH/MAR22 | 1      | 1       |       |
            | party2 | USD   | ETH/MAR22 | 3      | 3       |       |

        #set mark price to 8
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | party4 | ETH/MAR22 | sell | 100    | 8     | 1                | TYPE_LIMIT | TIF_GTC | buy-ref-1 |

        When the network moves ahead "3" blocks
        And the market data for the market "ETH/MAR22" should be:
            | mark price | trading mode            | horizon | min bound | max bound |
            | 8          | TRADING_MODE_CONTINUOUS | 7200000 | 3         | 30        |

        And the parties should have the following account balances:
            | party  | asset | market id | margin | general | bond  |
            | party0 | USD   | ETH/MAR22 | 17073  | 4947358 | 35569 |
            | party1 | USD   | ETH/MAR22 | 1      | 1       |       |
            | party2 | USD   | ETH/MAR22 | 2      | 4       |       |

        Then the following transfers should happen:
            | from   | to     | from account        | to account           | market id | amount | asset |
            | party2 | party2 | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 1      | USD   |

        #set mark price to 7
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | party4 | ETH/MAR22 | sell | 100    | 7     | 1                | TYPE_LIMIT | TIF_GTC | buy-ref-1 |

        When the network moves ahead "3" blocks
        And the market data for the market "ETH/MAR22" should be:
            | mark price | trading mode            | horizon | min bound | max bound |
            | 7          | TRADING_MODE_CONTINUOUS | 7200000 | 3         | 30        |
        And the parties should have the following account balances:
            | party  | asset | market id | margin | general | bond  |
            | party0 | USD   | ETH/MAR22 | 17073  | 4947360 | 35569 |
            | party1 | USD   | ETH/MAR22 | 1      | 1       |       |
            | party2 | USD   | ETH/MAR22 | 2      | 4       |       |

        #set mark price to 6
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | party4 | ETH/MAR22 | sell | 100    | 6     | 1                | TYPE_LIMIT | TIF_GTC | buy-ref-1 |

        When the network moves ahead "3" blocks
        And the market data for the market "ETH/MAR22" should be:
            | mark price | trading mode            | horizon | min bound | max bound |
            | 6          | TRADING_MODE_CONTINUOUS | 7200000 | 3         | 30        |
        And the parties should have the following account balances:
            | party  | asset | market id | margin | general | bond  |
            | party0 | USD   | ETH/MAR22 | 12805  | 4951628 | 35569 |
            | party1 | USD   | ETH/MAR22 | 1      | 1       |       |
            | party2 | USD   | ETH/MAR22 | 2      | 4       |       |

        #set mark price to 5
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | party4 | ETH/MAR22 | sell | 100    | 5     | 1                | TYPE_LIMIT | TIF_GTC | buy-ref-1 |

        When the network moves ahead "3" blocks
        And the parties should have the following account balances:
            | party  | asset | market id | margin | general | bond  |
            | party0 | USD   | ETH/MAR22 | 10671  | 4953764 | 35569 |
            | party1 | USD   | ETH/MAR22 | 1      | 1       |       |
            | party2 | USD   | ETH/MAR22 | 2      | 4       |       |

        And the parties should have the following profit and loss:
            | party  | volume | unrealised pnl | realised pnl |
            | party1 | 5      | 0              | 0            |
            | party2 | -5     | 0              | 0            |

        #set mark price to 10
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | party3 | ETH/MAR22 | buy  | 100    | 10    | 1                | TYPE_LIMIT | TIF_GTC | buy-ref-1 |

        When the network moves ahead "3" blocks
        And the parties should have the following account balances:
            | party  | asset | market id | margin | general | bond  |
            | party0 | USD   | ETH/MAR22 | 21342  | 4943093 | 35569 |
            | party1 | USD   | ETH/MAR22 | 1      | 1       |       |
            | party2 | USD   | ETH/MAR22 | 3      | 3       |       |

        #set mark price to 11
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | party3 | ETH/MAR22 | buy  | 100    | 11    | 1                | TYPE_LIMIT | TIF_GTC | buy-ref-1 |

        When the network moves ahead "3" blocks
        And the parties should have the following account balances:
            | party  | asset | market id | margin | general | bond  |
            | party0 | USD   | ETH/MAR22 | 23475  | 4940962 | 35569 |
            | party1 | USD   | ETH/MAR22 | 1      | 1       |       |
            | party2 | USD   | ETH/MAR22 | 3      | 3       |       |

        #set mark price to 12
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | party3 | ETH/MAR22 | buy  | 100    | 12    | 1                | TYPE_LIMIT | TIF_GTC | buy-ref-1 |

        When the network moves ahead "3" blocks
        And the parties should have the following account balances:
            | party  | asset | market id | margin | general | bond  |
            | party0 | USD   | ETH/MAR22 | 25610  | 4938827 | 35569 |
            | party1 | USD   | ETH/MAR22 | 2      | 0       |       |
            | party2 | USD   | ETH/MAR22 | 3      | 3       |       |

        And the parties should have the following profit and loss:
            | party  | volume | unrealised pnl | realised pnl |
            | party1 | 5      | 0              | 0            |
            | party2 | -5     | 0              | 0            |








