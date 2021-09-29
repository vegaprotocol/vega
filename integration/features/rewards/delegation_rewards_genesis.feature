Feature: Staking & Delegation - scenarios focusing on initial epoch

  Background:
    Given the following network parameters are set:
      | name                                              |  value  |
      | reward.asset                                      |  VEGA   |
      | validators.epoch.length                           |  24h    |
      | validators.delegation.minAmount                   |  10     |
      | reward.staking.delegation.payoutDelay             |  0s     |
      | reward.staking.delegation.delegatorShare          |  0.883  |
      | reward.staking.delegation.minimumValidatorStake   |  100    |
      | reward.staking.delegation.payoutFraction          |  0.5    |
      | reward.staking.delegation.maxPayoutPerParticipant |  100000 |
      | reward.staking.delegation.competitionLevel        |  1.1    |
      | reward.staking.delegation.maxPayoutPerEpoch       |  50000  |
  
    And the average block duration is "1"
    And time is updated to "2021-09-10T00:00:00Z"
 
    And the validators:
      | id     | staking account balance |
      | node1  |         1000000         |
      | node2  |         1000000         |
      | node3  |         1000000         |
      | node4  |         1000000         |
      | node5  |         1000000         |
      | node6  |         1000000         |
      | node7  |         1000000         |
      | node8  |         1000000         |
      | node9  |         1000000         |
      | node10 |         1000000         |
      | node11 |         1000000         |
      | node12 |         1000000         |
      | node13 |         1000000         |

  Scenario: No delegation in the first epoch

    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
    
    Then the network moves ahead "172804" blocks

    And the parties receive the following reward for epoch 1:
      | party  | asset | amount |
      | node1  | VEGA  |     0  | 
      | node2  | VEGA  |     0  | 
      | node3  | VEGA  |     0  | 
      | node4  | VEGA  |     0  | 
      | node5  | VEGA  |     0  | 
      | node6  | VEGA  |     0  | 
      | node8  | VEGA  |     0  | 
      | node10 | VEGA  |     0  | 
      | node11 | VEGA  |     0  | 
      | node12 | VEGA  |     0  | 
      | node13 | VEGA  |     0  | 

    Then the network moves ahead "86403" blocks

    And the parties receive the following reward for epoch 2:
      | party  | asset | amount |
      | node1  | VEGA  |     0  | 
      | node2  | VEGA  |     0  | 
      | node3  | VEGA  |     0  | 
      | node4  | VEGA  |     0  | 
      | node5  | VEGA  |     0  | 
      | node6  | VEGA  |     0  | 
      | node8  | VEGA  |     0  | 
      | node10 | VEGA  |     0  | 
      | node11 | VEGA  |     0  | 
      | node12 | VEGA  |     0  | 
      | node13 | VEGA  |     0  | 

    And the parties deposit on asset's general account the following amount:
      | party  | asset  | amount |
      | party1 | VEGA   | 111000 |

    And the parties deposit on staking account the following amount:
      | party  | asset  | amount |
      | party1 | VEGA   | 111000 |  

    #set up the self delegation of the validators (number of validators < min. validators parameter)
    Then the parties submit the following delegations:
      | party  | node id  | amount |
      | node1  |  node1   |  11000 | 
      | node2  |  node2   |  12000 | 
      | node3  |  node3   |  13000 | 
      | node4  |  node4   |     99 | 
      | party1 |  node4   | 111000 | 

    And the parties should have the following delegation balances for epoch 4:
      | party  | node id  | amount |
      | node1  |  node1   |  11000 | 
      | node2  |  node2   |  12000 |       
      | node3  |  node3   |  13000 |  
      | node4  |  node4   |     99 |  
      | party1 |  node4   | 111000 |  

    Then the network moves ahead "172804" blocks

    And the parties should have the following delegation balances for epoch 4:
      | party  | node id  | amount |
      | node1  |  node1   |  11000 | 
      | node2  |  node2   |  12000 |       
      | node3  |  node3   |  12446 |  
      | node4  |  node4   |     99 |  
      | party1 |  node4   |  12347 |  
    
    Then the validators should have the following val scores for epoch 4:
      | node id | validator score  | normalised score |
      |  node1  |      0.08462     |     0.25000      |    
      |  node2  |      0.08462     |     0.25000      |
      |  node3  |      0.08462     |     0.25000      | 
      |  node4  |      0.08462     |     0.25000      | 
      |  node5  |      0.00000     |     0.00000      | 
      |  node6  |      0.00000     |     0.00000      | 
      |  node7  |      0.00000     |     0.00000      | 
      |  node8  |      0.00000     |     0.00000      | 
      |  node9  |      0.00000     |     0.00000      | 
      |  node10 |      0.00000     |     0.00000      | 
      |  node11 |      0.00000     |     0.00000      | 
      |  node12 |      0.00000     |     0.00000      | 
      |  node13 |      0.00000     |     0.00000      |

    And the parties receive the following reward for epoch 4:
      | party  | asset | amount |
      | party1 | VEGA  | 10949  | 
      | node1  | VEGA  | 12500  | 
      | node2  | VEGA  | 12500  | 
      | node3  | VEGA  | 12500  | 
      | node4  | VEGA  |     0  | 
      | node5  | VEGA  |     0  | 
      | node6  | VEGA  |     0  | 
      | node8  | VEGA  |     0  | 
      | node10 | VEGA  |     0  | 
      | node11 | VEGA  |     0  | 
      | node12 | VEGA  |     0  | 
      | node13 | VEGA  |     0  | 

  Scenario: Only a few validators self-delegate, no delegation

    #set up the self delegation of the validators (number of validators = min. validators parameter)
    Then the parties submit the following delegations:
      | party  | node id  | amount |
      | node1  |  node1   | 11000  | 
      | node2  |  node2   | 12000  |       
      | node3  |  node3   | 13000  | 
      | node4  |  node4   | 14000  | 
      | node5  |  node5   | 15000  | 

    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
    
    #complete the initial epoch for delegation to take effect
    Then the network moves ahead "172804" blocks

    Then the validators should have the following val scores for epoch 1:
      | node id | validator score  | normalised score |
      |  node1  |      0.08462     |     0.20000      |    
      |  node2  |      0.08462     |     0.20000      |
      |  node3  |      0.08462     |     0.20000      | 
      |  node4  |      0.08462     |     0.20000      | 
      |  node5  |      0.08462     |     0.20000      | 
      |  node6  |      0.00000     |     0.00000      | 
      |  node7  |      0.00000     |     0.00000      | 
      |  node8  |      0.00000     |     0.00000      | 
      |  node9  |      0.00000     |     0.00000      | 
      |  node10 |      0.00000     |     0.00000      | 
      |  node11 |      0.00000     |     0.00000      | 
      |  node12 |      0.00000     |     0.00000      | 
      |  node13 |      0.00000     |     0.00000      | 

    And the parties receive the following reward for epoch 1:
      | party  | asset | amount |
      | node1  | VEGA  | 10000  | 
      | node2  | VEGA  | 10000  | 
      | node3  | VEGA  | 10000  | 
      | node4  | VEGA  | 10000  | 
      | node5  | VEGA  | 10000  | 
      | node6  | VEGA  |     0  | 
      | node8  | VEGA  |     0  | 
      | node10 | VEGA  |     0  | 
      | node11 | VEGA  |     0  | 
      | node12 | VEGA  |     0  | 
      | node13 | VEGA  |     0  | 

  Scenario: Only a few validators self-delegate, small delegation to a single validator (with own stake). Some validators delegate over max delegatable amount.

    And the parties deposit on asset's general account the following amount:
      | party  | asset  | amount |
      | party1 | VEGA   |     10 |
      | party2 | VEGA   |     50 |
      | party3 | VEGA   |    200 |

    And the parties deposit on staking account the following amount:
      | party  | asset  | amount |
      | party1 | VEGA   |     10 |  
      | party2 | VEGA   |     50 |  
      | party3 | VEGA   |    200 |  

    Then the parties submit the following delegations:
      | party  | node id  |  amount |
      | party1 |  node1   |      10 | 
      | party2 |  node1   |      50 | 
      | party3 |  node1   |     200 | 

    #set up the self delegation of the validators (number of validators = min. validators parameter)
    Then the parties submit the following delegations:
      | party  | node id  | amount |
      | node1  |  node1   |   100  | 
      | node2  |  node2   |   200  |       
      | node3  |  node3   |   300  | 
      | node4  |  node4   |   400  | 
      | node5  |  node5   |   500  | 

    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
    
    #complete the initial epoch for delegation to take effect
    Then the network moves ahead "172804" blocks

    And the parties should have the following delegation balances for epoch 1:
      | party  | node id  |  amount |
      | node1  |  node1   |     100 | 
      | node2  |  node2   |     148 |       
      | node3  |  node3   |     148 |  
      | party1 |  node1   |      10 |  
      | party2 |  node1   |      38 |  
      | party3 |  node1   |       0 |  

    Then the validators should have the following val scores for epoch 1:
      | node id | validator score  | normalised score |
      |  node1  |      0.08462     |     0.20000      |    
      |  node2  |      0.08462     |     0.20000      |
      |  node3  |      0.08462     |     0.20000      | 
      |  node4  |      0.08462     |     0.20000      | 
      |  node5  |      0.08462     |     0.20000      | 
      |  node6  |      0.00000     |     0.00000      | 
      |  node7  |      0.00000     |     0.00000      | 
      |  node8  |      0.00000     |     0.00000      | 
      |  node9  |      0.00000     |     0.00000      | 
      |  node10 |      0.00000     |     0.00000      | 
      |  node11 |      0.00000     |     0.00000      | 
      |  node12 |      0.00000     |     0.00000      | 
      |  node13 |      0.00000     |     0.00000      | 

    And the parties receive the following reward for epoch 1:
      | party  | asset | amount |
      | party1 | VEGA  |   596  | 
      | party2 | VEGA  |  2266  | 
      | party3 | VEGA  |     0  | 
      | node1  | VEGA  |  7136  | 
      | node2  | VEGA  | 10000  | 
      | node3  | VEGA  | 10000  |  
      | node4  | VEGA  | 10000  | 
      | node5  | VEGA  | 10000  | 
      | node6  | VEGA  | 0      | 
      | node7  | VEGA  | 0      | 
      | node8  | VEGA  | 0      | 
      | node10 | VEGA  | 0      | 
      | node11 | VEGA  | 0      | 
      | node12 | VEGA  | 0      | 
      | node13 | VEGA  | 0      | 

  Scenario: Only a few validators self-delegate, significant delegation to a three validators only (one w/o own stake)

    And the parties deposit on asset's general account the following amount:
      | party  | asset  | amount |
      | party1 | VEGA   | 111000 |
      | party2 | VEGA   | 222000 |
      | party3 | VEGA   | 333000 |
      | party4 | VEGA   | 444000 |
      | party5 | VEGA   | 555000 |
  
    And the parties deposit on staking account the following amount:
      | party  | asset  | amount |
      | party1 | VEGA   | 111000 |  
      | party2 | VEGA   | 222000 |  
      | party3 | VEGA   | 333000 |  
      | party4 | VEGA   | 444000 |  
      | party5 | VEGA   | 555000 |  
  
    Then the parties submit the following delegations:
      | party  | node id  | amount |
      | node1  |  node1   | 11000  | 
      | node2  |  node2   | 12000  |       
      | node3  |  node3   | 13000  | 
      | node4  |  node4   | 14000  | 
      | node5  |  node5   | 15000  | 
      | node6  |  node6   | 16000  | 
  
    Then the parties submit the following delegations:
      | party  | node id  | amount  |
      | party1 |  node1   |  111000 | 
      | party2 |  node2   |  111000 | 
      | party2 |  node7   |  111000 | 
      | party3 |  node1   |  111000 | 
      | party3 |  node2   |  111000 | 
      | party3 |  node7   |  111000 | 
      | party4 |  node1   |  222000 | 
      | party4 |  node7   |  222000 | 
      | party5 |  node2   |  555000 | 
  
    #set up the self delegation of the validators (number of validators > min. validators parameter)
    And the parties should have the following delegation balances for epoch 1:
      | party  | node id  | amount  |
      | node1  |  node1   |   11000 | 
      | node2  |  node2   |   12000 |       
      | node3  |  node3   |   13000 |  
      | node4  |  node4   |   14000 |  
      | node5  |  node5   |   15000 |  
      | node6  |  node6   |   16000 |  
      | node7  |  node7   |       0 |  
      | party1 |  node1   |  111000 | 
      | party2 |  node2   |  111000 | 
      | party2 |  node7   |  111000 | 
      | party3 |  node1   |  111000 | 
      | party3 |  node2   |  111000 | 
      | party3 |  node7   |  111000 | 
      | party4 |  node1   |  222000 | 
      | party4 |  node7   |  222000 | 
      | party5 |  node2   |  555000 | 
  
    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
    
    #complete the initial epoch for delegation to take effect
    Then the network moves ahead "172804" blocks
  
    And the parties should have the following delegation balances for epoch 1:
      | party  | node id  |  amount |
      | node1  |  node1   |   11000 | 
      | node2  |  node2   |   12000 |       
      | node3  |  node3   |   13000 |  
      | node4  |  node4   |   14000 |  
      | node5  |  node5   |   15000 |  
      | node6  |  node6   |   16000 |  
      | node7  |  node7   |       0 |  
      | party1 |  node1   |  111000 | 
      | party2 |  node2   |  111000 | 
      | party2 |  node7   |  111000 | 
      | party3 |  node1   |   25738 | 
      | party3 |  node2   |   24738 | 
      | party3 |  node7   |   36738 | 
      | party4 |  node1   |       0 | 
      | party4 |  node7   |       0 | 
      | party5 |  node2   |       0 | 
  
    Then the validators should have the following val scores for epoch 1:
      | node id | validator score  | normalised score |
      |  node1  |      0.08462     |     0.22896      |    
      |  node2  |      0.08462     |     0.22896      |
      |  node3  |      0.02594     |     0.07018      | 
      |  node4  |      0.02793     |     0.07558      | 
      |  node5  |      0.02993     |     0.08098      | 
      |  node6  |      0.03192     |     0.08638      | 
      |  node7  |      0.08462     |     0.22896      | 
      |  node8  |      0.00000     |     0.00000      | 
      |  node9  |      0.00000     |     0.00000      | 
      |  node10 |      0.00000     |     0.00000      | 
      |  node11 |      0.00000     |     0.00000      | 
      |  node12 |      0.00000     |     0.00000      | 
      |  node13 |      0.00000     |     0.00000      | 
  
    And the parties receive the following reward for epoch 1:
      | party  | asset | amount |
      | party1 | VEGA  | 7594   | 
      | party2 | VEGA  | 15188  | 
      | party3 | VEGA  | 5965   | 
      | node1  | VEGA  | 2092   | 
      | node2  | VEGA  | 2160   | 
      | node3  | VEGA  | 3509   |  
      | node4  | VEGA  | 3779   | 
      | node5  | VEGA  | 4048   | 
      | node6  | VEGA  | 4318   | 
      | node7  | VEGA  | 0      | 
      | node8  | VEGA  | 0      | 
      | node10 | VEGA  | 0      | 
      | node11 | VEGA  | 0      | 
      | node12 | VEGA  | 0      | 
      | node13 | VEGA  | 0      | 

