Feature: Test one off transfers

Background:
    Given time is updated to "2021-08-26T00:00:00Z"
    Given the following network parameters are set:
      | name                                    | value |
      | transfer.fee.factor                     |  0.5  |
      | transfer.fee.factor                     |  0.5  |
      | network.markPriceUpdateMaximumFrequency | 0s    |


    Given the parties deposit on asset's general account the following amount:
    | party    | asset | amount          |
    | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c   | VEGA  | 10000000        |

Scenario: simple successful transfers (0057-TRAN-001, 0057-TRAN-007, 0057-TRAN-008)
    Given the parties submit the following one off transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | delivery_time         |
    | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 | 2021-08-26T00:00:01Z  |
    | 2  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL | 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 | ACCOUNT_TYPE_GENERAL | VEGA  |  20000 | 2021-08-26T00:00:02Z  |
    | 3  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | VEGA  |  30000 | 2021-08-26T00:00:03Z  |

    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9910000" for asset "VEGA"

    Given time is updated to "2021-08-26T00:00:01Z"
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9910000" for asset "VEGA"
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "10000" for asset "VEGA"

    Given time is updated to "2021-08-26T00:00:02Z"
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9910000" for asset "VEGA"
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "10000" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "20000" for asset "VEGA"

    Given time is updated to "2021-08-26T00:00:03Z"
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9910000" for asset "VEGA"
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "10000" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "20000" for asset "VEGA"
    Then "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "30000" for asset "VEGA"

Scenario: invalid transfers (0057-TRAN-005, 0057-TRAN-006)
     Given the parties submit the following one off transfers:
    | id | from   |              from_account_type           |   to   |         to_account_type          | asset | amount | delivery_time         |               error            |
    | 1  |        |  ACCOUNT_TYPE_GENERAL                    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:01Z  |  invalid from account          |
    | 2  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    |        | ACCOUNT_TYPE_GENERAL             | VEGA  |  20000 | 2021-08-26T00:00:02Z  |  invalid to account            |
    | 3  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  0     | 2021-08-26T00:00:03Z  |  cannot transfer zero funds    |
    | 4  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_UNSPECIFIED                | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 5  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_INSURANCE                  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 6  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_SETTLEMENT                 | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 7  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_MARGIN                     | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 8  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_FEES_INFRASTRUCTURE        | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 9  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_FEES_LIQUIDITY             | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 10 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_FEES_MAKER                 | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 11 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_LOCK_WITHDRAW              | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 12 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_BOND                       | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 13 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_EXTERNAL                   | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 14 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GLOBAL_INSURANCE           | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 15 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GLOBAL_REWARD              | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 16 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_PENDING_TRANSFERS          | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 17 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES     | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 18 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 19 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 20 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |
    | 21 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_UNSPECIFIED         | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 22 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_INSURANCE           | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 23 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_SETTLEMENT          | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 24 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_MARGIN              | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 25 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 26 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_FEES_LIQUIDITY      | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 27 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_FEES_MAKER          | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 28 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_LOCK_WITHDRAW       | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 29 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_BOND                | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 30 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_EXTERNAL            | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 31 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GLOBAL_INSURANCE    | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 32 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_PENDING_TRANSFERS   | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |

Scenario: transfer to self succeeds
    Given the parties submit the following one off transfers:
    | id | from   |   from_account_type    |   to   |   to_account_type    | asset | amount | delivery_time         |
    | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 | 2021-08-26T00:00:01Z  |

    Given time is updated to "2021-08-26T00:00:05Z"
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9995000" for asset "VEGA"

Scenario: transfer from non existing account fails
    Given the parties submit the following one off transfers:
    | id | from   |   from_account_type    |   to   |   to_account_type    | asset | amount | delivery_time         |                            error                                         |
    | 1  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |  ACCOUNT_TYPE_GENERAL  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 | 2021-08-26T00:00:01Z  | could not pay the fee for transfer: account does not exist: !a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4VEGA4 |


Scenario: payout time in the past - should be executed immediately
    Given the parties submit the following one off transfers:
    | id | from   |   from_account_type    |   to   |   to_account_type          | asset | amount | delivery_time         |
    | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL       | VEGA  |  10000 | 2021-08-25T00:00:01Z  |
    | 2  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |  ACCOUNT_TYPE_GENERAL  |   0000000000000000000000000000000000000000000000000000000000000000    | ACCOUNT_TYPE_GLOBAL_REWARD | VEGA  |  5000 | 2021-08-25T00:00:01Z   |

    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9985000" for asset "VEGA"
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "2500" for asset "VEGA"

Scenario: Transfer from general account to reward account (0057-TRAN-002, 0057-TRAN-007)
    Given the parties submit the following one off transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type                       | asset | amount | delivery_time         |
    | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL |    0000000000000000000000000000000000000000000000000000000000000000   | ACCOUNT_TYPE_GLOBAL_REWARD              | VEGA  |  10000 | 2021-08-25T00:00:00Z  |
    | 2  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL |    0000000000000000000000000000000000000000000000000000000000000000   | ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS    | VEGA  |  20000 | 2021-08-26T00:00:02Z  |
    | 3  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL |    0000000000000000000000000000000000000000000000000000000000000000   | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES    | VEGA  |  30000 | 2021-08-26T00:00:03Z  |
    | 4  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL |    0000000000000000000000000000000000000000000000000000000000000000   | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | VEGA  |  40000 | 2021-08-26T00:00:03Z  |
    | 5  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL |    0000000000000000000000000000000000000000000000000000000000000000   | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES     | VEGA  |  50000 | 2021-08-26T00:00:03Z  |

    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9775000" for asset "VEGA"

    # first transfer's time is in the past so done immediately
    And the reward account of type "ACCOUNT_TYPE_GLOBAL_REWARD" should have balance of "10000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS" should have balance of "0" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES" should have balance of "0" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES" should have balance of "0" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES" should have balance of "0" for asset "VEGA"

    #  advance to the payout to ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS
    Given time is updated to "2021-08-26T00:00:02Z"
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9775000" for asset "VEGA"
    Then the reward account of type "ACCOUNT_TYPE_GLOBAL_REWARD" should have balance of "10000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS" should have balance of "20000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES" should have balance of "0" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES" should have balance of "0" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES" should have balance of "0" for asset "VEGA"

    # advance to the payout to all other rewards,
    Given time is updated to "2021-08-26T00:00:03Z"
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9775000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_GLOBAL_REWARD" should have balance of "10000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS" should have balance of "20000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES" should have balance of "30000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES" should have balance of "40000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES" should have balance of "50000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_FEES_INFRASTRUCTURE" should have balance of "75000" for asset "VEGA"

Scenario: Insufficient funds to cover transfer + fees (0057-TRAN-007)
    Given the parties submit the following one off transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | delivery_time         |                            error                                 |
    | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | VEGA  | 10000  | 2021-08-25T00:00:00Z  |                                                                  |
    | 2  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |  ACCOUNT_TYPE_GENERAL | 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 | ACCOUNT_TYPE_GENERAL | VEGA  | 10000 | 2021-08-26T00:00:02Z   | could not pay the fee for transfer: not enough funds to transfer |


Scenario: Cannot cancel scheduled one off transfer (0057-TRAN-010)
    Given the parties submit the following one off transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | delivery_time         |
    | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | VEGA  | 10000  | 2021-08-28T00:00:00Z  |

   When the parties submit the following transfer cancellations:
    | party  | transfer_id |                error               |
    | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |      1      | recurring transfer does not exists |
