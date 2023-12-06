Feature: Test fee discounts for one off transfers

    Background:
        Given time is updated to "2021-08-26T00:00:00Z"
        And the following network parameters are set:
            | name                                    | value  |
            | transfer.fee.factor                     | 0.5    |
            | market.fee.factors.makerFee             | 0.004  |
            | market.fee.factors.infrastructureFee    | 0.002  |
            | network.markPriceUpdateMaximumFrequency | 0s     |
            | transfer.fee.maxQuantumAmount           | 100000 |
            | transfer.feeDiscountDecayFraction       | 0.9    |
        And the markets:
            | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
            | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-2 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       | default-futures |
        And the following network parameters are set:
            | name                                    | value |
            | market.auction.minimumDuration          | 1     |
            | limits.markets.maxPeggedOrders          | 1500  |
            | network.markPriceUpdateMaximumFrequency | 0s    |

        And the parties deposit on asset's general account the following amount:
            | party                                                            | asset | amount       |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 100000000000 |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 100000000000 |

        When the parties deposit on asset's general account the following amount:
            | party       | asset | amount       |
            | aux         | ETH   | 100000000000 |
            | aux2        | ETH   | 100000000000 |
            | lpprov      | ETH   | 100000000000 |
            | tx_to_party | ETH   | 100000000000 |
        Then the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee   | lp type    |
            | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
        # move market to continuous
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | lpprov | ETH/DEC19 | buy  | 1000   | 970   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2   | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux    | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | lpprov | ETH/DEC19 | sell | 1000   | 1035  | 0                | TYPE_LIMIT | TIF_GTC |
        And the opening auction period ends for market "ETH/DEC19"
        And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

        Given the parties place the following orders:
            | party                                                            | market id | side | volume | price | resulting trades | type       | tif     |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH/DEC19 | buy  | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH/DEC19 | sell | 1000   | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
        When the network moves ahead "1" epochs
        Then the following transfers should happen:
            | from                                                             | to                                                               | from account            | to account                       | market id | amount | asset |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market                                                           | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC19 | 4000   | ETH   |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market                                                           | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 2000   | ETH   |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market                                                           | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC19 | 1000   | ETH   |
            | market                                                           | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC19 | 4000   | ETH   |


    @transfer @fee-discount
    Scenario: 0057-TRAN-021 when a party paid taker fee g in previous epoch, and transfer.feeDiscountDecayFraction = 0.9, then in the next epoch when a party (did not generate any fees) makes a transfer and the theoretical fee the party should pay is f, fee-free amount is then c = 0.9 x g. If c > f, then no transfer fee is paid. And a party makes another transfer, and the theoretical fee the party should pay is f, then the party is not getting any fee-free discount
        # fee free discount total = 400000 + 200000 + 100000
        Given the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 7000               |

        # assert decay is 7000 * 0.9
        When the network moves ahead "1" epochs
        Then the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 6300               |

        # transfer depletes fees discount total
        Given the parties submit the following one off transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type      | asset | amount | delivery_time        |
            | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | ETH   | 12600  | 2021-08-26T00:00:10Z |
        And time is updated to "2021-08-26T00:00:10Z"
        When the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 0                  |
        Then the following transfers should happen:
            | from                                                             | to     | from account         | to account                       | market id | amount | asset |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 0      | ETH   |

        # one more transfer that will incur fees
        Given the parties submit the following one off transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type      | asset | amount | delivery_time        |
            | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | ETH   | 10000  | 2021-08-26T00:00:10Z |
        When the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 0                  |
        Then the following transfers should happen:
            | from                                                             | to     | from account         | to account                       | market id | amount | asset |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 5000   | ETH   |

    @transfer @fee-discount
    Scenario: 0057-TRAN-020 and 0057-TRAN-022 when a party made maker fee g in previous epoch, and transfer.feeDiscountDecayFraction = 0.9, then in the next epoch when a party (did not generate any fees) makes a transfer and the theoretical fee the party should pay is f, fee-free amount is then c = 0.9 x g. If c > f, then no transfer fee is paid. And a party makes another transfer, and the theoretical fee the party should pay is f, then the party is not getting any fee-free discount
        # 0057-TRAN-020: when a party made maker fee g in previous epoch, and transfer.feeDiscountDecayFraction = 0.9, then in the next epoch when a party (did not generate any fees) makes a transfer and the theoretical fee the party should pay is f, fee-free amount is then c = 0.9 x g. If c > f, then no transfer fee is paid
        # fee free discount total = maker fees made = 4000
        Given the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 4000               |

        # assert decay is 7000 * 0.9
        When the network moves ahead "1" epochs
        Then the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 3600               |

        # transfer depletes fees discount total 0057-TRAN-020
        Given the parties submit the following one off transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type      | asset | amount | delivery_time        |
            | 1  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | ETH   | 7200   | 2021-08-26T00:00:10Z |
        And time is updated to "2021-08-26T00:00:10Z"
        When the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 0                  |
        Then the following transfers should happen:
            | from                                                             | to     | from account         | to account                       | market id | amount | asset |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 0      | ETH   |

        # now we have used up the fee discount, one more transfer that will incur fees 0057-TRAN-022
        Given the parties submit the following one off transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type      | asset | amount | delivery_time        |
            | 1  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | ETH   | 10000  | 2021-08-26T00:00:10Z |
        When the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 0                  |
        Then the following transfers should happen:
            | from                                                             | to     | from account         | to account                       | market id | amount | asset |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 5000   | ETH   |

    @transfer @fee-discount
    Scenario: 0057-TRAN-027 when a party makes a transfer and f would be the theoretical fee the party should pay then the fee on the transfer that is actually charged is -min(f-c,0). The system subsequently updates c <- max(0,c-f). At the end of epoch, update c <- c x D and c <- c + all_trading_fees_for_trades_involved_in, if c < M x quantum(M is transfer.feeDiscountMinimumTrackedAmount), then set c <- 0
        # Scenario make a transfer that total discount < below transfer.feeDiscountMinimumTrackedAmount and next epoch check total discount = 0

        # fee free discount total = 4000 + 2000 + 1000
        Given the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 7000               |
        # min fee discount = transfer.feeDiscountMinimumTrackedAmount x quamtum = 0.5 x 100 = 50
        And the following network parameters are set:
            | name                                     | value |
            | transfer.feeDiscountMinimumTrackedAmount | 0.5   |
        And the following assets are updated:
            | id  | decimal places | quantum |
            | ETH | 0              | 100     |
        # assert decay is 7000 * 0.9
        When the network moves ahead "1" epochs
        Then the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 6300               |

        # transfer depletes fees discount total
        Given the parties submit the following one off transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type      | asset | amount | delivery_time        |
            | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | ETH   | 12490  | 2021-08-26T00:00:10Z |
        When time is updated to "2021-08-26T00:00:10Z"
        Then the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 55                 |

        Given the following transfers should happen:
            | from                                                             | to     | from account         | to account                       | market id | amount | asset |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 0      | ETH   |
        When the network moves ahead "1" epochs
        # fee discount decay = 55 * 0.9 = 49.5 < 50 so it becomes 0
        Then the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 0                  |

    @transfer @fee-discount
    Scenario: 0057-TRAN-027 when a party makes a transfer and f would be the theoretical fee the party should pay then the fee on the transfer that is actually charged is -min(f-c,0). The system subsequently updates c <- max(0,c-f). At the end of epoch, update c <- c x D and c <- c + all_trading_fees_for_trades_involved_in, if c < M x quantum(M is transfer.feeDiscountMinimumTrackedAmount), then set c <- 0
        # Scenario make a trade that generates discount > transfer.feeDiscountMinimumTrackedAmount and next epoch check total discount is retained

        # fee free discount total = 4000 + 2000 + 1000
        Given the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 7000               |
        # min fee discount = transfer.feeDiscountMinimumTrackedAmount x quamtum = 0.5 x 100 = 50
        And the following network parameters are set:
            | name                                     | value |
            | transfer.feeDiscountMinimumTrackedAmount | 0.5   |
        And the following assets are updated:
            | id  | decimal places | quantum |
            | ETH | 0              | 100     |

        # transfer depletes fees discount total
        Given the parties submit the following one off transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type      | asset | amount | delivery_time        |
            | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | ETH   | 13950  | 2021-08-26T00:00:10Z |
        When time is updated to "2021-08-26T00:00:10Z"
        Then the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 25                 |
        And the following transfers should happen:
            | from                                                             | to     | from account         | to account                       | market id | amount | asset |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 0      | ETH   |

        Given the parties place the following orders:
            | party                                                            | market id | side | volume | price | resulting trades | type       | tif     |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH/DEC19 | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH/DEC19 | sell | 5      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
        When the network moves ahead "1" epochs
        Then the following transfers should happen:
            | from                                                             | to                                                               | from account            | to account                       | market id | amount | asset |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market                                                           | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC19 | 20     | ETH   |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market                                                           | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 10     | ETH   |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market                                                           | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC19 | 5      | ETH   |
            | market                                                           | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC19 | 20     | ETH   |
        # fee discount previous epoch decayed to 25 * 0.9 = 22.5 + (trade fees - 20 + 10 + 5) = 57.5 > 50
        # trade at previous epoch generated total fee discount of 16 + 8 + 4 = 28 < 50 so it becomes 0
        And the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 57                 |

    @transfer @fee-discount
    Scenario: 0057-TRAN-027 when a party makes a transfer and f would be the theoretical fee the party should pay then the fee on the transfer that is actually charged is -min(f-c,0). The system subsequently updates c <- max(0,c-f). At the end of epoch, update c <- c x D and c <- c + all_trading_fees_for_trades_involved_in, if c < M x quantum(M is transfer.feeDiscountMinimumTrackedAmount), then set c <- 0
        # Scenario make a trade that generates discount but the total discount < transfer.feeDiscountMinimumTrackedAmount and next epoch check total discount is 0

        # fee free discount total = 4000 + 2000 + 1000
        Given the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 7000               |
        # min fee discount = transfer.feeDiscountMinimumTrackedAmount x quamtum = 0.5 x 100 = 50
        And the following network parameters are set:
            | name                                     | value |
            | transfer.feeDiscountMinimumTrackedAmount | 0.5   |
        And the following assets are updated:
            | id  | decimal places | quantum |
            | ETH | 0              | 100     |

        # transfer depletes fees discount total
        Given the parties submit the following one off transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type      | asset | amount | delivery_time        |
            | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | ETH   | 13950  | 2021-08-26T00:00:10Z |
        When time is updated to "2021-08-26T00:00:10Z"
        Then the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 25                 |
        And the following transfers should happen:
            | from                                                             | to     | from account         | to account                       | market id | amount | asset |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 0      | ETH   |

        Given the parties place the following orders:
            | party                                                            | market id | side | volume | price | resulting trades | type       | tif     |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH/DEC19 | buy  | 3      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH/DEC19 | sell | 3      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
        When the network moves ahead "1" epochs
        Then the following transfers should happen:
            | from                                                             | to                                                               | from account            | to account                       | market id | amount | asset |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market                                                           | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC19 | 12     | ETH   |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market                                                           | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 6      | ETH   |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | market                                                           | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC19 | 3      | ETH   |
            | market                                                           | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC19 | 12     | ETH   |
        # fee discount previous epoch decayed to 25 * 0.9 = 22.5 + (trade fees = 12 + 6 + 3=21) = 43.5 < 50
        # trade at previous epoch generated total fee discount of 16 + 8 + 4 = 28 < 50 so it becomes 0
        And the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH   | 0                  |

    @transfer @fee-discount
    Scenario: 0057-TRAN-024 when a party received maker fee f in previous epoch, and transfer.feeDiscountDecayFraction = 0.9, then in 3 epochs the fee-free discount amount would be c = 0.9^3 x f, when a party makes a transfer and the theoretical fee the party should pay is f1, and f1 <= 0.729 x f, then no amount is paid for transfer
        # fee free discount total = maker fees made = 4000
        # move ahead 3 epochs but check discount at each epoch
        Given the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 4000               |
        # assert decay is 3240 * 0.9
        When the network moves ahead "1" epochs
        Then the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 3600               |

        Given the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 3600               |
        # assert decay is 3600 * 0.9
        When the network moves ahead "1" epochs
        Then the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 3240               |

        Given the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 3240               |
        # assert decay is 3240 * 0.9
        When the network moves ahead "1" epochs
        Then the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 2916               |

        # transfer depletes fees discount total
        Given the parties submit the following one off transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type      | asset | amount | delivery_time        |
            | 1  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | ETH   | 5832   | 2021-08-26T00:00:10Z |
        And time is updated to "2021-08-26T00:00:10Z"
        When the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 0                  |
        Then the following transfers should happen:
            | from                                                             | to     | from account         | to account                       | market id | amount | asset |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 0      | ETH   |

        # one more transfer that will incur fees
        Given the parties submit the following one off transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type      | asset | amount | delivery_time        |
            | 1  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | ETH   | 10000  | 2021-08-26T00:00:10Z |
        When the parties have the following transfer fee discounts:
            | party                                                            | asset | available discount |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ETH   | 0                  |
        Then the following transfers should happen:
            | from                                                             | to     | from account         | to account                       | market id | amount | asset |
            | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 5000   | ETH   |
