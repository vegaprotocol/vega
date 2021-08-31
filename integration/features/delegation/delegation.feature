Feature: Staking & Delegation 

  Background:
    Given the following network parameters are set:
      | name                                            | value |
      | governance.vote.asset                           | VEGA  |
      | validators.epoch.length                         | 10s   |
      | validators.delegation.minAmount                 | 10    |
      | reward.staking.delegation.payoutDelay           | 0s    |
      
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

    The parties deposit on staking account the following amount:  
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

    #advance to the end of epoch for the undelegation to take place 
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

  Scenario: A party changes delegation from one validator to another in the same epoch
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

  Scenario: A party cannot delegate to an unknown node 
    Desciption: A party should fail in trying to delegate to a non existing node

    The parties deposit on staging account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 10000  |  

    Then the parties submit the following delegations:
    | party  | node id   |   amount | reference | error           |
    | party1 |  unknown1 |    100   |      a    | invalid node ID |
    | party1 |  unknonw2 |    200   |      b    | invalid node ID |    

  Scenario: A party cannot undelegate from an unknown node
    Desciption: A party should fail in trying to undelegate from a non existing node

    The parties deposit on staging account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 10000  |  

    Then the parties submit the following undelegations:
    | party  | node id   |   amount | reference | error           |
    | party1 |  unknown1 |    100   |      a    | invalid node ID |
    | party1 |  unknonw2 |    200   |      b    | invalid node ID |      

  Scenario: A party cannot delegate more than their staking account balance considering all active and pending delegation 
   Desciption: A party has pending delegations and is trying to exceed their stake account balance delegation, 
    i.e. the balance of their pending delegation + requested delegation exceeds stake account balance

    The parties deposit on staging account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 10000   |  
    
    Then the parties submit the following delegations:
    | party  | node id  |  amount | reference | error                               |
    | party1 |  node1   |   5000  |           |                                     |
    | party1 |  node2   |   6000  |     a     | insufficient balance for delegation |    

    #advance to the end of the epoch
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 1423   | 

    #start a new epoch
    When time is updated to "2021-08-26T00:00:22Z"    

    #party1 already have 1423 delegated so they can only theoretically delegate 10000-1423 = 8577
    Then the parties submit the following delegations:
    | party  | node id  |  amount | reference | error                               |
    | party1 |  node2   |   1000  |           |                                     |    
    | party1 |  node2   |   7578  |     a     | insufficient balance for delegation |    

  Scenario: A party cannot delegate more than their staking account balance considering all active and pending undelegation 
   Desciption: A party has pending delegations and undelegations and is trying to exceed their stake account balance delegation, 
    i.e. the balance of their pending delegation + requested delegation exceeds stake account balance

    The parties deposit on staging account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 10000   |  
    
    Then the parties submit the following delegations:
    | party  | node id  |  amount | reference | error                               |
    | party1 |  node1   |   5000  |           |                                     |

    #advance to the end of the epoch
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 1423   | 

    #start a new epoch
    When time is updated to "2021-08-26T00:00:22Z"    
    Then the parties submit the following undelegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000   | 

    #party1 already have 423 delegated so they can only theoretically delegate 10000-423 = 9577
    Then the parties submit the following delegations:
    | party  | node id  |  amount | reference | error                               |
    | party1 |  node2   |   1000  |           |                                     |    
    | party1 |  node2   |   8578  |     a     | insufficient balance for delegation |    

  Scenario: A party can request delegate and undelegate from the same node at the same epoch such that the request can balance each other without affecting the actual delegate balance
    Description: party requests to delegate to node1 at the end of the epoch and regrets it and undelegate the whole amount to delegate it to node2

    The parties deposit on staging account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 10000   |  
    
    Then the parties submit the following delegations:
    | party  | node id  |  amount | 
    | party1 |  node1   |   1000  | 

    Then the parties submit the following undelegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    Then the parties submit the following delegations:
    | party  | node id  |  amount | 
    | party1 |  node2   |   1000  | 

    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 0      | 
    | party1 |  node2   | 1000   | 

  Scenario: A party has active delegations and submits an undelegate request followed by a delegation request that covers only part of the undelegation such that the undelegation still takes place
    Description: A party delegated tokens to node1 at previous epoch such that the delegations is now active and is requesting to undelegate some of the tokens at the end of the current epoch. Then regret some of it and submit a delegation request that undoes some of the undelegation but still some of it remains. 

    The parties deposit on staging account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 10000   |  
    
    Then the parties submit the following delegations:
    | party  | node id  |  amount | 
    | party1 |  node1   |   1000  | 

     #advance to the end of the epoch
    Then time is updated to "2021-08-26T00:00:21Z"    

     #start a new epoch 
    When time is updated to "2021-08-26T00:00:22Z" 
    Then the parties submit the following undelegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    Then the parties submit the following delegations:
    | party  | node id  |  amount | 
    | party1 |  node1   |   100   |

     Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   | 100    | 

  Scenario: A party cannot undelegate more than the delegated balance 
    Description: A party trying to undeleagte from a node more than the amount that was delegated to it should fail 

    The parties deposit on staging account the following amount:  
      | party  | asset  | amount |
      | party1 | VEGA   | 10000  |

    Then the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  100   | 
    | party1 |  node2   |  200   |       
    | party1 |  node3   |  300   |   

    #advance to the beginning of the next epoch 
    Then time is updated to "2021-08-26T00:00:21Z"    
    Then time is updated to "2021-08-26T00:00:22Z"    

    Then the parties submit the following undelegations:
    | party  | node id   |   amount | reference | error                                    |
    | party1 |  node1    |    101   |      a    | incorrect token amount for undelegation  |
    | party1 |  node2    |    201   |      b    | incorrect token amount for undelegation  |    
    | party1 |  node3    |    301   |      c    | incorrect token amount for undelegation  |    
    | party1 |  node1    |    100   |           |                                          |
    | party1 |  node2    |    200   |           |                                          |    
    | party1 |  node3    |    300   |           |                                          |  

