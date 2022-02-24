Feature: Auto Delegation 

Background:
    Given the following network parameters are set:
      | name                                              | value  |
      | reward.asset                                      | VEGA   |
      | validators.epoch.length                           | 10s    |
      | validators.delegation.minAmount                   | 10     |
      | reward.staking.delegation.delegatorShare          |  0.883 |
      | reward.staking.delegation.minimumValidatorStake   |  100   |
      | reward.staking.delegation.maxPayoutPerParticipant | 100000 |
      | reward.staking.delegation.competitionLevel        |  1.1   |
      | reward.staking.delegation.minValidators           |  5     |
      | reward.staking.delegation.optimalStakeMultiplier  |  5.0   |


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

    And the parties deposit on staking account the following amount:  
      | party  | asset  | amount |
      | party1 | VEGA   | 10000  |
      | party2 | VEGA   | 20000  |

    #complete the first epoch for the self delegation to take effect
    Then the network moves ahead "7" blocks

Scenario: A party enters auto delegation mode by nominating all of its associated stake (0059-STKG-019)
    Description: Once a party has delegated all of its associated stake, it enters auto delegation mode. Once it has more stake associated, it gets automatically distributed between the validators maintaining the same distribution.  

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  |
    | party1 |  node2   |  1000  |
    | party1 |  node3   |  1000  |
    | party1 |  node4   |  1000  |
    | party1 |  node5   |  1000  |
    | party1 |  node6   |  1000  |
    | party1 |  node7   |  1000  |
    | party1 |  node8   |  1000  |
    | party1 |  node9   |  1000  |
    | party1 |  node10  |  1000  |

    #advance to the end of the second epoch
    #by now party1 is in auto delegation mode - start the next epoch
    Then the network moves ahead "7" blocks

    And the parties deposit on staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 1000   |

    # move to the end of the third epoch for auto delegation to take place for party1
    When the network moves ahead "7" blocks
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   | 1100   | 
    | party1 |  node2   | 1100   | 
    | party1 |  node3   | 1100   | 
    | party1 |  node4   | 1100   | 
    | party1 |  node5   | 1100   | 
    | party1 |  node6   | 1100   | 
    | party1 |  node7   | 1100   | 
    | party1 |  node8   | 1100   | 
    | party1 |  node9   | 1100   | 
    | party1 |  node10  | 1100   |

Scenario: A party dissociates VEGA token leading to undelegation proportionally to their current delegation (0059-STKG-012, 0059-STKG-018)
    Description: Once a party dissociates VEGA tokens their delegation is adjusted automatically to reflect that in the same propotion as the delegation with respect to the amount withdrawn

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  2000  |
    | party1 |  node2   |  2000  |
    | party1 |  node3   |  1000  |
    | party1 |  node4   |  1000  |
    | party1 |  node5   |  1000  |
    | party1 |  node6   |  1000  |
    | party1 |  node7   |  500   |
    | party1 |  node8   |  500   |
    | party1 |  node9   |  500   |
    | party1 |  node10  |  500   |

    #advance to the end of the epoch 2 and start epoch 3
    When the network moves ahead "7" blocks  
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   |  5000  |

    #advance to the end of epoch 3
    When the network moves ahead "7" blocks
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  |
    | party1 |  node2   |  1000  |
    | party1 |  node3   |   500  |
    | party1 |  node4   |   500  |
    | party1 |  node5   |   500  |
    | party1 |  node6   |   500  |
    | party1 |  node7   |   250  |
    | party1 |  node8   |   250  |
    | party1 |  node9   |   250  |
    | party1 |  node10  |   250  |

