Feature: Auto Delegation 

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
    Then time is updated to "2021-08-26T00:00:10Z"
    Then time is updated to "2021-08-26T00:00:11Z"

Scenario: A party enters auto delegation mode by nominating all of its associated stake 
    Desciption: Once a party has delegated all of its associated stake, it enters auto delegation mode. Once it has more stake associated, it gets automatically distributed between the validators maintaining the same distribution.  

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

    #advance to the end of the epoch
    Then time is updated to "2021-08-26T00:00:21Z"

    #by now party1 is in auto delegation mode - start the next epoch
    Then time is updated to "2021-08-26T00:00:22Z"

    And the parties deposit on staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 1000   |

    # move to the end of the epoch for auto delegation to take place for party1
    When time is updated to "2021-08-26T00:00:32Z"
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

Scenario: A party dissociates VEGA token leading to undelegation propotionally to their current delegation
    Desciption: Once a party dissociates VEGA tokens their delegation is adjusted automatically to reflect that in the same propotion as the delegation with respect to the amount withdrawn

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

    #advance to the end of the epoch 1 and start epoch 2
    When time is updated to "2021-08-26T00:00:21Z"   
    When time is updated to "2021-08-26T00:00:22Z"    
    Given the parties withdraw from staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   |  5000  |

    #advance to the end of epoch 2
    When time is updated to "2021-08-26T00:00:32Z"    
    Then the parties should have the following delegation balances for epoch 2:
    | party  | node id  | amount |
    | party1 |  node1   |  1000  |
    | party1 |  node2   |  1000  |
    | party1 |  node3   |  500  |
    | party1 |  node4   |  500  |
    | party1 |  node5   |  500  |
    | party1 |  node6   |  500  |
    | party1 |  node7   |  250   |
    | party1 |  node8   |  250   |
    | party1 |  node9   |  250   |
    | party1 |  node10  |  250   |

Scenario: A party enters auto delegation mode by nominating all of its associated stake, once more tokens are associated they are distributed however not in epochs when manual delegation takes place
    Desciption: Once a party has delegated all of its associated stake, it enters auto delegation mode. In the following epoch they submit manual delegations so the auto delegation doesn't kick in. An epoch later the remaining tokens are distributed.

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

    #advance to the end of the epoch
    Then time is updated to "2021-08-26T00:00:21Z"

    #by now party1 is in auto delegation mode - start the next epoch
    Then time is updated to "2021-08-26T00:00:22Z"

    And the parties deposit on staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 1500   |

    Given the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  50  |
    | party1 |  node2   |  50  |
    | party1 |  node3   |  50  |
    | party1 |  node4   |  50  |
    | party1 |  node5   |  50  |
    | party1 |  node6   |  50  |
    | party1 |  node7   |  50  |
    | party1 |  node8   |  50  |
    | party1 |  node9   |  50  |
    | party1 |  node10  |  50  |

    # move to the end of the epoch - auto delegation will not take place party1 because they requested to manually delegate
    When time is updated to "2021-08-26T00:00:32Z"
    When time is updated to "2021-08-26T00:00:33Z"
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
    When time is updated to "2021-08-26T00:00:43Z"
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

Scenario: A party qualifies to auto delegation by delegating all of their associated tokens however by manually undelegation they exit auto delegation mode
    Desciption: Once a party dissociates VEGA tokens their delegation is adjusted automatically to reflect that in the same propotion as the delegation with respect to the amount withdrawn

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

    #advance to the end of the epoch 1 and start epoch 2
    When time is updated to "2021-08-26T00:00:21Z"   
    When time is updated to "2021-08-26T00:00:22Z"    
   
    Then the parties submit the following undelegations:
    | party  | node id  | amount |  when  |
    | party1 |  node1   |  100   |  now   |

    #advance to the end of epoch 2
    When time is updated to "2021-08-26T00:00:32Z"   

    #as we're out of auto delegation, due to the undelegateNow, no auto delegation will take place until all amount is fully delegated again 
    Then the parties should have the following delegation balances for epoch 2:
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

    #start epoch 3 
    When time is updated to "2021-08-26T00:00:33Z"   
    #increase the stake to make it availble however it will not be distributed because auto delegation has been switched off by the undelegation
    Then the parties deposit on staking account the following amount:  
    | party  | asset  | amount |
    | party1 | VEGA   | 1000   |

    Then the parties submit the following delegations:
    | party  | node id  | amount | 
    | party1 |  node10  |  100   | 

    #by now we qualify again to auto delegation but we don't in this epoch due to the manual delegation 
    #end epoch3 
    When time is updated to "2021-08-26T00:00:43Z"   
    #start and end epoch4 
    When time is updated to "2021-08-26T00:00:44Z"   
    When time is updated to "2021-08-26T00:00:54Z"   
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