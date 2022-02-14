Feature: Test one off transfers

Background:
    Given time is updated to "2021-08-26T00:00:00Z"
    Given the following network parameters are set:
      | name                 |  value  |
      | transfer.fee.factor  |  0.5    |
      | transfer.fee.factor  |  0.5    |

    
    Given the parties deposit on asset's general account the following amount:
    | party    | asset | amount          |
    | party1   | VEGA  | 10000000        |

Scenario: simple successful transfers
    Given the parties submit the following one off transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | delivery_time         |
    | 1  | party1 |  ACCOUNT_TYPE_GENERAL | party2 | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 | 2021-08-26T00:00:01Z  |    
    | 2  | party1 |  ACCOUNT_TYPE_GENERAL | party3 | ACCOUNT_TYPE_GENERAL | VEGA  |  20000 | 2021-08-26T00:00:02Z  |    
    | 3  | party1 |  ACCOUNT_TYPE_GENERAL | party4 | ACCOUNT_TYPE_GENERAL | VEGA  |  30000 | 2021-08-26T00:00:03Z  |    

    Then "party1" should have general account balance of "9910000" for asset "VEGA"

    Given time is updated to "2021-08-26T00:00:01Z"
    Then "party1" should have general account balance of "9910000" for asset "VEGA"
    Then "party2" should have general account balance of "10000" for asset "VEGA"

    Given time is updated to "2021-08-26T00:00:02Z"
    Then "party1" should have general account balance of "9910000" for asset "VEGA"
    Then "party2" should have general account balance of "10000" for asset "VEGA"
    Then "party3" should have general account balance of "20000" for asset "VEGA"
    
    Given time is updated to "2021-08-26T00:00:03Z"
    Then "party1" should have general account balance of "9910000" for asset "VEGA"
    Then "party2" should have general account balance of "10000" for asset "VEGA"
    Then "party3" should have general account balance of "20000" for asset "VEGA"
    Then "party4" should have general account balance of "30000" for asset "VEGA"
    
Scenario: invalid transfers
     Given the parties submit the following one off transfers:
    | id | from   |              from_account_type           |   to   |         to_account_type          | asset | amount | delivery_time         |               error            |
    | 1  |        |  ACCOUNT_TYPE_GENERAL                    | party2 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:01Z  |  invalid from account          |  
    | 2  | party1 |  ACCOUNT_TYPE_GENERAL                    |        | ACCOUNT_TYPE_GENERAL             | VEGA  |  20000 | 2021-08-26T00:00:02Z  |  invalid to account            |  
    | 3  | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  0     | 2021-08-26T00:00:03Z  |  cannot transfer zero funds    |  
    | 4  | party1 |  ACCOUNT_TYPE_UNSPECIFIED                | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 5  | party1 |  ACCOUNT_TYPE_INSURANCE                  | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 6  | party1 |  ACCOUNT_TYPE_SETTLEMENT                 | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 7  | party1 |  ACCOUNT_TYPE_MARGIN                     | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 8  | party1 |  ACCOUNT_TYPE_FEES_INFRASTRUCTURE        | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 9  | party1 |  ACCOUNT_TYPE_FEES_LIQUIDITY             | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 10 | party1 |  ACCOUNT_TYPE_FEES_MAKER                 | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 11 | party1 |  ACCOUNT_TYPE_LOCK_WITHDRAW              | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 12 | party1 |  ACCOUNT_TYPE_BOND                       | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 13 | party1 |  ACCOUNT_TYPE_EXTERNAL                   | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 14 | party1 |  ACCOUNT_TYPE_GLOBAL_INSURANCE           | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 15 | party1 |  ACCOUNT_TYPE_GLOBAL_REWARD              | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 16 | party1 |  ACCOUNT_TYPE_PENDING_TRANSFERS          | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 17 | party1 |  ACCOUNT_TYPE_REWARD_TAKER_PAID_FEES     | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 18 | party1 |  ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 19 | party1 |  ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES    | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 20 | party1 |  ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS    | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported from account type |  
    | 21 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_UNSPECIFIED         | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |
    | 22 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_INSURANCE           | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |  
    | 23 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_SETTLEMENT          | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |  
    | 24 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_MARGIN              | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |  
    | 25 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |  
    | 26 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_FEES_LIQUIDITY      | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |  
    | 27 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_FEES_MAKER          | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |  
    | 28 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_LOCK_WITHDRAW       | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |  
    | 29 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_BOND                | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |  
    | 30 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_EXTERNAL            | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |  
    | 31 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_GLOBAL_INSURANCE    | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |  
    | 32 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_PENDING_TRANSFERS   | VEGA  |  10000 | 2021-08-26T00:00:03Z  |  unsupported to account type   |  

Scenario: transfer to self succeeds
    Given the parties submit the following one off transfers:
    | id | from   |   from_account_type    |   to   |   to_account_type    | asset | amount | delivery_time         |
    | 1  | party1 |  ACCOUNT_TYPE_GENERAL  | party1 | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 | 2021-08-26T00:00:01Z  |

    Given time is updated to "2021-08-26T00:00:05Z"
    Then "party1" should have general account balance of "9995000" for asset "VEGA"

Scenario: transfer from non existing account fails
    Given the parties submit the following one off transfers:
    | id | from   |   from_account_type    |   to   |   to_account_type    | asset | amount | delivery_time         |                            error                                         |
    | 1  | party2 |  ACCOUNT_TYPE_GENERAL  | party1 | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 | 2021-08-26T00:00:01Z  | could not pay the fee for transfer: account does not exist: !party2VEGA4 |


Scenario: payout time in the past - should be executed immediately
    Given the parties submit the following one off transfers:
    | id | from   |   from_account_type    |   to   |   to_account_type          | asset | amount | delivery_time         |
    | 1  | party1 |  ACCOUNT_TYPE_GENERAL  | party2 | ACCOUNT_TYPE_GENERAL       | VEGA  |  10000 | 2021-08-25T00:00:01Z  |
    | 2  | party2 |  ACCOUNT_TYPE_GENERAL  |   *    | ACCOUNT_TYPE_GLOBAL_REWARD | VEGA  |  5000 | 2021-08-25T00:00:01Z   |

    Then "party1" should have general account balance of "9985000" for asset "VEGA"
    Then "party2" should have general account balance of "2500" for asset "VEGA"
    