Scenario: Validator owns more tokens than the minimumValidatorStake, but most of them are delegated to a different validator, then withdraws so that he owns less than minimumValidatorStake

  And the parties deposit on asset's general account the following amount:
    | party  | asset  | amount |
    | party1 | VEGA   | 111000 |
    | party2 | VEGA   | 222000 |

  And the parties deposit on staking account the following amount:
    | party  | asset  | amount |
    | party1 | VEGA   | 111000 |  
    | party2 | VEGA   | 222000 |   

  Then the parties submit the following delegations:
    | party  | node id  | amount |
    | node1  |  node1   |  11000 | 
    | node2  |  node2   |     20 |       
    | node3  |  node3   |     30 | 
    | node4  |  node4   |  14000 | 
    | node5  |  node5   |  15000 | 
    | node6  |  node6   |  16000 | 
    | node8  |  node8   |    110 |       
    | node2  |  node7   |    180 |       
    | node3  |  node7   |   3000 | 

  Then the parties submit the following delegations:
    | party  | node id  | amount  |
    | party1 |  node1   |  111000 | 
    | party2 |  node2   |  222000 | 

  #set up the self delegation of the validators (number of validators > min. validators parameter)
  And the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | node1  |  node1   |  11000 | 
    | node2  |  node2   |     20 |       
    | node3  |  node3   |     30 | 
    | node4  |  node4   |  14000 | 
    | node5  |  node5   |  15000 | 
    | node6  |  node6   |  16000 | 
    | node8  |  node8   |    110 |  
    | node2  |  node7   |    180 |       
    | node3  |  node7   |   3000 | 
    | party1 |  node1   | 111000 | 
    | party2 |  node2   | 222000 | 

    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
    
    #complete the initial epoch for delegation to take effect
    Then the network moves ahead "172804" blocks

    And the parties should have the following delegation balances for epoch 1:
      | party  | node id  | amount |
      | node1  |  node1   |  11000 | 
      | node2  |  node2   |     20 |       
      | node3  |  node3   |     30 | 
      | node4  |  node4   |  14000 | 
      | node5  |  node5   |  15000 | 
      | node6  |  node6   |  16000 | 
      | node8  |  node8   |    110 |       
      | node2  |  node7   |    180 |       
      | node3  |  node7   |   3000 | 
      | party1 |  node1   |  22197 | 
      | party2 |  node2   |  33177 | 

    Then the validators should have the following val scores for epoch 1:
      | node id | validator score  | normalised score |
      |  node1  |      0.08462     |     0.18719      |    
      |  node2  |      0.08462     |     0.18719      |
      |  node3  |      0.00026     |     0.00058      | 
      |  node4  |      0.08462     |     0.18719      | 
      |  node5  |      0.08462     |     0.18719      | 
      |  node6  |      0.08462     |     0.18719      | 
      |  node7  |      0.02772     |     0.06133      | 
      |  node8  |      0.00096     |     0.00212      | 
      |  node9  |      0.00000     |     0.00000      | 
      |  node10 |      0.00000     |     0.00000      | 
      |  node11 |      0.00000     |     0.00000      | 
      |  node12 |      0.00000     |     0.00000      | 
      |  node13 |      0.00000     |     0.00000      | 

    And the parties receive the following reward for epoch 1:
      | party  | asset | amount |
      | party1 | VEGA  | 5526   | 
      | party2 | VEGA  | 8259   | 
      | node1  | VEGA  | 3833   | 
      | node2  | VEGA  | 153    | 
      | node3  | VEGA  | 2553   |  
      | node4  | VEGA  | 9359   | 
      | node5  | VEGA  | 9359   | 
      | node6  | VEGA  | 9359   | 
      | node7  | VEGA  | 0      | 
      | node8  | VEGA  | 106    | 
      | node10 | VEGA  | 0      | 
      | node11 | VEGA  | 0      | 
      | node12 | VEGA  | 0      | 
      | node13 | VEGA  | 0      | 

   # Leave 20 in the account
   Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | node2  | VEGA   | 999980 | 

  And the parties submit the following undelegations:
    | party | node id | amount | when |
    | node3 |  node7  |   2900 | now  |
    | node8 |  node8  |     60 | now  |

  Then the network moves ahead "1" blocks

  # Delegation changes due to undelegation are immediate, need to complete the epoch for withdrawal to get registered
  And the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | node1  |  node1   |  11000 | 
    | node2  |  node2   |     20 |       
    | node3  |  node3   |     30 | 
    | node4  |  node4   |  14000 | 
    | node5  |  node5   |  15000 | 
    | node6  |  node6   |  16000 | 
    | node8  |  node8   |     50 |       
    | node2  |  node7   |    180 |       
    | node3  |  node7   |    100 | 
    | party1 |  node1   |  22197 | 
    | party2 |  node2   |  33177 | 

  Then the network moves ahead "86401" blocks

  And the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | node1  |  node1   |  11000 | 
    | node2  |  node2   |      2 |       
    | node3  |  node3   |     30 | 
    | node4  |  node4   |  14000 | 
    | node5  |  node5   |  15000 | 
    | node6  |  node6   |  16000 | 
    | node8  |  node8   |     50 |       
    | node2  |  node7   |     18 |       
    | node3  |  node7   |    100 | 
    | party1 |  node1   |  22197 | 
    | party2 |  node2   |  33177 | 

