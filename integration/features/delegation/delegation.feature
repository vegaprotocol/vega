Feature: Staking & Delegation 

  Background:
    Given the following network parameters are set:
      | name                                            | value |
      | reward.asset                                    | VEGA  |
      | validators.epoch.length                         | 9s    |
      | validators.delegation.minAmount                 | 10    |
      | reward.staking.delegation.payoutDelay           | 0s    |
      | reward.staking.delegation.competitionLevel      | 1.1   |


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

    And the parties deposit on staking account the following amount:  
      | party  | asset  | amount |
      | party1 | VEGA   | 10000  |

    #complete the first epoch for the self delegation to take effect
    Then time is updated to "2021-08-26T00:00:10Z"
    Then time is updated to "2021-08-26T00:00:11Z"

  Scenario: A party can delegate to a validator and undelegate at the end of an epoch
    Desciption: A party with a balance in the staking account can delegate to a validator

    When the parties submit the following delegations:
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
    | party  | node id  | amount |      when     |
    | party1 |  node1   |  100   |  end of epoch |
    | party1 |  node2   |  200   |  end of epoch |              
    | party1 |  node3   |  300   |  end of epoch |           

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

    When the parties submit the following delegations:
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

    When the parties submit the following delegations:
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

    When the parties submit the following delegations:
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
    #roughly 1.1/13 * (13 * 10000 + 1000 + 2000 + 3000) =  ~1507 which means each node will only accept max delegation of ~1507
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 1500   | 
    | party1 |  node2   | 1507   |       
    | party1 |  node3   | 1507   | 

    When time is updated to "2021-08-26T00:00:33Z"    
    When time is updated to "2021-08-26T00:00:43Z"    
  
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   | 1500   | 
    | party1 |  node2   | 1507   |       
    | party1 |  node3   | 1507   | 

    And the parties submit the following undelegations:
    | party  | node id  | amount |     when     |
    | party1 |  node1   |  500   | end of epoch | 
    | party1 |  node2   |  500   | end of epoch |     
    | party1 |  node3   |  500   | end of epoch | 

    When time is updated to "2021-08-26T00:00:42Z"    
    When time is updated to "2021-08-26T00:00:53Z"

    Then the parties should have the following delegation balances for epoch 4:
    | party  | node id  | amount |
    | party1 |  node1   | 1500   | 
    | party1 |  node2   | 1507   |       
    | party1 |  node3   | 1507   | 

    When time is updated to "2021-08-26T00:00:54Z"    
    When time is updated to "2021-08-26T00:01:03Z"
    When time is updated to "2021-08-26T00:01:13Z"

    Then the parties should have the following delegation balances for epoch 5:
    | party  | node id  | amount |
    | party1 |  node1   | 1500   | 
    | party1 |  node2   | 1507   |       
    | party1 |  node3   | 1507   | 

    When time is updated to "2021-08-26T00:01:14Z"
    When time is updated to "2021-08-26T00:01:24Z"

    Then the parties should have the following delegation balances for epoch 6:
    | party  | node id  | amount |
    | party1 |  node1   | 1500   | 
    | party1 |  node2   | 1507   |       
    | party1 |  node3   | 1507   | 

  Scenario: A party changes delegation from one validator to another in the same epoch
    Desciption: A party can change delegatation from one Validator to another

    When the parties submit the following delegations:
    | party  | node id  |   amount  | 
    | party1 |  node1   |    100   | 
    | party1 |  node2   |    100   | 
    | party1 |  node3   |    100   | 

    Then the parties should have the following delegation balances for epoch 2:
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

    #start epoch 2
    When time is updated to "2021-08-26T00:00:22Z"   

    #now request to undelegate from node2 and node3 
    And the parties submit the following undelegations:
    | party  | node id  | amount |      when     |
    | party1 |  node2   |  80    |  end of epoch |     
    | party1 |  node3   |  90    |  end of epoch | 

    Then the parties submit the following delegations:
    | party  | node id  |  amount | 
    | party1 |  node1   |    80   | 
    | party1 |  node1   |    90   | 

    And the parties should have the following delegation balances for epoch 3:
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

    When the parties submit the following delegations:
    | party  | node id   |   amount | reference | error           |
    | party1 |  unknown1 |    100   |      a    | invalid node ID |
    | party1 |  unknonw2 |    200   |      b    | invalid node ID |    

  Scenario: A party cannot undelegate from an unknown node
    Desciption: A party should fail in trying to undelegate from a non existing node

    When the parties submit the following undelegations:
    | party  | node id   |   amount |     when     | reference | error           |
    | party1 |  unknown1 |    100   | end of epoch |      a    | invalid node ID |
    | party1 |  unknonw2 |    200   | end of epoch |      b    | invalid node ID |      

  Scenario: A party cannot delegate more than their staking account balance considering all active and pending delegation 
    Desciption: A party has pending delegations and is trying to exceed their stake account balance delegation, 
    i.e. the balance of their pending delegation + requested delegation exceeds stake account balance
  
    When the parties submit the following delegations:
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
    
    When the parties submit the following delegations:
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
    | party  | node id  | amount |    when      |
    | party1 |  node1   |  1000  | end of epoch |

    #party1 already have 423 delegated so they can only theoretically delegate 10000-423 = 9577
    Then the parties submit the following delegations:
    | party  | node id  |  amount | reference | error                               |
    | party1 |  node2   |   1000  |           |                                     |    
    | party1 |  node2   |   8578  |     a     | insufficient balance for delegation |    

  Scenario: A party can request delegate and undelegate from the same node at the same epoch such that the request can balance each other without affecting the actual delegate balance    Description: party requests to delegate to node1 at the end of the epoch and regrets it and undelegate the whole amount to delegate it to node2
  
    When the parties submit the following delegations:
    | party  | node id  |  amount | 
    | party1 |  node1   |   1000  | 

    And the parties submit the following undelegations:
    | party  | node id  | amount |    when      |
    | party1 |  node1   |  1000  | end of epoch |

    And the parties submit the following delegations:
    | party  | node id  |  amount | 
    | party1 |  node2   |   1000  | 

    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 0      | 
    | party1 |  node2   | 1000   | 

  Scenario: A party has active delegations and submits an undelegate request followed by a delegation request that covers only part of the undelegation such that the undelegation still takes place
    Description: A party delegated tokens to node1 at previous epoch such that the delegations is now active and is requesting to undelegate some of the tokens at the end of the current epoch. Then regret some of it and submit a delegation request that undoes some of the undelegation but still some of it remains. 

    When the parties submit the following delegations:
    | party  | node id  |  amount | 
    | party1 |  node1   |   1000  | 

      #advance to the end of the epoch
    When time is updated to "2021-08-26T00:00:21Z"    

      #start a new epoch 
    When time is updated to "2021-08-26T00:00:22Z" 
    Then the parties submit the following undelegations:
    | party  | node id  | amount |    when      |
    | party1 |  node1   |  1000  | end of epoch |

    And the parties submit the following delegations:
    | party  | node id  |  amount | 
    | party1 |  node1   |   100   |

    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   | 100    | 

  Scenario: A party cannot undelegate more than the delegated balance 
    Description: A party trying to undeleagte from a node more than the amount that was delegated to it should fail 

    And the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  100   | 
    | party1 |  node2   |  200   |       
    | party1 |  node3   |  300   |   

    #advance to the beginning of the next epoch 
    When time is updated to "2021-08-26T00:00:21Z"    
    When time is updated to "2021-08-26T00:00:22Z"    

    Then the parties submit the following undelegations:
    | party  | node id   |   amount |     when     | reference | error                                    |
    | party1 |  node1    |    101   | end of epoch |      a    | incorrect token amount for undelegation  |
    | party1 |  node2    |    201   | end of epoch |      b    | incorrect token amount for undelegation  |    
    | party1 |  node3    |    301   | end of epoch |      c    | incorrect token amount for undelegation  |    
    | party1 |  node1    |    100   | end of epoch |           |                                          |
    | party1 |  node2    |    200   | end of epoch |           |                                          |    
    | party1 |  node3    |    300   | end of epoch |           |                                          |  

  Scenario: A node can self delegate to itself and undelegate at the end of an epoch
    Desciption: A node with a balance in the staking account can delegate to itself

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | node1   |  node1   |  1000  | 
    
    #we are now in epoch 1 so the delegation balance for epoch 1 should not include the delegation but the hypothetical balance for epoch 2 should 
    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | node1  |  node1   |  10000 | 
        
    And the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | node1  |  node1   | 11000  | 
    
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | node1 |  node1   | 11000  | 

    When time is updated to "2021-08-26T00:00:22Z"  
    Then the parties submit the following undelegations:
    | party  | node id  | amount |    when      |
    | node1  |  node1   |  5000  | end of epoch |
    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | node1  |  node1   | 11000  | 
    
    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | node1  |  node1   |  6000  | 
    
    #advance to the end of epoch for the undelegation to take place 
    When time is updated to "2021-08-26T00:00:32Z"  
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | node1  |  node1   |  6000  | 
      
  Scenario: A party can undelegate all of their stake at the end of the epoch
    Desciption: A with delegated stake and pending delegation can request to undelegate all. This will be translated to undelegating all the stake they have at the time and will be executed at the end of the epoch

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    #end epoch 1 for the delegation to take effect   
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    #start epoch2
    When time is updated to "2021-08-26T00:00:22Z"  
    Then the parties submit the following delegations:
    | party  | node id  | amount |
    | party1  |  node1   |  50   | 
    | party1  |  node2   |  123  | 
    
    #we are in the middle of epoch2 and the party1 requests to undelegate all of their stake to party1
    Then the parties submit the following undelegations:
    | party  | node id  | amount |    when      |
    | party1 |  node1   |  0     | end of epoch |

    #therefore we expect their expected balance for epoch 3 for party1 node1 is 0
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |   0    |
    | party1 |  node2   |   123  |  
    
    #advance to the end of epoch2 for the delegations/undelegations to be applied
    When time is updated to "2021-08-26T00:00:32Z"  
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |   0    | 
    | party1 |  node2   |   123  |    

  Scenario: A party can request to undelegate their stake or part of it right now
    Desciption: A party with delegated stake and/or pending delegation can request to undelegate all or part of it immediately as opposed to at the end of the epoch

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    #end epoch 1 for the delegation to take effect   
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    #start epoch2
    When time is updated to "2021-08-26T00:00:22Z"  
    Then the parties submit the following delegations:
    | party  | node id  | amount |
    | party1  |  node1   |  50   | 
    | party1  |  node2   |  123  | 
    
    #we are in the middle of epoch2 and the party1 requests to undelegate all of their stake to party1
    Then the parties submit the following undelegations:
    | party  | node id  | amount |  when  |
    | party1 |  node1   |  0     |  now   |

    #therefore we expect their current balance for epoch 2 for party1 node1 is 0
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |   0    |
    | party1 |  node2   |   0    |
    
    #the expected balance for the following epoch should be the same for party1/node1 because undelegation has already been applied and for party1/node1 should include the expected delegation
    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |   0    |
    | party1 |  node2   |  123   |

    #advance to the end of epoch2 for the delegations/undelegations to be applied - and expect no change for node1 because undelegation has already taken place
    #the delegation from party1 to node2 will have been applied 
    When time is updated to "2021-08-26T00:00:32Z"
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |   0    | 
    | party1 |  node2   |   123  |  
  
 Scenario: A party withdraws from their staking account during an epoch - their stake is being undelegated automatically to match the difference
    Desciption: A party with delegated stake withdraws from their staking account during an epoch - at the end of the epoch when delegations are processed the party will be forced to undelegate the difference between the stake they have delegated and their staking account balance. 

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    #end epoch 1 for the delegation to take effect   
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    # start epoch2
    When time is updated to "2021-08-26T00:00:22Z"    
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   |  9500  |

    # advance to the end of epoch2
    When time is updated to "2021-08-26T00:00:32Z" 
    # we expect the actual balance of epoch 2 for party1 has changed retrospectively to 500 to reflect that the party has insufficient balance in their staking account to cover 1000 delegated tokens   
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |  500   |   

    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |  500   |

  Scenario: Undelegate now followed by withdraw - same behaviour as either underlegate now or withdraw

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    #end epoch 1 for the delegation to take effect   
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    Then the parties submit the following undelegations:
    | party  | node id  | amount |  when  |
    | party1 |  node1   |  500   |  now   |

    # start epoch2
    When time is updated to "2021-08-26T00:00:22Z"    
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   |  9600  |

    # advance to the end of epoch2
    When time is updated to "2021-08-26T00:00:32Z" 
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |  400   |   

    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |  400   |     

  Scenario: Withdrawal followed by undelegate now (same epoch) - same behaviour as either underlegate now or withdraw

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    #end epoch 1 for the delegation to take effect   
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    # start epoch2
    When time is updated to "2021-08-26T00:00:22Z"    
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   |  9500  |

    When time is updated to "2021-08-26T00:00:23Z"   

    Then the parties submit the following undelegations:
    | party  | node id  | amount |  when  |
    | party1 |  node1   |    400 |  now   |

    # advance to the end of epoch2
    When time is updated to "2021-08-26T00:00:32Z" 
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |    500 |   

    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |    500 |    

