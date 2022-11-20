Feature: Test one off transfers

Background:
    Given time is updated to "2021-08-26T00:00:00Z"
    Given the following network parameters are set:
      | name                                    | value |
      | transfer.fee.factor                     | 0.5   |
      | validators.epoch.length                 | 10s   |
      | transfer.minTransferQuantumMultiple     | 0.1   |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    Given the parties deposit on asset's general account the following amount:
    | party    | asset | amount          |
    | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c   | VEGA  | 10000000        |

    Then time is updated to "2021-08-26T00:00:12Z"
    Given the average block duration is "2"

Scenario: simple successful recurring transfers
    Given the parties submit the following recurring transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | start_epoch | end_epoch | factor |
    | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 |  1          |           | 0.5    |
    | 2  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL | 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 | ACCOUNT_TYPE_GENERAL | VEGA  |  20000 |  2          |     3     | 0.2    |

    # end of epoch 1 - transferring 10k from f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c to a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 + 5000 fees
    When the network moves ahead "14" blocks
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9985000" for asset "VEGA"
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "10000" for asset "VEGA"

    # end of epoch 2
    # transferring 5k from f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c to a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 + 2500 fees
    # transferring 20k from f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c to 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 + 10k fees
    When the network moves ahead "7" blocks
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9947500" for asset "VEGA"
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "15000" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "20000" for asset "VEGA"

    # end of epoch 3
    # transferring 2.5k from f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c to a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 + 1250 fees
    # transferring 4k from f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c to 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 + 2k fees
    When the network moves ahead "7" blocks
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9937750" for asset "VEGA"
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "17500" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "24000" for asset "VEGA"

    # end of epoch 4
    # transferring 1250 from f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c to a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 + 625 fees
    When the network moves ahead "7" blocks
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9935875" for asset "VEGA"
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "18750" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "24000" for asset "VEGA"

    When the parties submit the following transfer cancellations:
    | party  | transfer_id |
    | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |      1      |

    When the network moves ahead "7" blocks
    Then "f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c" should have general account balance of "9935875" for asset "VEGA"
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "18750" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "24000" for asset "VEGA"

Scenario: invalid recurring transfers
    Given the parties submit the following recurring transfers:
    | id | from   |              from_account_type           |   to   |         to_account_type          | asset | amount | start_epoch | end_epoch | factor |              error            |
    | 1  |        |  ACCOUNT_TYPE_GENERAL                    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | invalid from account          |
    | 2  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    |        | ACCOUNT_TYPE_GENERAL             | VEGA  |  20000 | 1           |           |   0.5  | invalid to account            |
    | 3  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  0     | 1           |           |   0.5  | cannot transfer zero funds    |
    | 4  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_UNSPECIFIED                | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 5  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_INSURANCE                  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 6  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_SETTLEMENT                 | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 7  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_MARGIN                     | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 8  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_FEES_INFRASTRUCTURE        | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 9  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_FEES_LIQUIDITY             | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 10 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_FEES_MAKER                 | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 11 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_LOCK_WITHDRAW              | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 12 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_BOND                       | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 13 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_EXTERNAL                   | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 14 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GLOBAL_INSURANCE           | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 15 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GLOBAL_REWARD              | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 16 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_PENDING_TRANSFERS          | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 17 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES     | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 18 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 19 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 20 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |
    | 21 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_UNSPECIFIED         | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type   |
    | 22 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_INSURANCE           | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type   |
    | 23 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_SETTLEMENT          | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type   |
    | 24 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_MARGIN              | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type   |
    | 25 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |
    | 26 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_FEES_LIQUIDITY      | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |
    | 27 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_FEES_MAKER          | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |
    | 28 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_LOCK_WITHDRAW       | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |
    | 29 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_BOND                | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |
    | 30 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_EXTERNAL            | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |
    | 31 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GLOBAL_INSURANCE    | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |
    | 32 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_PENDING_TRANSFERS   | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |
    | 33 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |      0    |   0.5  | end epoch is zero             |
    | 34 | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c |  ACCOUNT_TYPE_GENERAL                    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 0           |           |   0.5  | start epoch is zero           |

Scenario: As a user I can create a recurring transfer that decreases over time (0057-TRAN-050, 0057-TRAN-051)
    Given the parties deposit on asset's general account the following amount:
    | party    | asset |  amount |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4   | VEGA  | 1000000 |

    Given the parties submit the following recurring transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | start_epoch | end_epoch | factor |
    | 1  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |  ACCOUNT_TYPE_GENERAL | 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 |  2          |     5     |   0.7  |

    # end of epoch 1
    When the network moves ahead "14" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "1000000" for asset "VEGA"

    # end of epoch 2
    When the network moves ahead "7" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "985000" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "10000" for asset "VEGA"

    # end of epoch 3
    When the network moves ahead "7" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "974500" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "17000" for asset "VEGA"

    # end of epoch 4
    When the network moves ahead "7" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "967150" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "21900" for asset "VEGA"

    # end of epoch 5 - the transfer is ended so can't be cancelled
    When the network moves ahead "7" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "962005" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "25330" for asset "VEGA"

    When the parties submit the following transfer cancellations:
    | party  | transfer_id |               error                |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |      1      | recurring transfer does not exists |


