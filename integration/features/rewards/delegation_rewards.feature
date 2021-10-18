Feature: Staking & Delegation 

  Background:
    Given the following network parameters are set:
      | name                                              |  value                   |
      | reward.asset                                      |  VEGA                    |
      | validators.epoch.length                           |  10s                     |
      | validators.delegation.minAmount                   |  10                      |
      | reward.staking.delegation.payoutDelay             |  0s                      |
      | reward.staking.delegation.delegatorShare          |  0.883                   |
      | reward.staking.delegation.minimumValidatorStake   |  100                     |
      | reward.staking.delegation.payoutFraction          |  0.5                     |
      | reward.staking.delegation.maxPayoutPerParticipant | 100000                   |
      | reward.staking.delegation.competitionLevel        |  1.1                     |
      | reward.staking.delegation.maxPayoutPerEpoch       |  50000                   |
      | reward.staking.delegation.minValidators           |  5                       |


    Given time is updated to "2021-08-26T00:00:00Z"
    Given the average block duration is "2"

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

    #set up the self delegation of the validators
    Then the parties submit the following delegations:
      | party  | node id  | amount |
      | node1  |  node1   | 10000  | 
      | node2  |  node2   | 10000  |       
      | node3  |  node3   | 10000  | 
      | node4  |  node4   | 10000  | 
      | node5  |  node5   | 10000  | 
      | node6  |  node6   | 10000  | 
      | node7  |  node7   | 10000  | 
      | node8  |  node8   | 10000  | 
      | node9  |  node9   | 10000  | 
      | node10 |  node10  | 10000  | 
      | node11 |  node11  | 10000  | 
      | node12 |  node12  | 10000  | 
      | node13 |  node13  | 10000  | 

    And the parties deposit on staking account the following amount:
      | party  | asset  | amount |
      | party1 | VEGA   | 10000  |  

    Then the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  100   | 
    | party1 |  node2   |  200   |       
    | party1 |  node3   |  300   |     

    #complete the first epoch for the self delegation to take effect
    Then the network moves ahead "7" blocks

   Scenario: Parties get rewarded for a full epoch of having delegated stake
    Desciption: Parties have had their tokens delegated to nodes for a full epoch and get rewarded for the full epoch. 

    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 

    #advance to the end of the epoch / start next epoch
    Then the network moves ahead "7" blocks

    #verify validator score 
    Then the validators should have the following val scores for epoch 1:
    | node id | validator score  | normalised score |
    |  node1  |      0.07734     |     0.07734      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07657     |     0.07657      | 

  Scenario: No funds in reward account however validator scores get published
    Desciption: Parties have had their tokens delegated to nodes for a full epoch but the reward account balance is 0

    #advance to the end of the epoch / start next epoch
    When the network moves ahead "7" blocks

    #verify validator score 
    Then the validators should have the following val scores for epoch 1:
    | node id | validator score  | normalised score |
    |  node1  |      0.07734     |     0.07734      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07657     |     0.07657      | 

  Scenario: Parties get rewarded for a full epoch of having delegated stake - the reward amount is capped 
    Desciption: Parties have had their tokens delegated to nodes for a full epoch and get rewarded for the full epoch. 
    
    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 

    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 120000 |   

    #the available amount for the epoch is 60k but it is capped to 50 by the max.  

    #advance to the end of the epoch
    Then the network moves ahead "7" blocks

    #verify validator score 
    Then the validators should have the following val scores for epoch 1:
    | node id | validator score  | normalised score |
    |  node1  |      0.07734     |     0.07734      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07657     |     0.07657      | 

    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 200 from party2 
    #node3 has 10k self delegation + 300 from party3 
    #all other nodes have 10k self delegation 
    #party1 gets 0.07734 * 50000 * 0.883 * 100/10100 + 0.07810 * 50000 * 0.883 * 200/10200 + 0.07887 * 50000 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07734 * 50000
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 50000
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 50000
    #node4 - node13 gets: 0.07657 * 50000
    And the parties receive the following reward for epoch 1:
    | party  | asset | amount |
    | party1 | VEGA  |  201   | 
    | node1  | VEGA  |  3832  | 
    | node2  | VEGA  |  3837  | 
    | node3  | VEGA  |  3841  | 
    | node4  | VEGA  |  3828  | 
    | node5  | VEGA  |  3828  | 
    | node6  | VEGA  |  3828  | 
    | node8  | VEGA  |  3828  | 
    | node10 | VEGA  |  3828  | 
    | node11 | VEGA  |  3828  | 
    | node12 | VEGA  |  3828  | 
    | node13 | VEGA  |  3828  | 

  Scenario: Parties request to undelegate at the end of the epoch. They get fully rewarded for the current epoch and not get rewarded in the following epoch for the undelegated stake
    Desciption: Parties have had their tokens delegated to nodes for a full epoch and get rewarded for the full epoch. During the epoch however they request to undelegate at the end of the epoch part of their stake. On the following epoch they are not rewarded for the undelegated stake. 

    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 

    Then the parties submit the following undelegations:
    | party  | node id  | amount | when         |
    | party1 |  node2   |  150   | end of epoch |      
    | party1 |  node3   |  300   | end of epoch |


    #advance to the end of the epoch
    When the network moves ahead "7" blocks

    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 200 from party2 
    #node3 has 10k self delegation + 300 from party3 
    #all other nodes have 10k self delegation 
    #party1 gets 0.07734 * 50000 * 0.883 * 100/10100 + 0.07810 * 50000 * 0.883 * 200/10200 + 0.07887 * 50000 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07734 * 50000
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 50000
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 50000
    #node4 - node13 gets: 0.07657 * 50000
    And the parties receive the following reward for epoch 1:
    | party  | asset | amount |
    | party1 | VEGA  |  201   | 
    | node1  | VEGA  |  3832  | 
    | node2  | VEGA  |  3837  | 
    | node3  | VEGA  |  3841  | 
    | node4  | VEGA  |  3828  | 
    | node5  | VEGA  |  3828  | 
    | node6  | VEGA  |  3828  | 
    | node7  | VEGA  |  3828  | 
    | node8  | VEGA  |  3828  | 
    | node9  | VEGA  |  3828  | 
    | node10 | VEGA  |  3828  | 
    | node11 | VEGA  |  3828  | 
    | node12 | VEGA  |  3828  | 
    | node13 | VEGA  |  3828  | 

    #advance to the beginning and end of the following epoch 
    When the network moves ahead "7" blocks

    #verify validator score 
    Then the validators should have the following val scores for epoch 2:
    | node id | validator score  | normalised score |
    |  node1  |      0.07760     |     0.07760      |    
    |  node2  |      0.07722     |     0.07722      |
    |  node3  |      0.07683     |     0.07683      | 

    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 50 from party2 
    #all other nodes have 10k self delegation 
    #party1 gets 0.0776 * 25004 * 0.883 * 100/10100 + 0.07722 * 25004 * 0.883 * 50/10050
    #node1 gets: (1 - 0.883 * 100/10100) * 0.0776 * 25004
    #node2 gets: (1 - 0.883 * 50/10050) * 0.07722 * 25004
    #node4 - node13 gets: 0.07683 * 25004
    And the parties receive the following reward for epoch 2:
    | party  | asset | amount |
    | party1 | VEGA  |  24    | 
    | node1  | VEGA  |  1923  | 
    | node2  | VEGA  |  1922  | 
    | node3  | VEGA  |  1921  | 
    | node4  | VEGA  |  1921  | 
    | node5  | VEGA  |  1921  | 
    | node6  | VEGA  |  1921  | 
    | node7  | VEGA  |  1921  | 
    | node8  | VEGA  |  1921  | 
    | node9  | VEGA  |  1921  | 
    | node10 | VEGA  |  1921  | 
    | node11 | VEGA  |  1921  | 
    | node12 | VEGA  |  1921  | 
    | node13 | VEGA  |  1921  | 

  Scenario: Parties request to undelegate now during the epoch. They only get rewarded for the current epoch for the fraction that remained for the whole duration 
    Desciption: Parties have had their tokens delegated to nodes for a full epoch and get rewarded for the full epoch. During the epoch however they request to undelegate at the end of the epoch part of their stake. On the following epoch they are not rewarded for the undelegated stake. 

    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
      
    Then the parties submit the following undelegations:
    | party  | node id  | amount | when |
    | party1 |  node2   |  150   | now  |      
    | party1 |  node3   |  300   | now  |

    #advance to the end of the epoch
    When the network moves ahead "7" blocks

    #verify validator score 
    Then the validators should have the following val scores for epoch 1:
    | node id | validator score  | normalised score |
    |  node1  |      0.07760     |     0.07760      |    
    |  node2  |      0.07722     |     0.07722      |
    |  node3  |      0.07683     |     0.07683      | 

    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 50 from party2 
    #all other nodes have 10k self delegation 

    #party1 gets 0.07760 * 50000 * 0.883 * 100/10100 + 0.07722 * 50000 * 0.883 * 50/10050 
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07760 * 50000
    #node2 gets: (1 - 0.883 * 50/10050) * 0.07722 * 50000
    #node3 - node13 gets: 0.07683 * 50000
    And the parties receive the following reward for epoch 1:
    | party  | asset | amount |
    | party1 | VEGA  |  49    | 
    | node1  | VEGA  |  3846  | 
    | node2  | VEGA  |  3843  | 
    | node3  | VEGA  |  3841  | 
    | node4  | VEGA  |  3841  | 
    | node5  | VEGA  |  3841  | 
    | node6  | VEGA  |  3841  | 
    | node7  | VEGA  |  3841  | 
    | node8  | VEGA  |  3841  | 
    | node9  | VEGA  |  3841  | 
    | node10 | VEGA  |  3841  | 
    | node11 | VEGA  |  3841  | 
    | node12 | VEGA  |  3841  | 
    | node13 | VEGA  |  3841  | 

  Scenario: Parties withdraw from their staking account during an epoch once having active delegations - they should not get rewarded for those uncovered delegations 
    Desciption: Parties have active delegations on epoch 1 and withdraw stake from the staking account. They should only get rewarded for any delegation that still has cover 

    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
      
    #party1 has a balance of 10k tokens in their staking account and an active delegation in this epoch of 600. By withdrawing 9850, 450 of their delegation needs to be revoked and they should only get rewarded for the 150 tokens
    #NB: the undelegation is done proportionally to the stake they have in each node, so for example party1 has 100, 200, 300 in nodes 1-3 respectively so 
    #after undelegation they will have 25, 50, 75 in nodes 1-3 respectively
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   |  9850  |

    #advance to the end of the epoch
    When the network moves ahead "7" blocks

    #verify validator score 
    Then the validators should have the following val scores for epoch 1:
    | node id | validator score  | normalised score |
    |  node1  |      0.07703     |     0.07703      |    
    |  node2  |      0.07722     |     0.07722      |
    |  node3  |      0.07741     |     0.07741      | 
    |  node4  |      0.07683     |     0.07683      | 

    #node1 has 10k self delegation + 25 from party1
    #node2 has 10k self delegation + 50 from party1
    #node3 has 10k self delegation + 75 from party1
    #all other nodes have 10k self delegation 

    #party1 gets 0.07703 * 50000 * 0.883 * 25/10025 + 0.07722 * 50000 * 0.883 * 50/10050 + 0.07741 * 50000 * 0.883 * 75/10075
    #node1 gets: (1 - 0.883 * 25/10025) * 0.07703 * 50000
    #node2 gets: (1 - 0.883 * 50/10050) * 0.07722 * 50000
    #node3 gets: (1 - 0.883 * 75/10075) * 0.07741 * 50000
    #node4 - node13 get: 0.07683 * 50000

    And the parties receive the following reward for epoch 1:
    | party  | asset | amount |
    | party1 | VEGA  |  49    | 
    | node1  | VEGA  |  3842  | 
    | node2  | VEGA  |  3843  | 
    | node3  | VEGA  |  3845  | 
    | node4  | VEGA  |  3841  | 
    | node5  | VEGA  |  3841  | 
    | node6  | VEGA  |  3841  | 
    | node7  | VEGA  |  3841  | 
    | node8  | VEGA  |  3841  | 
    | node9  | VEGA  |  3841  | 
    | node10 | VEGA  |  3841  | 
    | node11 | VEGA  |  3841  | 
    | node12 | VEGA  |  3841  | 
    | node13 | VEGA  |  3841  | 

    Then "party1" should have general account balance of "49" for asset "VEGA"
    Then "node1" should have general account balance of "3842" for asset "VEGA"
  
  Scenario: Party has delegation unfunded for majority of the epoch (except for begining and end) - should get no rewards.

    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
      
    Given the parties withdraw from staking account the following amount:  
      | party  | asset | amount |
      | party1 | VEGA  |  9999  |

    When the network moves ahead "6" blocks
    
    And the parties deposit on staking account the following amount:
      | party  | asset | amount |
      | party1 | VEGA  | 9999   |  

    #advance to the end of the epoch
    When the network moves ahead "1" blocks

    And the parties receive the following reward for epoch 1:
      | party  | asset | amount |
      | party1 | VEGA  |  0     | 

   Scenario: A party changes delegation from one validator to another in the same epoch
   Description: A party can change delegation from one validator to another      

    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 

    When the network moves ahead "7" blocks

    #now request to undelegate from node2 and node3 
    And the parties submit the following undelegations:
    | party  | node id  | amount |      when     |
    | party1 |  node2   |  180   |  end of epoch |     
    | party1 |  node3   |  300   |  end of epoch | 
    Then the parties submit the following delegations:
    | party  | node id  | amount | 
    | party1 |  node1   |  180   | 
    | party1 |  node1   |  190   |  

    #advance to the end of the epoch for the delegation to become effective
    Then the network moves ahead "7" blocks    

     #verify validator score 
    Then the validators should have the following val scores for epoch 2:
    | node id | validator score  | normalised score |
    |  node1  |      0.07734     |     0.07734      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07657     |     0.07657      |

    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 200 from party1
    #node3 has 10k self delegation + 300 from party1
    #all other nodes have 10k self delegation
    #party1 gets 0.07734  * 25004 * 0.883 * 100/10100 + 0.07810 * 25004 * 0.883 * 200/10200 + 0.07887 * 25004 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07734 * 25004
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 25004
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 25004
    #node4 - node13 gets: 0.07657 * 25004

    And the parties receive the following reward for epoch 2:
    | party  | asset | amount |
    | party1 | VEGA  |  99    | 
    | node1  | VEGA  |  1916  | 
    | node2  | VEGA  |  1919  | 
    | node3  | VEGA  |  1921  | 
    | node4  | VEGA  |  1914  | 
    | node5  | VEGA  |  1914  | 
    | node6  | VEGA  |  1914  | 
    | node7  | VEGA  |  1914  | 
    | node8  | VEGA  |  1914  | 
    | node9  | VEGA  |  1914  | 
    | node10 | VEGA  |  1914  | 
    | node11 | VEGA  |  1914  | 
    | node12 | VEGA  |  1914  | 
    | node13 | VEGA  |  1914  | 
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   | 470    | 
    | party1 |  node2   | 20     |       
    | party1 |  node3   |  0     | 

     #advance to the beginning and end of the following epoch 
    Then the network moves ahead "7" blocks

    #verify validator score 
    Then the validators should have the following val scores for epoch 3:
    | node id | validator score  | normalised score |
    |  node1  |      0.08024     |     0.08024      |    
    |  node2  |      0.07679     |     0.07679      |
    |  node3  |      0.07663     |     0.07663      | 

    #node1 has 10k self delegation + 470 from party1
    #node2 has 10k self delegation + 20 from party1
    #node3 has 10k self delegation + 0 from party1
    #all other nodes have 10k self delegation
    #party1 gets 0.08024 * 12506 * 0.883 * 470/10470 + 0.07679 * 12506 * 0.883 * 20/10020
    #node1 gets: (1 - 0.883 * 470/10470)  * 0.08024 * 12506
    #node2 gets: (1 - 0.883 * 20/10020) * 0.07679 * 12506
    #node3 gets: (1 - 0.883 * 0/1000) * 0.07663 * 12506
    #node4 - node13 gets: 0.07663 * 12506

    And the parties receive the following reward for epoch 3:
    | party  | asset | amount |
    | party1 | VEGA  |  40    | 
    | node1  | VEGA  |  963   | 
    | node2  | VEGA  |  958   | 
    | node3  | VEGA  |  958   | 
    | node4  | VEGA  |  958   | 
    | node5  | VEGA  |  958   | 
    | node6  | VEGA  |  958   | 
    | node7  | VEGA  |  958   | 
    | node8  | VEGA  |  958   | 
    | node9  | VEGA  |  958   | 
    | node10 | VEGA  |  958   | 
    | node11 | VEGA  |  958   | 
    | node12 | VEGA  |  958   | 
    | node13 | VEGA  |  958   | 
  
  Scenario: A party can request delegate and undelegate from the same node at the same epoch such that the request can balance each other without affecting the actual delegate balance
    Description: party requests to delegate to node1 at the end of the epoch and regrets it and undelegate the whole amount to delegate it to another node

    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 

    And the parties submit the following undelegations:
    | party  | node id  | amount |    when      |
    | party1 |  node1   |  100   | end of epoch |
    And the parties submit the following delegations:
    | party  | node id  |  amount | 
    | party1 |  node4   |   100   | 
    #advance to the end of the epoch
    Then the network moves ahead "7" blocks
    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 200 from party1 
    #node3 has 10k self delegation + 300 from party1 
    #all other nodes have 10k self delegation 
    #party1 gets 0.07734 * 50000 * 0.883 * 100/10100 + 0.07810 * 50000 * 0.883 * 200/10200 + 0.07887 * 50000 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07734 * 50000
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 50000
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 50000
    #node4 - node13 gets: 0.07657 * 50000
    And the parties receive the following reward for epoch 1:
    | party  | asset | amount |
    | party1 | VEGA  |  201   | 
    | node1  | VEGA  |  3832  | 
    | node2  | VEGA  |  3837  | 
    | node3  | VEGA  |  3841  | 
    | node4  | VEGA  |  3828  | 
    | node5  | VEGA  |  3828  | 
    | node6  | VEGA  |  3828  | 
    | node7  | VEGA  |  3828  | 
    | node8  | VEGA  |  3828  | 
    | node9  | VEGA  |  3828  | 
    | node10 | VEGA  |  3828  | 
    | node11 | VEGA  |  3828  | 
    | node12 | VEGA  |  3828  | 
    | node13 | VEGA  |  3828  | 
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 0      | 
    | party1 |  node4   | 100    | 
    #advance to the beginning and end of the following epoch 
    Then the network moves ahead "7" blocks
    #verify validator score 
    Then the validators should have the following val scores for epoch 2:
    | node id | validator score  | normalised score |
    |  node1  |      0.07657     |     0.07657      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07734     |     0.07734      | 
    |  node5  |      0.07657     |     0.07657      | 
    #node1 has 10k self delegation
    #node2 has 10k self delegation + 200 from party1
    #node3 has 10k self delegation + 300 from party1 
    #node4 has 10k self delegation + 100 from party1 
    #all other nodes have 10k self delegation
    #party1 gets 0.07657  * 25004 * 0.883 * 0/10100 + 0.07810 * 25004 * 0.883 * 200/10200 + 0.07887 * 25004 * 0.883 * 300/10300 + 0.07734 * 25004 * 0.883 * 100/10100
    #node1 gets: (1 - 0.883 * 0/10100) * 0.07657 * 25004
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 25004
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 25004
    #node4 gets: (1 - 0.883 * 100/10100) * 0.07734 * 25004
    #node5 - node13 gets: 0.07657 * 25004
    And the parties receive the following reward for epoch 2:
    | party  | asset | amount |
    | party1 | VEGA  |  99    | 
    | node1  | VEGA  |  1914  | 
    | node2  | VEGA  |  1919  | 
    | node3  | VEGA  |  1921  | 
    | node4  | VEGA  |  1916  | 
    | node5  | VEGA  |  1914  | 
    | node6  | VEGA  |  1914  | 
    | node7  | VEGA  |  1914  | 
    | node8  | VEGA  |  1914  | 
    | node9  | VEGA  |  1914  | 
    | node10 | VEGA  |  1914  | 
    | node11 | VEGA  |  1914  | 
    | node12 | VEGA  |  1914  | 
    | node13 | VEGA  |  1914  | 
  
  Scenario: A party has active delegations and submits an undelegate request followed by a delegation request that covers only part of the undelegation such that the undelegation still takes place
    Description: A party delegated tokens to node1 at previous epoch such that the delegations is now active and is requesting to undelegate some of the tokens at the end of the current epoch. Then regret some of it and submit a delegation request that undoes some of the undelegation but still some of it remains.

    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
      
    #advance to the end of the epoch
    Then the network moves ahead "7" blocks
    Then the parties submit the following undelegations:
    | party  | node id  | amount |    when      |
    | party1 |  node1   |  100   | end of epoch |
    And the parties submit the following delegations:
    | party  | node id  |  amount | 
    | party1 |  node1   |    50   |
    When the network moves ahead "7" blocks
    #verify validator score 
    Then the validators should have the following val scores for epoch 2:
    | node id | validator score  | normalised score |
    |  node1  |      0.07734     |     0.07734      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07657     |     0.07657      |
    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 200 from party1
    #node3 has 10k self delegation + 300 from party1
    #all other nodes have 10k self delegation
    #party1 gets 0.07734  * 25004 * 0.883 * 100/10100 + 0.07810 * 25004 * 0.883 * 200/10200 + 0.07887 * 25004 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07734 * 25004
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 25004
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 25004
    #node4 - node13 gets: 0.07657 * 25004
    And the parties receive the following reward for epoch 2:
    | party  | asset | amount |
    | party1 | VEGA  |  99    | 
    | node1  | VEGA  |  1916  | 
    | node2  | VEGA  |  1919  | 
    | node3  | VEGA  |  1921  | 
    | node4  | VEGA  |  1914  | 
    | node5  | VEGA  |  1914  | 
    | node6  | VEGA  |  1914  | 
    | node7  | VEGA  |  1914  | 
    | node8  | VEGA  |  1914  | 
    | node9  | VEGA  |  1914  | 
    | node10 | VEGA  |  1914  | 
    | node11 | VEGA  |  1914  | 
    | node12 | VEGA  |  1914  | 
    | node13 | VEGA  |  1914  | 
    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |  50    | 
    #advance to the beginning and end of the following epoch 
    When the network moves ahead "7" blocks
    #verify validator score 
    Then the validators should have the following val scores for epoch 3:
    | node id | validator score  | normalised score |
    |  node1  |      0.07698     |     0.07698      |    
    |  node2  |      0.07813     |     0.07813      |
    |  node3  |      0.07890     |     0.07890      | 
    |  node4  |      0.07660     |     0.07660      |
    #node1 has 10k self delegation +  50 from party1
    #node2 has 10k self delegation + 200 from party1
    #node3 has 10k self delegation + 300 from party1
    #all other nodes have 10k self delegation
    #party1 gets 0.07698  * 12506 * 0.883 * 50/10050 + 0.07813 * 12506 * 0.883 * 200/10200 + 0.07890 * 12506 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 50/10050)  * 0.07698 * 12506
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07813 * 12506
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07890 * 12506
    #node4 - node13 gets: 0.07660 * 12506
    And the parties receive the following reward for epoch 3:
    | party  | asset | amount |
    | party1 | VEGA  |  45    | 
    | node1  | VEGA  |  958   | 
    | node2  | VEGA  |  960   | 
    | node3  | VEGA  |  961   | 
    | node4  | VEGA  |  958   | 
    | node5  | VEGA  |  958   | 
    | node6  | VEGA  |  958   | 
    | node7  | VEGA  |  958   | 
    | node8  | VEGA  |  958   | 
    | node9  | VEGA  |  958   | 
    | node10 | VEGA  |  958   | 
    | node11 | VEGA  |  958   | 
    | node12 | VEGA  |  958   | 
    | node13 | VEGA  |  958   | 
  
  Scenario: Parties get rewarded for a full epoch of having delegated stake - the reward amount is capped per participant
   Description: Parties have had their tokens delegated to nodes for a full epoch and get rewarded for the full epoch and the reward amount per participant is capped
  
    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
      
    Given the following network parameters are set:
      | name                                              | value |
      | reward.staking.delegation.maxPayoutPerParticipant | 3000  |
    #the reward amount for each participant per epoch is capped to 3k by maxPayoutPerParticipant
    #advance to the end of the epoch
    When the network moves ahead "7" blocks
    #verify validator score 
    Then the validators should have the following val scores for epoch 1:
    | node id | validator score  | normalised score |
    |  node1  |      0.07734     |     0.07734      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07657     |     0.07657      | 
    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 200 from party2 
    #node3 has 10k self delegation + 300 from party3 
    #all other nodes have 10k self delegation 
    #party1 gets 0.07734 * 50000 * 0.883 * 100/10100 + 0.07810 * 50000 * 0.883 * 200/10200 + 0.07887 * 50000 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07734 * 50000
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 50000
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 50000
    #node4 - node13 gets: 0.07657 * 50000
    And the parties receive the following reward for epoch 1:
    | party  | asset | amount |
    | party1 | VEGA  |  201   | 
    | node1  | VEGA  |  3000  | 
    | node2  | VEGA  |  3000  | 
    | node3  | VEGA  |  3000  | 
    | node4  | VEGA  |  3000  | 
    | node5  | VEGA  |  3000  | 
    | node6  | VEGA  |  3000  | 
    | node7  | VEGA  |  3000  | 
    | node8  | VEGA  |  3000  | 
    | node9  | VEGA  |  3000  | 
    | node10 | VEGA  |  3000  | 
    | node11 | VEGA  |  3000  | 
    | node12 | VEGA  |  3000  | 
    | node13 | VEGA  |  3000  |

  Scenario: Topping up the reward account and confirming reward transfers are correctly refelected in parties account balances
    Description: Topping up the reward account and confirming reward transfers are correctly refelected in parties account balances when they get rewarded for a full epoch of having delegated stake

    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  |  50000 | 

    #advance to the end of the epoch
    When the network moves ahead "7" blocks

    #verify validator score 
    Then the validators should have the following val scores for epoch 1:
    | node id | validator score  | normalised score |
    |  node1  |      0.07734     |     0.07734      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07657     |     0.07657      | 
    
    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 200 from party2 
    #node3 has 10k self delegation + 300 from party3 
    #all other nodes have 10k self delegation 
    #party1 gets 0.07734 * 25000 * 0.883 * 100/10100 + 0.07810 * 25000 * 0.883 * 200/10200 + 0.07887 * 25000 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07734 * 25000
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 25000
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 25000
    #node4 - node13 gets: 0.07657 * 25000
    
    And the parties receive the following reward for epoch 1:
    | party  | asset | amount |
    | party1 | VEGA  |   99   | 
    | node1  | VEGA  |  1916  | 
    | node2  | VEGA  |  1918  | 
    | node3  | VEGA  |  1920  | 
    | node4  | VEGA  |  1914  | 
    | node5  | VEGA  |  1914  | 
    | node6  | VEGA  |  1914  | 
    | node7  | VEGA  |  1914  | 
    | node8  | VEGA  |  1914  | 
    | node9  | VEGA  |  1914  | 
    | node10 | VEGA  |  1914  | 
    | node11 | VEGA  |  1914  | 
    | node12 | VEGA  |  1914  | 
    | node13 | VEGA  |  1914  | 

    Then "party1" should have general account balance of "99" for asset "VEGA"
    Then "node1" should have general account balance of "1916" for asset "VEGA"
  
    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 74993  | 

    # The total amount now in rewards account is 100000-50000-24993(rewards payout)+74993 = 100000
    When the network moves ahead "7" blocks

    #verify validator score 
    Then the validators should have the following val scores for epoch 2:
    | node id | validator score  | normalised score |
    |  node1  |      0.07734     |     0.07734      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07657     |     0.07657      | 

    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 200 from party2 
    #node3 has 10k self delegation + 300 from party3 
    #all other nodes have 10k self delegation 
    #party1 gets 0.07734 * 50000 * 0.883 * 100/10100 + 0.07810 * 50000 * 0.883 * 200/10200 + 0.07887 * 50000 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07734 * 50000
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 50000
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 50000
    #node4 - node13 gets: 0.07657 * 50000
    
    And the parties receive the following reward for epoch 2:
    | party  | asset | amount |
    | party1 | VEGA  |  201   | 
    | node1  | VEGA  |  3832  | 
    | node2  | VEGA  |  3837  | 
    | node3  | VEGA  |  3841  | 
    | node4  | VEGA  |  3828  | 
    | node5  | VEGA  |  3828  | 
    | node6  | VEGA  |  3828  | 
    | node8  | VEGA  |  3828  | 
    | node10 | VEGA  |  3828  | 
    | node11 | VEGA  |  3828  | 
    | node12 | VEGA  |  3828  | 
    | node13 | VEGA  |  3828  | 

    Then "party1" should have general account balance of "300" for asset "VEGA"
    Then "node1" should have general account balance of "5748" for asset "VEGA"

  Scenario: Parties get the smallest reward amount of 1 when the reward pot is smallest
    Description:  Validators get the smallest reward amount of 1 and delegator earns nothing
    # Explanation - 1 vega is actually 1000000000000000000 so when reward account = 27 then that’s a very very very small fraction of a vega. Hence noone gets anything because the calculation is made in integers so anything that ends up being less than one is 0

    Given the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  |     28 | 

    #advance to the end of the epoch
    When the network moves ahead "7" blocks

    #verify validator score 
    Then the validators should have the following val scores for epoch 1:
    | node id | validator score  | normalised score |
    |  node1  |      0.07734     |     0.07734      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07657     |     0.07657      | 

    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 200 from party2 
    #node3 has 10k self delegation + 300 from party3 
    #all other nodes have 10k self delegation 
    #party1 gets 0.07734 * 28 * 0.883 * 100/10100 + 0.07810 * 28 * 0.883 * 200/10200 + 0.07887 * 28 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07734 * 28
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 28
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 28
    #node4 - node13 gets: 0.07657 * 28

    And the parties receive the following reward for epoch 1:
    | party  | asset | amount |
    | party1 | VEGA  | 0 | 
    | node1  | VEGA  | 1 | 
    | node2  | VEGA  | 1 | 
    | node3  | VEGA  | 1 | 
    | node4  | VEGA  | 1 |
    | node5  | VEGA  | 1 | 
    | node6  | VEGA  | 1 | 
    | node7  | VEGA  | 1 | 
    | node8  | VEGA  | 1 | 
    | node9  | VEGA  | 1 | 
    | node10 | VEGA  | 1 | 
    | node11 | VEGA  | 1 | 
    | node12 | VEGA  | 1 | 
    | node13 | VEGA  | 1 | 

    Then "node1" should have general account balance of "1" for asset "VEGA"

    When the network moves ahead "7" blocks

    And the parties receive the following reward for epoch 2:
    | party  | asset | amount |
    | party1 | VEGA  | 0 | 
    | node1  | VEGA  | 0 | 
    | node2  | VEGA  | 0 | 
    | node3  | VEGA  | 0 | 
    | node4  | VEGA  | 0 |
    | node5  | VEGA  | 0 | 
    | node6  | VEGA  | 0 | 
    | node7  | VEGA  | 0 | 
    | node8  | VEGA  | 0 | 
    | node9  | VEGA  | 0 | 
    | node10 | VEGA  | 0 | 
    | node11 | VEGA  | 0 | 
    | node12 | VEGA  | 0 | 
    | node13 | VEGA  | 0 | 

    When the network moves ahead "37542" blocks
    And the parties receive the following reward for epoch 3153602:
    | party  | asset | amount |
    | party1 | VEGA  | 0 | 
    | node1  | VEGA  | 0 | 
    | node2  | VEGA  | 0 | 
    | node3  | VEGA  | 0 | 
    | node4  | VEGA  | 0 |
    | node5  | VEGA  | 0 | 
    | node6  | VEGA  | 0 | 
    | node7  | VEGA  | 0 | 
    | node8  | VEGA  | 0 | 
    | node9  | VEGA  | 0 | 
    | node10 | VEGA  | 0 | 
    | node11 | VEGA  | 0 | 
    | node12 | VEGA  | 0 | 
    | node13 | VEGA  | 0 | 