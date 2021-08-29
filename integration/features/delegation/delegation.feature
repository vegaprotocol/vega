Feature: Staking & Delegation 

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | governance.vote.asset                         | VEGA  |
      | validators.epoch.length                       | 10s   |
      | validators.delegation.minAmount               | 10    |
      | reward.staking.delegation.payoutDelay         | 0s    |
      
    Given time is updated to "2021-08-26T00:00:00Z"

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

    And the parties deposit on asset's general account the following amount:
      | party  | asset  | amount |
      | party1 | VEGA   |  10000 |
      | party2 | VEGA   |  20000 |
      | party3 | VEGA   |  30000 |

    #complete the first epoch for the self delegation to take effect
    Then time is updated to "2021-08-26T00:00:10Z"
    Then time is updated to "2021-08-26T00:00:11Z"

  Scenario: A party can delegate to a validator and undelegate at the end of an epoch
    Desciption: A party with a balance in the staking account can delegate to a validator

    The parties deposit on staging account the following amount:  
      | party  | asset  | amount |
      | party1 | VEGA   | 10000  |

    And the parties should have the following staking account balances:
      | party  | asset  | amount |
      | party1 | VEGA   | 10000  |    

    Then the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  100   | 
    | party1 |  node2   |  200   |       
    | party1 |  node3   |  300   |   
    
    #we are now in epoch 1 so the delegation balance for epoch 1 should not include the delegation but the hypothetical balance for epoch 2 should 
    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | party1 |  node1   |   0    | 
    | party1 |  node2   |   0    |       
    | party1 |  node3   |   0    |        

    And the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 100   | 
    | party1 |  node2   | 200   |       
    | party1 |  node3   | 300   |        

    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 100    | 
    | party1 |  node2   | 200    |       
    | party1 |  node3   | 300    |   

    When time is updated to "2021-08-26T00:00:22Z"  
    Then the parties submit the following undelegations:
    | party  | node id  | amount |
    | party1 |  node1   |  100   | 
    | party1 |  node2   |  200   |       
    | party1 |  node3   |  300   |   

    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 100    | 
    | party1 |  node2   | 200    |       
    | party1 |  node3   | 300    |   

    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |   0    | 
    | party1 |  node2   |   0    |       
    | party1 |  node3   |   0    |   

    #advance to teh end of epoch for the undelegation to take place 
    When time is updated to "2021-08-26T00:00:32Z"  
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |   0    | 
    | party1 |  node2   |   0    |       
    | party1 |  node3   |   0    |   

  Scenario: A party cannot delegate less than minimum delegateable stake  
    Desciption: A party attempts to delegate less than minimum delegateable stake from its staking account to a validator minimum delegateable stake
      
    The parties deposit on staging account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 10000  |  

    Then the parties submit the following delegations:
    | party  | node id  | amount | reference | error                                                                             |
    | party1 |  node1   |    1   |      a    | delegation amount is lower than the minimum amount for delegation for a validator |
    | party1 |  node2   |    2   |      b    | delegation amount is lower than the minimum amount for delegation for a validator |    
    | party1 |  node3   |    3   |      c    | delegation amount is lower than the minimum amount for delegation for a validator |

    #we are now in epoch 1 so the delegation balance for epoch 1 should not include the delegation but the hypothetical balance for epoch 2 should 
    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | party1 |  node1   |   0    | 
    | party1 |  node2   |   0    |       
    | party1 |  node3   |   0    |        

    And the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |    0   | 
    | party1 |  node2   |    0   |       
    | party1 |  node3   |    0   |        

  Scenario: A party cannot delegate more than it has in staking account
    Desciption: A party attempts to delegate more than it has in its staking account to a validator
    
    The parties deposit on staging account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 10000  |  

    Then the parties submit the following delegations:
    | party  | node id  |   amount   | reference | error                               |
    | party1 |  node1   |    10001   |      a    | insufficient balance for delegation |
    | party1 |  node2   |    10002   |      b    | insufficient balance for delegation |    
    | party1 |  node3   |    10003   |      c    | insufficient balance for delegation |

    #we are now in epoch 1 so the delegation balance for epoch 1 should not include the delegation but the hypothetical balance for epoch 2 should 
    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | party1 |  node1   |   0    | 
    | party1 |  node2   |   0    |       
    | party1 |  node3   |   0    |        

    And the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |    0   | 
    | party1 |  node2   |    0   |       
    | party1 |  node3   |    0   |        

  Scenario: A party cannot delegate stake size such that it exceeds maximum amount of stake for a validator
    Desciption: A party attempts to delegate token stake which exceed maximum stake for a validator
      
    The parties deposit on staging account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 10000  |  

    Then the parties submit the following delegations:
    | party  | node id  |   amount  | 
    | party1 |  node1   |    1500   | 
    | party1 |  node2   |    2000   | 
    | party1 |  node3   |    2500   | 

    #we are now in epoch 1 so the delegation balance for epoch 1 should not include the delegation but the hypothetical balance for epoch 2 should 
    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | party1 |  node1   |   0    | 
    | party1 |  node2   |   0    |       
    | party1 |  node3   |   0    |        

    And the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 1500   | 
    | party1 |  node2   | 2000   |       
    | party1 |  node3   | 2500   |        

    #however when the epoch ends and the delegations are processed the max allowed delegation per node is calculated as 
    #roughly 13/1.1 * (13 * 10000 + 1000 + 2000 + 3000) =  ~1507 which means each node will only accept max delegation of ~1507
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 1500   | 
    | party1 |  node2   | 1507   |       
    | party1 |  node3   | 1507   |   

  Scenario: A party changes delegation from one validator another in the same epoch
    Desciption: A party can change delegatation from one Validator to another

    The parties deposit on staging account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 10000  |  

    Then the parties submit the following delegations:
    | party  | node id  |   amount  | 
    | party1 |  node1   |    100   | 
    | party1 |  node2   |    100   | 
    | party1 |  node3   |    100   | 

    And the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 100   | 
    | party1 |  node2   | 100   |       
    | party1 |  node3   | 100   |        

    #advance to the end of the epoch for the delegation to become effective
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 100    | 
    | party1 |  node2   | 100    |       
    | party1 |  node3   | 100    |   

    #start epoch 3
    When time is updated to "2021-08-26T00:00:22Z"   

    #now request to undelegate from node2 and node3 
    Then the parties submit the following undelegations:
    | party  | node id  | amount |
    | party1 |  node2   |  80    |       
    | party1 |  node3   |  90    |   

    Then the parties submit the following delegations:
    | party  | node id  |  amount  | 
    | party1 |  node1   |    80   | 
    | party1 |  node1   |    90   | 

    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   | 270    | 
    | party1 |  node2   | 20     |       
    | party1 |  node3   | 10     |   

    #advance to the end of the epoch for the delegation to become effective
    When time is updated to "2021-08-26T00:00:32Z"    
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   | 270    | 
    | party1 |  node2   | 20     |       
    | party1 |  node3   | 10     | 




