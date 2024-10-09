Feature: Given the program currently has no funds earmarked for an auction, if a program's expiry timestamp is reached, the program will be cancelled and no further auctions will take place. (0097-PAPU-021).

    Background:
        Given time is updated to "2024-09-24T00:00:00Z"
        And the average block duration is "1"
        Given the log normal risk model named "log-normal-risk-model":
            | risk aversion | tau | mu | r | sigma |
            | 0.000001      | 0.1 | 0  | 0 | 1.0   |
        And the following network parameters are set:
            | name                                    | value |
            | market.value.windowLength               | 60s   |
            | network.markPriceUpdateMaximumFrequency | 0s    |
            | limits.markets.maxPeggedOrders          | 6     |
            | market.auction.minimumDuration          | 1     |
            | market.fee.factors.infrastructureFee    | 0.001 |
            | market.fee.factors.makerFee             | 0.004 |
            | spam.protection.max.stopOrdersPerMarket | 5     |
            | validators.epoch.length                 | 60m   |
        And the liquidity monitoring parameters:
            | name       | triggering ratio | time window | scaling factor |
            | lqm-params | 1.0              | 20s         | 1              |
        And the fees configuration named "fees-config-1":
            | maker fee | infrastructure fee |
            | 0.0004    | 0.001              |
        And the price monitoring named "price-monitoring":
            | horizon | probability | auction extension |
            | 3600    | 0.99        | 30                |
        And the liquidity sla params named "SLA-22":
            | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
            | 0.5         | 0.6                          | 1                             | 1.0                    |
        And the following network parameters are set:
            | name                           | value |
            | limits.markets.maxPeggedOrders | 2     |
        And the following assets are registered:
            | id  | decimal places |
            | ETH | 0              |

        And the spot markets:
            | id      | name    | base asset | quote asset | liquidity monitoring | risk model            | auction duration | fees          | price monitoring | sla params |
            | BTC/ETH | BTC/ETH | BTC        | ETH         | lqm-params           | log-normal-risk-model | 2                | fees-config-1 | price-monitoring | SLA-22     |

        Given the parties deposit on asset's general account the following amount:
            | party  | asset | amount     |
            | party1 | ETH   | 1000000000 |
            | party2 | ETH   | 1000000000 |
            | party3 | ETH   | 1000000000 |
            | party1 | BTC   | 1000000000 |
            | party3 | BTC   | 1000000000 |
            | lpprov | ETH   | 1000000000 |
            | lpprov | BTC   | 1000000000 |

        When the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee | lp type    |
            | lp1 | lpprov | BTC/ETH   | 937000            | 0.1 | submission |
            | lp1 | lpprov | BTC/ETH   | 937000            | 0.1 | submission |
        And the parties place the following pegged iceberg orders:
            | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
            | lpprov | BTC/ETH   | 2         | 1                    | buy  | MID              | 50     | 100    |
            | lpprov | BTC/ETH   | 2         | 1                    | sell | MID              | 50     | 100    |

        # place orders and generate trades - slippage 100
        And the parties place the following orders:
            | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
            | party2 | BTC/ETH   | buy  | 1      | 950000  | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
            | party1 | BTC/ETH   | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
            | party3 | BTC/ETH   | sell | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |

        When the opening auction period ends for market "BTC/ETH"

        And the following trades should be executed:
            | buyer  | price   | size | seller |
            | party1 | 1000000 | 1    | party3 |
        And the mark price should be "1000000" for the market "BTC/ETH"

        And the composite price oracles from "0xCAFECAFE2":
            | name         | price property   | price type   | price decimals |
            | price_oracle | prices.ETH.value | TYPE_INTEGER | 0              |

        And the time triggers oracle spec is:
            | name                      | initial   | every |
            | auction_schedule          | 172713601 | 30    |
            | auction_vol_snap_schedule | 172713600 | 30    |

        And the parties deposit on asset's general account the following amount:
            | party                                                            | asset | amount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | BTC   | 50000  |
       

    Scenario: pap is not funded and expires nothing gets earmarked (0097-PAPU-021)
        When the protocol automated purchase is defined as:
            | id    | from | from account type          | to account type               | market id | price oracle | price oracle staleness tolerance | oracle offset factor | auction schedule oracle | auction volume snapshot schedule oracle | auction duration | minimum auction size | maximum auction size | expiry timestamp |
            | 12345 | BTC  | ACCOUNT_TYPE_BUY_BACK_FEES | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | price_oracle | 10s                              | 1.01                 | auction_schedule        | auction_vol_snap_schedule               | 60s              | 100                  | 200                  | 1727136030       |

        And the active pap id should be "12345" for the market "BTC/ETH"

        Then the oracles broadcast data with block time signed with "0xCAFECAFE2":
            | name             | value   | time offset |
            | prices.ETH.value | 1000000 | -1s         |

        And the network moves ahead "120" blocks
        And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
        And the active pap id should be "none" for the market "BTC/ETH"
        And the active pap order id should be "none" for the market "BTC/ETH"



