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
      | id     | staking account balance | pub_key |
      | node1  |         1000000         |   pk1   |
      | node2  |         1000000         |   pk2   |
      | node3  |         1000000         |   pk3   |
      | node4  |         1000000         |   pk4   |
      | node5  |         1000000         |   pk5   |
      | node6  |         1000000         |   pk6   |
      | node7  |         1000000         |   pk7   |
      | node8  |         1000000         |   pk8   |
      | node9  |         1000000         |   pk9   |
      | node10 |         1000000         |   pk10  |
      | node11 |         1000000         |   pk11  |
      | node12 |         1000000         |   pk12  |
      | node13 |         1000000         |   pk13  |

    #set up the self delegation of the validators
    Then the parties submit the following delegations:
      | party  | node id  | amount |
      | pk1    |  node1   | 10000  | 
      | pk2    |  node2   | 10000  |       
      | pk3    |  node3   | 10000  | 
      | pk4    |  node4   | 10000  | 
      | pk5    |  node5   | 10000  | 
      | pk6    |  node6   | 10000  | 
      | pk7    |  node7   | 10000  | 
      | pk8    |  node8   | 10000  | 
      | pk9    |  node9   | 10000  | 
      | pk10   |  node10  | 10000  | 
      | pk11   |  node11  | 10000  | 
      | pk12   |  node12  | 10000  | 
      | pk13   |  node13  | 10000  | 

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

    And the parties receive the following reward for epoch 1:
    | party  | asset | amount |
    | party1 | VEGA  |  201   | 
    | pk1    | VEGA  |  3832  | 
    | pk2    | VEGA  |  3837  | 
    | pk3    | VEGA  |  3841  | 
    | pk4    | VEGA  |  3828  | 
    | pk5    | VEGA  |  3828  | 
    | pk6    | VEGA  |  3828  | 
    | pk7    | VEGA  |  3828  | 
    | pk8    | VEGA  |  3828  | 
    | pk10   | VEGA  |  3828  | 
    | pk11   | VEGA  |  3828  | 
    | pk12   | VEGA  |  3828  | 
    | pk13   | VEGA  |  3828  | 