Scenario: As a user I can create a recurring transfer that recurs forever, with the same balance transferred each time (0057-TRAN-052)
    Given the parties deposit on asset's general account the following amount:
    | party    | asset |  amount |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4   | VEGA  | 1000000 |

    Given the parties submit the following recurring transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | start_epoch | end_epoch | factor |
    | 1  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |  ACCOUNT_TYPE_GENERAL | 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 | ACCOUNT_TYPE_GENERAL | VEGA  |  1000  |  2          |           |   1    |

     # end of epoch 1
    When the network moves ahead "14" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "1000000" for asset "VEGA"

    # run for 100 epochs
    When the network moves ahead "700" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "850000" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "100000" for asset "VEGA"

Scenario: As a user I can create a recurring transfer that recurs as long as the amount is transfer.minTransferQuantumMultiple x quantum, with the amount transfer decreasing. (0057-TRAN-053)
    Given the parties deposit on asset's general account the following amount:
    | party    | asset | amount |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4   | VEGA  | 40000  |

    Given the parties submit the following recurring transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | start_epoch | end_epoch | factor |
    | 1  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |  ACCOUNT_TYPE_GENERAL | 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 |  2          |           |   0.1  |

    # end of epoch 1
    When the network moves ahead "14" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "40000" for asset "VEGA"

    # end of epoch 2
    When the network moves ahead "7" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "25000" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "10000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_FEES_INFRASTRUCTURE" should have balance of "5000" for asset "VEGA"

    # end of epoch 3
    When the network moves ahead "7" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "23500" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "11000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_FEES_INFRASTRUCTURE" should have balance of "5500" for asset "VEGA"

    # at this point the transfer amount for the next epoch is 1000*0.1 = 100 < 0.1 * quantum (=5k) = 500
    When the network moves ahead "100" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "23500" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "11000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_FEES_INFRASTRUCTURE" should have balance of "5500" for asset "VEGA"


Scenario: As a user I can cancel a recurring transfer (0057-TRAN-054)
    Given the parties deposit on asset's general account the following amount:
    | party    | asset | amount |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4   | VEGA  | 40000  |

    Given the parties submit the following recurring transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | start_epoch | end_epoch | factor |
    | 1  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |  ACCOUNT_TYPE_GENERAL | 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 |  2          |           |   1    |

    # end of epoch 2
    When the network moves ahead "21" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "25000" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "10000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_FEES_INFRASTRUCTURE" should have balance of "5000" for asset "VEGA"

    When the parties submit the following transfer cancellations:
    | party  | transfer_id |               error                |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |      1      |                                    |

   # progress a few epoch - the balance of a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 and 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 should not have changed as the transfer has been cancelled
    When the network moves ahead "100" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "25000" for asset "VEGA"

    # we can't cancel it again because it's already cancelled
    When the parties submit the following transfer cancellations:
    | party  | transfer_id |               error                |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |      1      | recurring transfer does not exists |

Scenario: As a user I can cancel a recurring transfer before any transfers have executed (0057-TRAN-055)
    Given the parties deposit on asset's general account the following amount:
    | party    | asset | amount |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4   | VEGA  | 40000  |

    Given the parties submit the following recurring transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | start_epoch | end_epoch | factor |
    | 1  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |  ACCOUNT_TYPE_GENERAL | 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 |  2          |           |   1    |

    When the parties submit the following transfer cancellations:
    | party  | transfer_id |               error                |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |      1      |                                    |

    # progress a few epoch - the balance of a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 and 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 should not have changed as the transfer has been cancelled
    When the network moves ahead "100" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "40000" for asset "VEGA"

    # we can't cancel it again because it's already cancelled
    When the parties submit the following transfer cancellations:
    | party  | transfer_id |               error                |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |      1      | recurring transfer does not exists |


Scenario: A user's recurring transfer is cancelled if any transfer fails due to insufficient funds (0057-TRAN-054)
    Given the parties deposit on asset's general account the following amount:
    | party    | asset | amount |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4   | VEGA  | 40000  |

    Given the parties submit the following recurring transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | start_epoch | end_epoch | factor |
    | 1  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |  ACCOUNT_TYPE_GENERAL | 576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303 | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 |  2          |           |   1    |

    # end of epoch 0
    When the network moves ahead "7" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "40000" for asset "VEGA"

    # end of epoch 1
    When the network moves ahead "7" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "40000" for asset "VEGA"

    # end of epoch 2
    When the network moves ahead "7" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "25000" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "10000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_FEES_INFRASTRUCTURE" should have balance of "5000" for asset "VEGA"

    # end of epoch 3
    When the network moves ahead "7" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "10000" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "20000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_FEES_INFRASTRUCTURE" should have balance of "10000" for asset "VEGA"

    # end of epoch 4 - there's insufficient funds to execute the transfer - it gets cancelled
    When the network moves ahead "7" blocks
    Then "a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4" should have general account balance of "10000" for asset "VEGA"
    Then "576380694832d9271682e86fffbbcebc09ca79c259baa5d4d0298e12ecdee303" should have general account balance of "20000" for asset "VEGA"
    And the reward account of type "ACCOUNT_TYPE_FEES_INFRASTRUCTURE" should have balance of "10000" for asset "VEGA"

    When the parties submit the following transfer cancellations:
    | party  | transfer_id |               error                |
    | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 |      1      | recurring transfer does not exists |
