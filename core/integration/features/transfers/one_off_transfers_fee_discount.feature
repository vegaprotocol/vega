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

        And the following assets are updated:
            | id   | decimal places | quantum |
            | VEGA | 0              | 50000   |

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

    @transfer @fee-discount @sp-test
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
