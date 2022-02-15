Feature: Test one off transfers

Background:
    Given time is updated to "2021-08-26T00:00:00Z"
    Given the following network parameters are set:
      | name                     |  value  |
      | transfer.fee.factor      |  0.5    |
      | validators.epoch.length  |  10s    |
    
    Given the parties deposit on asset's general account the following amount:
    | party    | asset | amount          |
    | party1   | VEGA  | 10000000        |

    Then time is updated to "2021-08-26T00:00:12Z"
    Given the average block duration is "2"
   
Scenario: simple successful recurring transfers
    Given the parties submit the following recurring transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | start_epoch | end_epoch | factor |
    | 1  | party1 |  ACCOUNT_TYPE_GENERAL | party2 | ACCOUNT_TYPE_GENERAL | VEGA  |  10000 |  1          |           | 0.5    |   
    | 2  | party1 |  ACCOUNT_TYPE_GENERAL | party3 | ACCOUNT_TYPE_GENERAL | VEGA  |  20000 |  2          |     3     | 0.2    |
    
    # end of epoch 1 - transferring 10k from party1 to party2 + 5000 fees
    When the network moves ahead "14" blocks
    Then "party1" should have general account balance of "9985000" for asset "VEGA"
    Then "party2" should have general account balance of "10000" for asset "VEGA"

    # end of epoch 2
    # transferring 5k from party1 to party2 + 2500 fees
    # transferring 20k from party1 to party3 + 10k fees
    When the network moves ahead "7" blocks
    Then "party1" should have general account balance of "9947500" for asset "VEGA"
    Then "party2" should have general account balance of "15000" for asset "VEGA"
    Then "party3" should have general account balance of "20000" for asset "VEGA"

    # end of epoch 3
    # transferring 2.5k from party1 to party2 + 1250 fees
    # transferring 4k from party1 to party3 + 2k fees
    When the network moves ahead "7" blocks
    Then "party1" should have general account balance of "9937750" for asset "VEGA"
    Then "party2" should have general account balance of "17500" for asset "VEGA"
    Then "party3" should have general account balance of "24000" for asset "VEGA"

    # end of epoch 4 
    # transferring 1250 from party1 to party2 + 625 fees
    When the network moves ahead "7" blocks
    Then "party1" should have general account balance of "9935875" for asset "VEGA"
    Then "party2" should have general account balance of "18750" for asset "VEGA"
    Then "party3" should have general account balance of "24000" for asset "VEGA"

    When the parties submit the following transfer cancellations:
    | party  | transfer_id |
    | party1 |      1      |

    When the network moves ahead "7" blocks
    Then "party1" should have general account balance of "9935875" for asset "VEGA"
    Then "party2" should have general account balance of "18750" for asset "VEGA"
    Then "party3" should have general account balance of "24000" for asset "VEGA"

Scenario: invalid recurring transfers
    Given the parties submit the following recurring transfers:
    | id | from   |              from_account_type           |   to   |         to_account_type          | asset | amount | start_epoch | end_epoch | factor |              error            |
    | 1  |        |  ACCOUNT_TYPE_GENERAL                    | party2 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | invalid from account          |  
    | 2  | party1 |  ACCOUNT_TYPE_GENERAL                    |        | ACCOUNT_TYPE_GENERAL             | VEGA  |  20000 | 1           |           |   0.5  | invalid to account            |  
    | 3  | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  0     | 1           |           |   0.5  | cannot transfer zero funds    |  
    | 4  | party1 |  ACCOUNT_TYPE_UNSPECIFIED                | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |  
    | 5  | party1 |  ACCOUNT_TYPE_INSURANCE                  | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |  
    | 6  | party1 |  ACCOUNT_TYPE_SETTLEMENT                 | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |  
    | 7  | party1 |  ACCOUNT_TYPE_MARGIN                     | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 8  | party1 |  ACCOUNT_TYPE_FEES_INFRASTRUCTURE        | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 9  | party1 |  ACCOUNT_TYPE_FEES_LIQUIDITY             | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 10 | party1 |  ACCOUNT_TYPE_FEES_MAKER                 | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 11 | party1 |  ACCOUNT_TYPE_LOCK_WITHDRAW              | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 12 | party1 |  ACCOUNT_TYPE_BOND                       | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 13 | party1 |  ACCOUNT_TYPE_EXTERNAL                   | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 14 | party1 |  ACCOUNT_TYPE_GLOBAL_INSURANCE           | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 15 | party1 |  ACCOUNT_TYPE_GLOBAL_REWARD              | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 16 | party1 |  ACCOUNT_TYPE_PENDING_TRANSFERS          | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 17 | party1 |  ACCOUNT_TYPE_REWARD_TAKER_PAID_FEES     | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 18 | party1 |  ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 19 | party1 |  ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES    | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 20 | party1 |  ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS    | party4 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |           |   0.5  | unsupported from account type |   
    | 21 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_UNSPECIFIED         | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type   |  
    | 22 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_INSURANCE           | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type   |  
    | 23 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_SETTLEMENT          | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type   |  
    | 24 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_MARGIN              | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type   |  
    | 25 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |    
    | 26 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_FEES_LIQUIDITY      | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |    
    | 27 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_FEES_MAKER          | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |    
    | 28 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_LOCK_WITHDRAW       | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |    
    | 29 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_BOND                | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |    
    | 30 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_EXTERNAL            | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |    
    | 31 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_GLOBAL_INSURANCE    | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |    
    | 32 | party1 |  ACCOUNT_TYPE_GENERAL                    | party4 | ACCOUNT_TYPE_PENDING_TRANSFERS   | VEGA  |  10000 | 1           |           |   0.5  | unsupported to account type |    
    | 33 | party1 |  ACCOUNT_TYPE_GENERAL                    | party2 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 1           |      0    |   0.5  | end epoch is zero             |   
    | 34 | party1 |  ACCOUNT_TYPE_GENERAL                    | party2 | ACCOUNT_TYPE_GENERAL             | VEGA  |  10000 | 0           |           |   0.5  | start epoch is zero           |   