Scenario: A party enters auto delegation mode by nominating all of its associated stake, once more tokens are associated they are distributed however not in epochs when manual delegation takes place (0059-STKG-020)
    Description: Once a party has delegated all of its associated stake, it enters auto delegation mode. In the following epoch they submit manual delegations so the auto delegation doesn't kick in. An epoch later the remaining tokens are distributed.

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  |
    | party1 |  node2   |  1000  |
    | party1 |  node3   |  1000  |
    | party1 |  node4   |  1000  |
    | party1 |  node5   |  1000  |
    | party1 |  node6   |  1000  |
    | party1 |  node7   |  1000  |
    | party1 |  node8   |  1000  |
    | party1 |  node9   |  1000  |
    | party1 |  node10  |  1000  |

    #advance to the end of the epoch and start a new epoch
    Given the network moves ahead "7" blocks
    And the parties deposit on staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 1500   |

    And the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  50    |
    | party1 |  node2   |  50    |
    | party1 |  node3   |  50    |
    | party1 |  node4   |  50    |
    | party1 |  node5   |  50    |
    | party1 |  node6   |  50    |
    | party1 |  node7   |  50    |
    | party1 |  node8   |  50    |
    | party1 |  node9   |  50    |
    | party1 |  node10  |  50    |

    # move to the end of the epoch - auto delegation will not take place party1 because they requested to manually delegate
    When the network moves ahead "7" blocks
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   | 1050   | 
    | party1 |  node2   | 1050   | 
    | party1 |  node3   | 1050   | 
    | party1 |  node4   | 1050   | 
    | party1 |  node5   | 1050   | 
    | party1 |  node6   | 1050   | 
    | party1 |  node7   | 1050   | 
    | party1 |  node8   | 1050   | 
    | party1 |  node9   | 1050   | 
    | party1 |  node10  | 1050   |

    #on the end of the next epoch however the remaining 1000 tokens will get auto delegated 
    When the network moves ahead "7" blocks
    Then the parties should have the following delegation balances for epoch 4:
    | party  | node id  | amount |
    | party1 |  node1   | 1150   | 
    | party1 |  node2   | 1150   | 
    | party1 |  node3   | 1150   | 
    | party1 |  node4   | 1150   | 
    | party1 |  node5   | 1150   | 
    | party1 |  node6   | 1150   | 
    | party1 |  node7   | 1150   | 
    | party1 |  node8   | 1150   | 
    | party1 |  node9   | 1150   | 
    | party1 |  node10  | 1150   |

Scenario: A party qualifies to auto delegation by delegating all of their associated tokens however by manually undelegation they exit auto delegation mode (0059-STKG-013, STKG-0014, 0059-STKG-021)
    Description: Once a party dissociates VEGA tokens their delegation is adjusted automatically to reflect that in the same proportion as the delegation with respect to the amount withdrawn

    When the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  2000  |
    | party1 |  node2   |  2000  |
    | party1 |  node3   |  1000  |
    | party1 |  node4   |  1000  |
    | party1 |  node5   |  1000  |
    | party1 |  node6   |  1000  |
    | party1 |  node7   |  500   |
    | party1 |  node8   |  500   |
    | party1 |  node9   |  500   |
    | party1 |  node10  |  500   |

    #advance to the end of the epoch 2 and start epoch 3
    When the network moves ahead "7" blocks"   
    
    Then the parties submit the following undelegations:
    | party  | node id  | amount |  when  |
    | party1 |  node1   |  100   |  now   |

    #advance to the end of epoch 3
    When the network moves ahead "7" blocks 

    #as we're out of auto delegation, due to the undelegateNow, no auto delegation will take place until all amount is fully delegated again 
    Then the parties should have the following delegation balances for epoch 3:
    | party  | node id  | amount |
    | party1 |  node1   |  1900  |
    | party1 |  node2   |  2000  |
    | party1 |  node3   |  1000  |
    | party1 |  node4   |  1000  |
    | party1 |  node5   |  1000  |
    | party1 |  node6   |  1000  |
    | party1 |  node7   |  500   |
    | party1 |  node8   |  500   |
    | party1 |  node9   |  500   |
    | party1 |  node10  |  500   |

    #increase the stake to make it availble however it will not be distributed because auto delegation has been switched off by the undelegation
    Then the parties deposit on staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 1000   |

    Then the parties submit the following delegations:
    | party  | node id  | amount | 
    | party1 |  node10  |  100   | 

    #by now we qualify again to auto delegation but we don't in this epoch due to the manual delegation 
    #end epoch3 - start and end epoch4
    When the network moves ahead "7" blocks
    #end epoch4 - start and end epoch5
    When the network moves ahead "7" blocks
    Then the parties should have the following delegation balances for epoch 5:
    | party  | node id  | amount |
    | party1 |  node1   |  2090  |
    | party1 |  node2   |  2200  |
    | party1 |  node3   |  1100  |
    | party1 |  node4   |  1100  |
    | party1 |  node5   |  1100  |
    | party1 |  node6   |  1100  |
    | party1 |  node7   |  550   |
    | party1 |  node8   |  550   |
    | party1 |  node9   |  550   |
    | party1 |  node10  |  660   |

    #verifying auto delegation works on recurring delegations  
    Then the parties deposit on staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 1000   |
    #end epoch5 - start and end epoch6
    When the network moves ahead "7" blocks
    Then the parties should have the following delegation balances for epoch 6:
    | party  | node id  | amount |
    | party1 |  node1   |  2280  |
    | party1 |  node2   |  2400  |
    | party1 |  node3   |  1200  |
    | party1 |  node4   |  1200  |
    | party1 |  node5   |  1200  |
    | party1 |  node6   |  1200  |
    | party1 |  node7   |  600   |
    | party1 |  node8   |  600   |
    | party1 |  node9   |  600   |
    | party1 |  node10  |  720   |