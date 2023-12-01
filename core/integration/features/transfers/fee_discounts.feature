Feature: Transfer fee discounts


# this sets up party f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c with a tranfer discount of 324
Background:
    Given time is updated to "2021-08-26T00:00:00Z"

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | position decimal places | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 2                       | 1e6                    | 1e6                       | default-futures |
    And the following network parameters are set:
      | name                                               | value |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |

    Given the following network parameters are set:
      | name                                    | value |
      | transfer.fee.factor                     |  1    |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | transfer.fee.maxQuantumAmount           |  1    |
      | transfer.feeDiscountDecayFraction       |  0.9  |
      | limits.markets.maxPeggedOrders          | 4     |
      | validators.epoch.length                 | 20s   |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | aux1    | ETH   | 100000000 |
      | aux2    | ETH   | 100000000 |
      | trader3 | ETH   | 10000     |
      | trader4 | ETH   | 10000     |
      | lpprov  | ETH   | 10000000  |
      | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c  | ETH   | 10000000  |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC21 | 10000     | 1                    | buy  | BID              | 20000  | 100    |
      | lpprov | ETH/DEC21 | 10000     | 1                    | sell | ASK              | 20000  | 100    |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |
    When the parties place the following orders "1" blocks apart:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ETH/DEC21 | buy  | 300    | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4 | ETH/DEC21 | sell | 400    | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer   | price | size | seller  | aggressor side |
      | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | 1002  | 300  | trader4 | sell           |

    And the following transfers should happen:
      | from    | to      | from account            | to account                       | market id | amount | asset |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 16     | ETH   |
      | trader4 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 7      | ETH   |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 301    | ETH   |
      | market  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 16     | ETH   |

    # make the epoch end so that the fees are registered
    And the network moves ahead "1" epochs
    And the current epoch is "2"

    
@fee-discount
Scenario: transfer where fee-discount has decayed to 0 results in no fee discount (0057-TRAN-016)

    And the parties have the following transfer fee discounts:
    | party                                                              | asset | available discount |
    | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c   |  ETH  | 324                |
    
    # let the discount decay to zero
    Given the following network parameters are set:
      | name                                    | value |
      | transfer.feeDiscountDecayFraction       |  0.0  |
    And the network moves ahead "1" epochs

    And the parties have the following transfer fee discounts:
    | party                                                              | asset | available discount |
    | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c   |  ETH  | 0                  |


    Given "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9998686" for asset "ETH"
    And the accumulated infrastructure fees should be "7" for the asset "ETH"

    # now do a transfer and check we pay the full fees
    # They will transfers 10000, fee is 0.5 * 10000 = 5000, discount is 0
    # 9998686 - 10000 - 5000 = 9983686
    Given the parties submit the following one off transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | delivery_time         |
    | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | ETH  |  10000 | 2021-08-26T00:09:01Z  |
  
    Given time is updated to "2021-08-26T00:10:01Z"
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9983686" for asset "ETH"
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "10000" for asset "ETH"

    # check fee went to insurance account
    And the accumulated infrastructure fees should be "5007" for the asset "ETH"

    # now try to transfer the rest, we won't have enough so it should get rejected
    Given the parties submit the following one off transfers:
    | id | from                                                             |  from_account_type    |   to                                                             |   to_account_type    | asset | amount   | delivery_time         | error                        |
    | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | ETH   |  9983686 | 2021-08-26T00:09:01Z  | could not pay the fee for transfer: not enough funds to transfer | 
  