Scenario: Withdrawal followed by undelegate now (next epoch) - results in additional undelegation

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    #end epoch 1 for the delegation to take effect   
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    # start epoch2
    When time is updated to "2021-08-26T00:00:22Z"    
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   |  9500  |

    # advance to the end of epoch2
    When time is updated to "2021-08-26T00:00:32Z" 
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |    500 |   

    Then the parties submit the following undelegations:
    | party  | node id  | amount |  when  |
    | party1 |  node1   |    200 |  now   |

    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |    300 |

    Then the parties submit the following undelegations:
    | party  | node id  | amount |  when  |
    | party1 |  node1   |    200 |  now   |

    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |    100 |    

    When time is updated to "2021-08-26T00:00:42Z" 

    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |    100 |    


Scenario: Mix undelegate now & end of epoch

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    #end epoch 1 for the delegation to take effect   
    When time is updated to "2021-08-26T00:00:21Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  | 

    # start epoch2
    When time is updated to "2021-08-26T00:00:22Z"    

    Then the parties submit the following undelegations:
    | party  | node id  | amount | when         |
    | party1 |  node1   |    500 | now          |
    | party1 |  node1   |    400 | end of epoch |

    # advance to the end of epoch2
    When time is updated to "2021-08-26T00:00:32Z" 
    # we expect the actual balance of epoch 2 for party1 has changed retrospectively to 500 to reflect that the party has insufficient balance in their staking account to cover 1000 delegated tokens   
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |    500 |   

    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |    100 |    

    When time is updated to "2021-08-26T00:00:42Z" 

    And the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |    100 |    
