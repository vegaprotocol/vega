Feature: Staking & Delegation - scenarios focusing on initial epoch

  Background:
    Given the following network parameters are set:
      | name                                              |  value  |
      | reward.asset                                      |  VEGA   |
      | validators.epoch.length                           |  24h    |
      | validators.delegation.minAmount                   |  10     |
      | reward.staking.delegation.delegatorShare          |  0.883  |
      | reward.staking.delegation.minimumValidatorStake   |  100    |
      | reward.staking.delegation.maxPayoutPerParticipant |  100000 |
      | reward.staking.delegation.competitionLevel        |  1.1    |
      | reward.staking.delegation.minValidators           |  5      |
      | reward.staking.delegation.optimalStakeMultiplier  |  5.0    |
  
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

    And the global reward account gets the following deposits:
    | asset | amount |
    | VEGA  | 50000  | 
    Then the network moves ahead "172804" blocks

    And the parties should have the following delegation balances for epoch 4:
      | party  | node id  | amount  |
      | node1  |  node1   |  11000  | 
      | node2  |  node2   |  12000  |       
      | node3  |  node3   |  13000  |  
      | node4  |  node4   |     99  |  
      | party1 |  node4   |  111000 |  
    
    # val score = max(0, (valStake - penaltyFlatAmt - penaltyDownAmt) / totalStake)
    # node1 score = (11000 - 0 - 0)/ 147099 = 0.07478 
    # node2 score = (12000 - 0 - 0)/ 147099 = 0.08158   
    # node3 score = (13000 - 0 - 0)/ 147099 = 0.08462 
    # node4 score = max(0, (111099 - 98652.16154 - 48,864.80769) / 147099) = 0
    Then the validators should have the following val scores for epoch 4:
      | node id | validator score  | normalised score |
      |  node1  |      0.07478     |     0.31032      |    
      |  node2  |      0.08158     |     0.33854      |
      |  node3  |      0.08462     |     0.35114      | 
      |  node4  |      0.00000     |     0.00000      | 
      |  node5  |      0.00000     |     0.00000      | 
      |  node6  |      0.00000     |     0.00000      | 
      |  node7  |      0.00000     |     0.00000      | 
      |  node8  |      0.00000     |     0.00000      | 
      |  node9  |      0.00000     |     0.00000      | 
      |  node10 |      0.00000     |     0.00000      | 
      |  node11 |      0.00000     |     0.00000      | 
      |  node12 |      0.00000     |     0.00000      | 
      |  node13 |      0.00000     |     0.00000      |

    # party1 is delegating to node 4 which has a 0 valScore so they get nothing 
    # node1 gets 0.31032 of the reward 
    # node2 gets 0.33854  of the reward 
    # node3 gets 0.35114 of the reward 
    And the parties receive the following reward for epoch 4:
      | party  | asset | amount  |
      | party1 | VEGA  | 0       | 
      | node1  | VEGA  | 15516   | 
      | node2  | VEGA  | 16926   | 
      | node3  | VEGA  | 17557   | 
      | node4  | VEGA  |     0   | 
      | node5  | VEGA  |     0   | 
      | node6  | VEGA  |     0   | 
      | node8  | VEGA  |     0   | 
      | node10 | VEGA  |     0   | 
      | node11 | VEGA  |     0   | 
      | node12 | VEGA  |     0   | 
      | node13 | VEGA  |     0   | 

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
      | VEGA  | 50000 | 
    
    #complete the initial epoch for delegation to take effect
    Then the network moves ahead "172804" blocks

    # optStake = 65000/(max(5, 13/1.1)) = 5500
    # val score = max(0, (valStake - penaltyFlatAmt - penaltyDownAmt) / totalStake)
    # node1 score = (11000 - 5500 - 0)/ 65000 = 0.08462 
    # node2 score = (12000 - 6500 - 0)/ 65000 = 0.08462   
    # node3 score = (13000 - 7500 - 0)/ 65000 = 0.08462
    # node4 score = (14000 - 8500 - 0)/ 65000 = 0.08462 
    # node5 score = (15000 - 9500 - 0)/ 65000 = 0.08462
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

    #each node gets 0.2 of the reward, only self delegation in place
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
      | VEGA  | 50000 | 
    
    #complete the initial epoch for delegation to take effect
    Then the network moves ahead "172804" blocks

    And the parties should have the following delegation balances for epoch 1:
      | party  | node id  |  amount |
      | node1  |  node1   |     100 | 
      | node2  |  node2   |     200 |       
      | node3  |  node3   |     300 |  
      | node4  |  node4   |     400 | 
      | node5  |  node5   |     500 | 
      | party1 |  node1   |      10 |  
      | party2 |  node1   |      50 |  
      | party3 |  node1   |     200 |  

    # totalStale = 1760
    # optStake = 1760/(max(5, 13/1.1)) = 148.9230769231
    # val score = max(0, (valStake - penaltyFlatAmt - penaltyDownAmt) / totalStake)
    # node1 score = (360 - 211.0769230769 - 0)/ 1760 = 0.08462 
    # node2 score = (200 - 51.0769230769 - 0)/ 1760 = 0.08462   
    # node3 score = (300 - 151.0769230769 - 0)/ 1760 = 0.08462
    # node4 score = (400 - 251.0769230769 - 0)/ 1760 = 0.08462 
    # node5 score = (500 - 351.0769230769 - 0)/ 1760 = 0.08462
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

    # node 1 and its delegators receive 10k
    # delegators to node1 receive 0.883 * 260 / 360 * 10000 = 6377
    # party1 gets 10/260 * 6377 = 245
    # party1 gets 50/260 * 6377 = 1126
    # party1 gets 200/260 * 6377 = 4905 
    # node1 takes the rest = 3622 
    # node 2, 3, 4, 5 receive 0.2 of the reward each - only self delegation
    And the parties receive the following reward for epoch 1:
      | party  | asset | amount |
      | party1 | VEGA  | 245    | 
      | party2 | VEGA  | 1226   | 
      | party3 | VEGA  | 4905   | 
      | node1  | VEGA  | 3622   | 
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
      | VEGA  | 50000 | 
    
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
      | party3 |  node1   |  111000 | 
      | party3 |  node2   |  111000 | 
      | party3 |  node7   |  111000 | 
      | party4 |  node1   |  222000 | 
      | party4 |  node7   |  222000 | 
      | party5 |  node2   |  555000 | 
  
    # totalStale = 1,746,000
    # optStake = 1,746,000/(max(5, 13/1.1)) = 147,738.4615384615
    # val score = max(0, (valStake - penaltyFlatAmt - penaltyDownAmt) / totalStake)
    # node1 score = (455000 - 307,261.5384615385 - 0)/ 1746000 = 0.08462 
    # node2 score = (789000 - 641,261.5384615385 - 50,307.6923076925)/ 1746000 = 0.05580   
    # node3 score = (13000 - 0 - 0)/ 1746000 = 0.00745 
    # node4 score = (14000 - 0 - 0)/ 1746000 = 0.00802 
    # node5 score = (15000 - 0 - 0)/ 1746000 = 0.00859
    # node6 score = (16000 - 0 - 0)/ 1746000 = 0.00916
    # node7 score = (444000 - 296,261.5384615385 - 0)/ 1746000 = 0.08462
    Then the validators should have the following val scores for epoch 1:
      | node id | validator score  | normalised score |
      |  node1  |      0.08462     |     0.32765      |    
      |  node2  |      0.05580     |     0.21608      |
      |  node3  |      0.00745     |     0.02883      | 
      |  node4  |      0.00802     |     0.03105      | 
      |  node5  |      0.00859     |     0.03327      | 
      |  node6  |      0.00916     |     0.03548      | 
      |  node7  |      0.08462     |     0.32765      | 
      |  node8  |      0.00000     |     0.00000      | 
      |  node9  |      0.00000     |     0.00000      | 
      |  node10 |      0.00000     |     0.00000      | 
      |  node11 |      0.00000     |     0.00000      | 
      |  node12 |      0.00000     |     0.00000      | 
      |  node13 |      0.00000     |     0.00000      | 
  
    And the parties receive the following reward for epoch 1:
      | party  | asset | amount |
      | party1 | VEGA  | 3528   | 
      | party2 | VEGA  | 4958   | 
      | party3 | VEGA  | 8486   | 
      | node1  | VEGA  | 2266   | 
      | node2  | VEGA  | 1409   | 
      | node3  | VEGA  | 1441   |  
      | node4  | VEGA  | 1552   | 
      | node5  | VEGA  | 1663   | 
      | node6  | VEGA  | 1774   | 
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
        | VEGA  | 50000 | 
      
      #complete the initial epoch for delegation to take effect
      Then the network moves ahead "172804" blocks

      And the parties should have the following delegation balances for epoch 1:
        | party  | node id  | amount  |
        | node1  |  node1   |  11000  | 
        | node2  |  node2   |     20  |       
        | node3  |  node3   |     30  | 
        | node4  |  node4   |  14000  | 
        | node5  |  node5   |  15000  | 
        | node6  |  node6   |  16000  |  
        | node8  |  node8   |    110  |       
        | node2  |  node7   |    180  |       
        | node3  |  node7   |   3000  | 
        | party1 |  node1   |  111000 | 
        | party2 |  node2   |  222000 | 

    # totalStale = 392,340
    # optStake = 392,340/(max(5, 13/1.1)) = 33,198
    # val score = max(0, (valStake - penaltyFlatAmt - penaltyDownAmt) / totalStake)
    # node1 score = (122000 - 88802 - 0)/ 392340 = 0.08462 
    # node2 score = max(0, (222020 - 188,822 - 56,030)/ 392340) = 0
    # node3 score = (30 - 0 - 0)/ 392340 = 0.00008
    # node4 score = (14000 - 0 - 0)/ 392340 = 0.03568 
    # node5 score = (15000 - 0 - 0)/ 392340 = 0.03823
    # node6 score = (16000 - 0 - 0)/ 392340 = 0.04078
    # node7 score = (3180 - 0 - 0)/ 392340 = 0.00811 
    # node8 score = (110 - 0 - 0)/ 392340 = 0.00028
      Then the validators should have the following val scores for epoch 1:
        | node id | validator score  | normalised score |
        |  node1  |      0.08462     |     0.40725      |    
        |  node2  |      0.00000     |     0.00000      |
        |  node3  |      0.00008     |     0.00037      | 
        |  node4  |      0.03568     |     0.17174      | 
        |  node5  |      0.03823     |     0.18401      | 
        |  node6  |      0.04078     |     0.19628      | 
        |  node7  |      0.00811     |     0.03901      | 
        |  node8  |      0.00028     |     0.00135      | 
        |  node9  |      0.00000     |     0.00000      | 
        |  node10 |      0.00000     |     0.00000      | 
        |  node11 |      0.00000     |     0.00000      | 
        |  node12 |      0.00000     |     0.00000      | 
        |  node13 |      0.00000     |     0.00000      |

      And the parties receive the following reward for epoch 1:
        | party  | asset | amount |
        | party1 | VEGA  | 16358  | 
        | party2 | VEGA  | 0      | 
        | node1  | VEGA  | 4003   | 
        | node2  | VEGA  | 97     | 
        | node3  | VEGA  | 1624   |  
        | node4  | VEGA  | 8587   | 
        | node5  | VEGA  | 9200   | 
        | node6  | VEGA  | 9813   | 
        | node7  | VEGA  | 0      | 
        | node8  | VEGA  | 67     | 
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
      | party  | node id  | amount  |
      | node1  |  node1   |  11000  | 
      | node2  |  node2   |     20  |       
      | node3  |  node3   |     30  | 
      | node4  |  node4   |  14000  | 
      | node5  |  node5   |  15000  | 
      | node6  |  node6   |  16000  | 
      | node8  |  node8   |     50  |       
      | node2  |  node7   |    180  |       
      | node3  |  node7   |    100  | 
      | party1 |  node1   |  111000 | 
      | party2 |  node2   |  222000 | 

    Then the network moves ahead "86401" blocks

    And the parties should have the following delegation balances for epoch 2:
      | party  | node id  | amount  |
      | node1  |  node1   |  11000  | 
      | node2  |  node2   |      2  |       
      | node3  |  node3   |     30  | 
      | node4  |  node4   |  14000  | 
      | node5  |  node5   |  15000  | 
      | node6  |  node6   |  16000  | 
      | node8  |  node8   |     50  |       
      | node2  |  node7   |     18  |       
      | node3  |  node7   |    100  | 
      | party1 |  node1   |  111000 | 
      | party2 |  node2   |  222000 | 

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
        | VEGA  | 50000 | 
      
      #complete the initial epoch for delegation to take effect
      Then the network moves ahead "172802" blocks

      And the parties should have the following delegation balances for epoch 1:
        | party  | node id  | amount  |
        | node1  |  node1   |  100000 | 
        | node2  |  node2   |  12000  |       
        | node3  |  node3   |  13000  | 
        | node4  |  node4   |  14000  | 
        | node5  |  node5   |  15000  | 
        | node6  |  node6   |  16000  | 
        | party1 |  node1   |  111000 | 
        | party2 |  node2   |  222000 | 

      # totalStale = 503000
      # optStake = 503000/(max(5, 13/1.1)) = 42,561.5384615385
      # val score = max(0, (valStake - penaltyFlatAmt - penaltyDownAmt) / totalStake)
      # node1 score = (211000 - 168,438.4615384615 - 0)/ 503000 = 0.08462 
      # node2 score = (234000 - 191,438.4615384615 - 21,192.3076923075)/ 503000 = 0.04248
      # node3 score = (13000 - 0 - 0)/ 503000 = 0.02584  
      # node4 score = (14000 - 0 - 0)/ 503000 = 0.02783 
      # node5 score = (15000 - 0 - 0)/ 503000 = 0.02982
      # node6 score = (16000 - 0 - 0)/ 503000 = 0.03181
      Then the validators should have the following val scores for epoch 1:
        | node id | validator score  | normalised score |
        |  node1  |      0.08462     |     0.34906      |    
        |  node2  |      0.04248     |     0.17526      |
        |  node3  |      0.02584     |     0.10662      | 
        |  node4  |      0.02783     |     0.11482      | 
        |  node5  |      0.02982     |     0.12302      | 
        |  node6  |      0.03181     |     0.13122      | 
        |  node7  |      0.00000     |     0.00000      | 
        |  node8  |      0.00000     |     0.00000      | 
        |  node9  |      0.00000     |     0.00000      | 
        |  node10 |      0.00000     |     0.00000      | 
        |  node11 |      0.00000     |     0.00000      | 
        |  node12 |      0.00000     |     0.00000      | 
        |  node13 |      0.00000     |     0.00000      | 

      And the parties receive the following reward for epoch 1:
        | party  | asset | amount |
        | party1 | VEGA  | 8107   | 
        | party2 | VEGA  | 7340   | 
        | node1  | VEGA  | 9345   | 
        | node2  | VEGA  | 1422   | 
        | node3  | VEGA  | 5330   |  
        | node4  | VEGA  | 5740   | 
        | node5  | VEGA  | 6151   | 
        | node6  | VEGA  | 6561   | 
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
        | node1  |  node1   |   100  | 
        | node2  |  node2   |   200  |       
        | node3  |  node3   |   300  | 
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