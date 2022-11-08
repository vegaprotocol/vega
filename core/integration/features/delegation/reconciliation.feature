Feature: Staking & Delegation 

  Background:
    Given the following network parameters are set:
      | name                                              | value  |
      | reward.asset                                      | VEGA   |
      | validators.epoch.length                           | 120s   |
      | validators.delegation.minAmount                   | 10     |
      | reward.staking.delegation.delegatorShare          |  0.883 |
      | reward.staking.delegation.minimumValidatorStake   |  100   |
      | reward.staking.delegation.maxPayoutPerParticipant | 100000 |
      | reward.staking.delegation.competitionLevel        |  1.1   |
      | reward.staking.delegation.minValidators           |  5     |
      | reward.staking.delegation.optimalStakeMultiplier  |  5.0   |
      | network.markPriceUpdateMaximumFrequency           | 0s     |

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
      | party1 | VEGA   | 700    |
      | party2 | VEGA   | 20000  |
      | party3 | VEGA   | 500    |

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  100   | 
    | party1 |  node2   |  200   |       
    | party1 |  node3   |  300   |     
    | party2 |  node2   |  400   |     
    | party2 |  node3   |  500   |     
    | party2 |  node4   |  600   |     

    #complete the first epoch for the self delegation to take effect
    Then the network moves ahead "63" blocks


  Scenario: Party submits delegations which by the time they are processed are uncovered
    Description: when the party submits the delegation request they have cover for all but by the time the epoch ends they only leave enough to cover some of the delegations
    
    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party3 |  node1   |  100   | 
    | party3 |  node2   |  100   |       
    | party3 |  node3   |  100   |    
    | party3 |  node4   |  100   |    
    | party3 |  node5   |  100   |    

    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party3 |  node1   | 100    | 
    | party3 |  node2   | 100    |       
    | party3 |  node3   | 100    |   
    | party3 |  node4   | 100    |   
    | party3 |  node5   | 100    |   
    
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party3 | VEGA   |  350   | 

    When the network moves ahead "63" blocks    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party3 |  node1   | 30     | 
    | party3 |  node2   | 30     |       
    | party3 |  node3   | 30     |   
    | party3 |  node4   | 30     |   
    | party3 |  node5   | 30     |   

  Scenario: Party dissociation gets reconciled during the epoch
    Description: A party with delegation dissociates all of their tokens which causes their whole delegation to be undone within 30 seconds and reflected before the epoch ends
    
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   |  700   | 

    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | party1 |  node1   | 100    | 
    | party1 |  node2   | 200    |       
    | party1 |  node3   | 300    |   
    | party2 |  node2   | 400    |   
    | party2 |  node3   | 500    |   
    | party2 |  node4   | 600    |  

    When the network moves ahead "16" blocks
    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | party1 |  node1   | 0      | 
    | party1 |  node2   | 0      |       
    | party1 |  node3   | 0      |   
    | party2 |  node2   | 400    |   
    | party2 |  node3   | 500    |   
    | party2 |  node4   | 600    |  

    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 0      | 
    | party1 |  node2   | 0      |       
    | party1 |  node3   | 0      |  
    | party2 |  node2   | 400    |   
    | party2 |  node3   | 500    |   
    | party2 |  node4   | 600    |   

    When the network moves ahead "47" blocks
    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | party1 |  node1   | 0      | 
    | party1 |  node2   | 0      |       
    | party1 |  node3   | 0      |   
    | party2 |  node2   | 400    |   
    | party2 |  node3   | 500    |   
    | party2 |  node4   | 600    |  

    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 0      | 
    | party1 |  node2   | 0      |       
    | party1 |  node3   | 0      |   
    | party2 |  node2   | 400    |   
    | party2 |  node3   | 500    |   
    | party2 |  node4   | 600    |  

  Scenario: Party dissociation gets reconciled during the epoch incrementally
    Description: A party with delegation dissociates some tokens in multiple withdrawals which causes their whole delegation to be undone within 30 seconds and reflected before the epoch ends
   
    Given the parties submit the following undelegations:
    | party  | node id  | amount |  when         |
    | party1 |  node1   |  100   |  end of epoch |      

    #epoch 1 withdraw 100 
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   |  100   | 

    When the network moves ahead "2" blocks

    #epoch 1 withdraw another 300 - in total 400 meaning only 300 remain associated
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   |  300   | 

    When the network moves ahead "14" blocks

    # within 30 seconds we expect to have seen events of the nomination corrected accordingly
    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | party1 |  node1   | 50     | 
    | party1 |  node2   | 100    |       
    | party1 |  node3   | 150    |   
    | party2 |  node2   | 400    |   
    | party2 |  node3   | 500    |   
    | party2 |  node4   | 600    | 

    #no changes in these 30 seconds so expect balances to not change 
    When the network moves ahead "15" blocks
    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | party1 |  node1   | 50     | 
    | party1 |  node2   | 100    |       
    | party1 |  node3   | 150    |   
    | party2 |  node2   | 400    |   
    | party2 |  node3   | 500    |   
    | party2 |  node4   | 600    |  

    #still in epoch 1 withdraw the remaining 300 tokens
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   |  300   | 

    When the network moves ahead "16" blocks
    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | party1 |  node1   | 0      | 
    | party1 |  node2   | 0      |       
    | party1 |  node3   | 0      |   
    | party2 |  node2   | 400    |   
    | party2 |  node3   | 500    |   
    | party2 |  node4   | 600    |  

    # the adjustment should be published for the next epoch as well 
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 0      | 
    | party1 |  node2   | 0      |       
    | party1 |  node3   | 0      |   
    
    #epoch 1 is ending
    When the network moves ahead "15" blocks
    Then the parties should have the following delegation balances for epoch 1:
    | party  | node id  | amount |
    | party1 |  node1   | 0      | 
    | party1 |  node2   | 0      |       
    | party1 |  node3   | 0      |   
    | party2 |  node2   | 400    |   
    | party2 |  node3   | 500    |   
    | party2 |  node4   | 600    |  

    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   | 0      | 
    | party1 |  node2   | 0      |       
    | party1 |  node3   | 0      |   
    | party2 |  node2   | 400    |   
    | party2 |  node3   | 500    |   
    | party2 |  node4   | 600    |  

 Scenario: A party withdraws during the epoch and has pending delegations for next epoch
    Description: A party with a balance in the staking account can delegate to a validator

    #we are in epoch one delegating for epoch 2
    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party3 |  node1   |  100   | 
    | party3 |  node2   |  100   |       
    | party3 |  node3   |  100   |   
    | party3 |  node4   |  100   | 
    | party3 |  node5   |  100   | 

    # the expected balance for epoch 2
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party3 |  node1   | 100    | 
    | party3 |  node2   | 100    |       
    | party3 |  node3   | 100    |  
    | party3 |  node4   | 100    |  
    | party3 |  node5   | 100    |  

    #advance to epoch 2
    When the network moves ahead "63" blocks

    #parties change the delegation for epoch 3 to undelegate 100 from each node and delegate the 400 to the last node 
    #to set a delegation of 500 to a single validator
    Then the parties submit the following undelegations:
    | party  | node id  | amount |      when     |
    | party3 |  node1   |  100   |  end of epoch |
    | party3 |  node2   |  100   |  end of epoch |              
    | party3 |  node3   |  100   |  end of epoch |         
    | party3 |  node4   |  100   |  end of epoch |         

    Then the parties submit the following delegations:
    | party  | node id  | amount |
    | party3 |  node5   |  400   | 

    #so the expected balance for epoch 3 is 500 in node 5 ano nothing in the other nodes
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party3 |  node1   | 0      | 
    | party3 |  node2   | 0      |       
    | party3 |  node3   | 0      |  
    | party3 |  node4   | 0      |  
    | party3 |  node5   | 500    |  

    #then during epoch 2 the party withdraws their stake 
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount  |
    | party3 | VEGA   |  500  |

    #within 30 seconds the *active delegations* are removed so the balances for epoch 2 are reset to 0
    When the network moves ahead "20" blocks
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party3 |  node1   | 0      | 
    | party3 |  node2   | 0      |       
    | party3 |  node3   | 0      |  
    | party3 |  node4   | 0      |  
    | party3 |  node5   | 0      |  

    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party3 |  node1   | 0      | 
    | party3 |  node2   | 0      |       
    | party3 |  node3   | 0      |  
    | party3 |  node4   | 0      |  
    | party3 |  node5   | 0      |  

    # the party deposits 500 again
    When the network moves ahead "5" blocks
    And the parties deposit on staking account the following amount:  
    | party  | asset  | amount |
    | party3 | VEGA   | 500    |

    # and delegates again 500 to node5 for epoch 3
    Then the parties submit the following delegations:
    | party  | node id  | amount |
    | party3 |  node5   |  500   |    

    # which gives a total of 900 
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party3 |  node1   | 0      | 
    | party3 |  node2   | 0      |       
    | party3 |  node3   | 0      |  
    | party3 |  node4   | 0      |  
    | party3 |  node5   | 500    |  
  
    # when the epoch ends we enforce the stronger rules and allow only 500 to go through .
    When the network moves ahead "50" blocks
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party3 |  node1   | 0      | 
    | party3 |  node2   | 0      |       
    | party3 |  node3   | 0      |  
    | party3 |  node4   | 0      |  
    | party3 |  node5   | 500    |