Scenario: In presence of max delegation cap self-delegation gets priorities even if submitted later

  Given the parties deposit on asset's general account the following amount:
    | party  | asset  | amount |
    | party1 | VEGA   | 111000 |
    | party2 | VEGA   | 222000 |

  And the parties deposit on staking account the following amount:
    | party  | asset  | amount |
    | party1 | VEGA   | 111000 |  
    | party2 | VEGA   | 222000 |   

  Then the parties submit the following delegations:
    | party  | node id  | amount  |
    | party1 |  node1   |  111000 | 
    | party2 |  node2   |  222000 | 

  Then the network moves ahead "1" blocks

  Then the parties submit the following delegations:
    | party  | node id  | amount |
    | node1  |  node1   | 100000 | 
    | node2  |  node2   |  12000 |       
    | node3  |  node3   |  13000 | 
    | node4  |  node4   |  14000 | 
    | node5  |  node5   |  15000 | 
    | node6  |  node6   |  16000 | 

  Then the network moves ahead "1" blocks

  And the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | node1  |  node1   | 100000 | 
    | node2  |  node2   |  12000 |       
    | node3  |  node3   |  13000 | 
    | node4  |  node4   |  14000 | 
    | node5  |  node5   |  15000 | 
    | node6  |  node6   |  16000 | 
    | party1 |  node1   | 111000 | 
    | party2 |  node2   | 222000 | 

    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
    
    #complete the initial epoch for delegation to take effect
    Then the network moves ahead "172802" blocks

    And the parties should have the following delegation balances for epoch 1:
      | party  | node id  | amount |
      | node1  |  node1   |  42561 | 
      | node2  |  node2   |  12000 |       
      | node3  |  node3   |  13000 | 
      | node4  |  node4   |  14000 | 
      | node5  |  node5   |  15000 | 
      | node6  |  node6   |  16000 | 
      | party1 |  node1   |      0 | 
      | party2 |  node2   |  30561 | 

    Then the validators should have the following val scores for epoch 1:
      | node id | validator score  | normalised score |
      |  node1  |      0.08462     |     0.16667      |    
      |  node2  |      0.08462     |     0.16667      |
      |  node3  |      0.08462     |     0.16667      | 
      |  node4  |      0.08462     |     0.16667      | 
      |  node5  |      0.08462     |     0.16667      | 
      |  node6  |      0.08462     |     0.16667      | 
      |  node7  |      0.00000     |     0.00000      | 
      |  node8  |      0.00000     |     0.00000      | 
      |  node9  |      0.00000     |     0.00000      | 
      |  node10 |      0.00000     |     0.00000      | 
      |  node11 |      0.00000     |     0.00000      | 
      |  node12 |      0.00000     |     0.00000      | 
      |  node13 |      0.00000     |     0.00000      | 

    And the parties receive the following reward for epoch 1:
      | party  | asset | amount |
      | party1 | VEGA  | 0      | 
      | party2 | VEGA  | 5283   | 
      | node1  | VEGA  | 8333   | 
      | node2  | VEGA  | 3049   | 
      | node3  | VEGA  | 8333   |  
      | node4  | VEGA  | 8333   | 
      | node5  | VEGA  | 8333   | 
      | node6  | VEGA  | 8333   | 
      | node7  | VEGA  | 0      | 
      | node8  | VEGA  | 0      | 
      | node10 | VEGA  | 0      | 
      | node11 | VEGA  | 0      | 
      | node12 | VEGA  | 0      | 
      | node13 | VEGA  | 0      | 

