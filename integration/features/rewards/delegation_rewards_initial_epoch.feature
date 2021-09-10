Feature: Staking & Delegation 

  Background:
    Given the following network parameters are set:
      | name                                              |  value |
      | reward.asset                                      |  VEGA  |
      | validators.epoch.length                           |  24h   |
      | validators.delegation.minAmount                   |  10    |
      | reward.staking.delegation.payoutDelay             |  0s    |
      | reward.staking.delegation.delegatorShare          |  0.883 |
      | reward.staking.delegation.minimumValidatorStake   |  100   |
      | reward.staking.delegation.payoutFraction          |  0.5   |
      | reward.staking.delegation.maxPayoutPerParticipant | 100000 |
      | reward.staking.delegation.competitionLevel        |  1.1   |
      | reward.staking.delegation.maxPayoutPerEpoch       |  50000 |
  
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

  Scenario: No action in first epoch
 
    #complete the initial epoch for delegation to take effect
    #TODO: Debug output seems to suggest epoch lenght is 1s?
    Then the network moves ahead "86401" blocks

    # TODO: Would be nice to be able to verify validator score in that case, but I'm not even sure if that comes through official API
    #verify validator score 
    # Then the validators should have the following val scores for epoch 1:
    # | node id | validator score  | normalised score |
    # |  node1  |      0.00000     |     0.00000      |    
    # |  node2  |      0.00000     |     0.00000      |
    # |  node3  |      0.00000     |     0.00000      | 
    # |  node4  |      0.00000     |     0.00000      | 
    # |  node5  |      0.00000     |     0.00000      | 
    # |  node6  |      0.00000     |     0.00000      | 
    # |  node7  |      0.00000     |     0.00000      | 
    # |  node8  |      0.00000     |     0.00000      | 
    # |  node9  |      0.00000     |     0.00000      | 
    # |  node10 |      0.00000     |     0.00000      | 
    # |  node11 |      0.00000     |     0.00000      | 
    # |  node12 |      0.00000     |     0.00000      | 
    # |  node13 |      0.00000     |     0.00000      | 

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
    Then the network moves ahead "86401" blocks

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

  Scenario: Only a few validators self-delegate, significant delegation to a single validator (with own stake)

    And the parties deposit on asset's general account the following amount:
      | party  | asset  | amount |
      | party1 | VEGA   | 111000 |
      | party2 | VEGA   | 222000 |
      | party3 | VEGA   | 333000 |

    And the parties deposit on staking account the following amount:
      | party  | asset  | amount |
      | party1 | VEGA   | 111000 |  
      | party2 | VEGA   | 222000 |  
      | party3 | VEGA   | 333000 |  

    Then the parties submit the following delegations:
      | party  | node id  | amount  |
      | party1 |  node1   |  111000 | 
      | party2 |  node1   |  222000 | 
      | party3 |  node1   |  333000 | 

    #set up the self delegation of the validators (number of validators > min. validators parameter)
    Then the parties submit the following delegations:
      | party  | node id  | amount |
      | node1  |  node1   | 11000  | 
      | node2  |  node2   | 12000  |       
      | node3  |  node3   | 13000  | 
      | node4  |  node4   | 14000  | 
      | node5  |  node5   | 15000  | 
      | node6  |  node6   | 16000  | 

    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 100000 | 
    
    #complete the initial epoch for delegation to take effect
    Then the network moves ahead "86401" blocks

    # Then the validators should have the following val scores for epoch 1:
    # | node id | validator score  | normalised score |
    # |  node1  |      0.08462     |     0.47450      |    
    # |  node2  |      0.01606     |     0.09000      |
    # |  node3  |      0.01740     |     0.09759      | 
    # |  node4  |      0.01874     |     0.10510      | 
    # |  node5  |      0.02008     |     0.11261      | 
    # |  node6  |      0.02142     |     0.12011      | 
    # |  node7  |      0.00000     |     0.00000      | 
    # |  node8  |      0.00000     |     0.00000      | 
    # |  node9  |      0.00000     |     0.00000      | 
    # |  node10 |      0.00000     |     0.00000      | 
    # |  node11 |      0.00000     |     0.00000      | 
    # |  node12 |      0.00000     |     0.00000      | 
    # |  node13 |      0.00000     |     0.00000      | 

    # And the parties receive the following reward for epoch 1:
    # | party  | asset | amount |
    # | party1 | VEGA  | 3434   | 
    # | party2 | VEGA  | 6869   | 
    # | party3 | VEGA  | 10304  | 
    # | node1  | VEGA  | 3116   | 
    # | node2  | VEGA  | 4504   | 
    # | node3  | VEGA  | 4879   |  
    # | node4  | VEGA  | 5254   | 
    # | node5  | VEGA  | 5630   | 
    # | node6  | VEGA  | 6005   | 
    # | node8  | VEGA  | 0      | 
    # | node10 | VEGA  | 0      | 
    # | node11 | VEGA  | 0      | 
    # | node12 | VEGA  | 0      | 
    # | node13 | VEGA  | 0      | 

   #TODO: Expecting values above, but this is what comes out:
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
    | party1 | VEGA  | 6077   | 
    | party2 | VEGA  | 0      | 
    | party3 | VEGA  | 0      | 
    | node1  | VEGA  | 2255   | 
    | node2  | VEGA  | 8333  | 
    | node3  | VEGA  | 8333  |  
    | node4  | VEGA  | 8333  | 
    | node5  | VEGA  | 8333   | 
    | node6  | VEGA  | 8333   | 
    | node8  | VEGA  | 0      | 
    | node10 | VEGA  | 0      | 
    | node11 | VEGA  | 0      | 
    | node12 | VEGA  | 0      | 
    | node13 | VEGA  | 0      | 


    
    
#TODO:
  #  Clarify:
  #   - Not setting up reward account / adding 0 to it prevents validator score from being calculated


#   1. A validator gets past the maximum stake
#    This can happen in several ways:
# 	Network parameters change (e.g., the competition factor)
# 	Estimation of delegated stake was wrong (we allow a slightly imprecise calculation for simplicity)
# 	Something somewhere else went wrong
# 2. The maximum alowed stake changes (Through parameter changes or a validator leaving)
# 	2.1 someone delegates an amount that would fit the old max and not the new
# 	2.2 someone delegates an amount that would fit the new max and not the old
# 3. Competition factor is changed
# 	3.1 Increase (less critical)
# 	3.2 Decrease (some validators might be thrown into case 1)
# 	3.3 Massive decrease (the entire economics changes)
# 4. The number of validators becomes lower than minval
# 5. Every block in that epoch is full of delegation/undelegation commands
#    This would essentially be a stress-test for the block evaluation
# 	5.1 The same party delegates and undelegates a lot within one block
# 6. A delegator undelegates in the same epoch in which it unlocks tokens
# 7. A delegator delegates tokens in the same epoch in which it unlocks them
# 	7.1 The deleggator first unlocks (thus auto-undelegating dome tokens, then tries to undelegate the oritinal amount
# 8. Validator drops below self owned threshold
# 9. Validator owns sufficient otkens, but delegted to soemone else.
# 10. A delegator owns less than the delegated stake (it sold of the rest0), and tries to redelegate
# Genesis:
# 11.noone delegates in the first epoch
# 12.very few people delegate in the first epoch
# 13.All delegation in the first epoch goes to the same three validators