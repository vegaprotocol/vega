Feature: Staking & Delegation

  Background:
    Given the following network parameters are set:
      | name                                              |  value                   |
      | reward.asset                                      |  VEGA                    |
      | validators.epoch.length                           |  10s                     |
      | validators.delegation.minAmount                   |  10                      |
      | reward.staking.delegation.delegatorShare          |  0.883                   |
      | reward.staking.delegation.minimumValidatorStake   |  100                     |
      | reward.staking.delegation.maxPayoutPerParticipant | 100000                   |
      | reward.staking.delegation.competitionLevel        |  1.1                     |
      | reward.staking.delegation.minValidators           |  5                       |
      | reward.staking.delegation.optimalStakeMultiplier  |  5.0                     |

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

  Scenario: Parties get rewarded for a full epoch of having delegated stake - the reward amount is capped 
    Description: Parties have had their tokens delegated to nodes for a full epoch and get rewarded for the full epoch. 
    
    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 50000 |   

    #advance to the end of the epoch
    Then the network moves ahead "7" blocks

    #verify validator score 
    Then the validators should have the following val scores for epoch 1:
    | node id | validator score  | normalised score |
    |  node1  |      0.07734     |     0.07734      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07657     |     0.07657      | 

    #50k are being distributed
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
    | pk9    | VEGA  |  3828  | 
    | pk10   | VEGA  |  3828  | 
    | pk11   | VEGA  |  3828  | 
    | pk12   | VEGA  |  3828  | 
    | pk13   | VEGA  |  3828  | 

    Then "party1" should have general account balance of "201" for asset "VEGA"
    And "pk1" should have general account balance of "3832" for asset "VEGA"
    And "pk2" should have general account balance of "3837" for asset "VEGA"
    And "pk3" should have general account balance of "3841" for asset "VEGA"
    And "pk4" should have general account balance of "3828" for asset "VEGA"
    And "pk5" should have general account balance of "3828" for asset "VEGA"
    And "pk6" should have general account balance of "3828" for asset "VEGA"
    And "pk7" should have general account balance of "3828" for asset "VEGA"
    And "pk8" should have general account balance of "3828" for asset "VEGA"
    And "pk9" should have general account balance of "3828" for asset "VEGA"
    And "pk10" should have general account balance of "3828" for asset "VEGA"
    And "pk11" should have general account balance of "3828" for asset "VEGA"
    And "pk12" should have general account balance of "3828" for asset "VEGA"
    And "pk13" should have general account balance of "3828" for asset "VEGA"

    #top up to 25000
    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 24991  |  

    Then the network moves ahead "7" blocks
    And the parties receive the following reward for epoch 2:
    | party  | asset | amount |
    | party1 | VEGA  |  99    | 
    | pk1    | VEGA  |  1916  | 
    | pk2    | VEGA  |  1918  | 
    | pk3    | VEGA  |  1920  | 
    | pk4    | VEGA  |  1914  | 
    | pk5    | VEGA  |  1914  | 
    | pk6    | VEGA  |  1914  | 
    | pk7    | VEGA  |  1914  | 
    | pk8    | VEGA  |  1914  | 
    | pk9    | VEGA  |  1914  | 
    | pk10   | VEGA  |  1914  | 
    | pk11   | VEGA  |  1914  | 
    | pk12   | VEGA  |  1914  | 
    | pk13   | VEGA  |  1914  | 

    Then "party1" should have general account balance of "300" for asset "VEGA"
    And "pk1" should have general account balance of "5748" for asset "VEGA"
    And "pk2" should have general account balance of "5755" for asset "VEGA"
    And "pk3" should have general account balance of "5761" for asset "VEGA"
    And "pk4" should have general account balance of "5742" for asset "VEGA"
    And "pk5" should have general account balance of "5742" for asset "VEGA"
    And "pk6" should have general account balance of "5742" for asset "VEGA"
    And "pk7" should have general account balance of "5742" for asset "VEGA"
    And "pk8" should have general account balance of "5742" for asset "VEGA"
    And "pk9" should have general account balance of "5742" for asset "VEGA"
    And "pk10" should have general account balance of "5742" for asset "VEGA"
    And "pk11" should have general account balance of "5742" for asset "VEGA"
    And "pk12" should have general account balance of "5742" for asset "VEGA"
    And "pk13" should have general account balance of "5742" for asset "VEGA"

    # top up to 12507
    When the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 12500  |  

    Then the network moves ahead "7" blocks
    And the parties receive the following reward for epoch 3:
    | party  | asset | amount |
    | party1 | VEGA  |  49    | 
    | pk1    | VEGA  |  958   | 
    | pk2    | VEGA  |  959   | 
    | pk3    | VEGA  |  961   | 
    | pk4    | VEGA  |  957   | 
    | pk5    | VEGA  |  957   | 
    | pk6    | VEGA  |  957   | 
    | pk7    | VEGA  |  957   | 
    | pk8    | VEGA  |  957   | 
    | pk9    | VEGA  |  957   | 
    | pk10   | VEGA  |  957   | 
    | pk11   | VEGA  |  957   | 
    | pk12   | VEGA  |  957   | 
    | pk13   | VEGA  |  957   | 

    Then the network moves ahead "7" blocks
    Then "party1" should have general account balance of "349" for asset "VEGA"
    And "pk1" should have general account balance of "6706" for asset "VEGA"
    And "pk2" should have general account balance of "6714" for asset "VEGA"
    And "pk3" should have general account balance of "6722" for asset "VEGA"
    And "pk4" should have general account balance of "6699" for asset "VEGA"
    And "pk5" should have general account balance of "6699" for asset "VEGA"
    And "pk6" should have general account balance of "6699" for asset "VEGA"
    And "pk7" should have general account balance of "6699" for asset "VEGA"
    And "pk8" should have general account balance of "6699" for asset "VEGA"
    And "pk9" should have general account balance of "6699" for asset "VEGA"
    And "pk10" should have general account balance of "6699" for asset "VEGA"
    And "pk11" should have general account balance of "6699" for asset "VEGA"
    And "pk12" should have general account balance of "6699" for asset "VEGA"
    And "pk13" should have general account balance of "6699" for asset "VEGA"