Scenario: Validator subset can self-delegate as to push themselves below min validator stake due to max delegatable amount cap

  Then the parties submit the following delegations:
    | party  | node id  | amount |
    | node1  |  node1   |    100 | 
    | node2  |  node2   |    200 |       
    | node3  |  node3   |    300 | 

  Then the network moves ahead "1" blocks

  And the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | node1  |  node1   |    100 | 
    | node2  |  node2   |    200 |       
    | node3  |  node3   |    300 | 
    | node4  |  node4   |      0 | 

    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
    
    #complete the initial epoch for delegation to take effect
    Then the network moves ahead "172802" blocks

    And the parties should have the following delegation balances for epoch 1:
      | party  | node id  | amount |
      | node1  |  node1   |     50 | 
      | node2  |  node2   |     50 |       
      | node3  |  node3   |     50 | 
      | node4  |  node4   |     0  | 

    And the parties receive the following reward for epoch 1:
      | party  | asset | amount |
      | node1  | VEGA  | 0      | 
      | node2  | VEGA  | 0      | 
      | node3  | VEGA  | 0      |  
      | node4  | VEGA  | 0      | 
      | node5  | VEGA  | 0      | 
      | node6  | VEGA  | 0      | 
      | node7  | VEGA  | 0      | 
      | node8  | VEGA  | 0      | 
      | node10 | VEGA  | 0      | 
      | node11 | VEGA  | 0      | 
      | node12 | VEGA  | 0      | 
      | node13 | VEGA  | 0      | 