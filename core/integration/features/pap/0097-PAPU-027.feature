Feature: Given a network with two PAP programs, A and B, funded from the same account with a balance of 1000. If the snapshot of program A is triggered and is allocated 750 tokens for it's next auction, once the snapshot of program B is triggered it will only be allocated 250 tokens for it's next auction. This happens regardless of whether the auction of program B is triggered before the auction of program A. (0097-PAPU-027).
    Background:
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
            | id       | name     | base asset | quote asset | liquidity monitoring | risk model            | auction duration | fees          | price monitoring | sla params |
            | BTC/ETH  | BTC/ETH  | BTC        | ETH         | lqm-params           | log-normal-risk-model | 2                | fees-config-1 | price-monitoring | SLA-22     |
            | BTC/ETH2 | BTC/ETH2 | BTC        | ETH         | lqm-params           | log-normal-risk-model | 2                | fees-config-1 | price-monitoring | SLA-22     |

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
            | lp1 | lpprov | BTC/ETH2  | 937000            | 0.1 | submission |
            | lp1 | lpprov | BTC/ETH2  | 937000            | 0.1 | submission |
        And the parties place the following pegged iceberg orders:
            | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
            | lpprov | BTC/ETH   | 2         | 1                    | buy  | MID              | 50     | 100    |
            | lpprov | BTC/ETH   | 2         | 1                    | sell | MID              | 50     | 100    |
            | lpprov | BTC/ETH2  | 2         | 1                    | buy  | MID              | 50     | 100    |
            | lpprov | BTC/ETH2  | 2         | 1                    | sell | MID              | 50     | 100    |

        # place orders and generate trades - slippage 100
        And the parties place the following orders:
            | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
            | party2 | BTC/ETH   | buy  | 1      | 950000  | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
            | party1 | BTC/ETH   | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
            | party3 | BTC/ETH   | sell | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
            | party2 | BTC/ETH2  | buy  | 1      | 950000  | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
            | party1 | BTC/ETH2  | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
            | party3 | BTC/ETH2  | sell | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |


        When the opening auction period ends for market "BTC/ETH"
        And the mark price should be "1000000" for the market "BTC/ETH"
        And the mark price should be "1000000" for the market "BTC/ETH2"

        And the composite price oracles from "0xCAFECAFE2":
            | name         | price property   | price type   | price decimals |
            | price_oracle | prices.ETH.value | TYPE_INTEGER | 0              |

        And the time triggers oracle spec is:
            | name                       | initial | every |
            | auction_vol_snap_schedule1 | 5       | 30    |
            | auction_vol_snap_schedule2 | 10      | 30    |
            | auction_schedule1          | 12      | 30    |
            | auction_schedule2          | 11      | 30    |



        And the average block duration is "1"

        And the parties deposit on asset's general account the following amount:
            | party                                                            | asset | amount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | BTC   | 1000   |
        Given time is updated to "2024-09-24T00:00:00Z"
        And the parties submit the following one off transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type            | asset | amount | delivery_time        |
            | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_BUY_BACK_FEES | BTC   | 1000   | 2024-09-23T00:00:00Z |

        And the buy back fees balance should be "1000" for the asset "BTC"

    Scenario: setting up multiple markets with pap program from the same account. The first market with the volume snapshot is earmarking as much as it can and the second market earmarks whatever is left although the second one's auction starts before the first one. (0097-PAPU-027)
        When the protocol automated purchase is defined as:
            | id    | from | from account type          | to account type               | market id | price oracle | price oracle staleness tolerance | oracle offset factor | auction schedule oracle | auction volume snapshot schedule oracle | auction duration | minimum auction size | maximum auction size | expiry timestamp |
            | 12345 | BTC  | ACCOUNT_TYPE_BUY_BACK_FEES | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | price_oracle | 10s                              | 1.01                 | auction_schedule1       | auction_vol_snap_schedule1              | 60s              | 100                  | 750                  | 0                |
            | 54321 | BTC  | ACCOUNT_TYPE_BUY_BACK_FEES | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH2  | price_oracle | 10s                              | 1.01                 | auction_schedule2       | auction_vol_snap_schedule2              | 60s              | 100                  | 750                  | 0                |

        Then the oracles broadcast data with block time signed with "0xCAFECAFE2":
            | name             | value   | time offset |
            | prices.ETH.value | 1000000 | -1s         |

        And the network moves ahead "30" blocks

        # the volume snapshot for the market BTC/ETH ticks first so it earmarks the maximum it is allowed, i.e. 750 
        # the volume snapshot for the market BTC/ETH2 ticks second so it earmarks the maximum it is has remaining, i.e. 250 
        Then the automated purchase program for market "BTC/ETH" should have a snapshot balance of "750"
        Then the automated purchase program for market "BTC/ETH2" should have a snapshot balance of "250"

        # both enter a pap auction
        And the trading mode should be "TRADING_MODE_PROTOCOL_AUTOMATED_PURCHASE_AUCTION" for the market "BTC/ETH"
        And the trading mode should be "TRADING_MODE_PROTOCOL_AUTOMATED_PURCHASE_AUCTION" for the market "BTC/ETH2"

        # for BTC/ETH: an order for sell BTC with size 750 and price 1.01 * 1000000 is placed
        And the order book should have the following volumes for market "BTC/ETH":
            | side | price   | volume |
            | sell | 1010000 | 750    |

         # for BTC/ETH2: an order for sell BTC with size 250 and price 1.01 * 1000000 is placed
        And the order book should have the following volumes for market "BTC/ETH2":
            | side | price   | volume |
            | sell | 1010000 | 250    |
