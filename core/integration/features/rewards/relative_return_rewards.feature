Feature: Relative return rewards

  Background:
    Given the following network parameters are set:
      | name                                              | value  |
      | reward.asset                                      | VEGA   |
      | validators.epoch.length                           | 10s    |
      | validators.delegation.minAmount                   | 10     |
      | reward.staking.delegation.delegatorShare          | 0.883  |
      | reward.staking.delegation.minimumValidatorStake   | 100    |
      | reward.staking.delegation.maxPayoutPerParticipant | 100000 |
      | reward.staking.delegation.competitionLevel        | 1.1    |
      | reward.staking.delegation.minValidators           | 5      |
      | reward.staking.delegation.optimalStakeMultiplier  | 5.0    |
      | network.markPriceUpdateMaximumFrequency           | 0s     |
      | limits.markets.maxPeggedOrders                    | 2      |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 3                 |

    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |


    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e6                    | 1e6                       | default-futures |
      | ETH/DEC22 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e6                    | 1e6                       | default-futures |

    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset | amount    |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | VEGA  | 1000000   |
      | aux1                                                             | ETH   | 100000000 |
      | aux2                                                             | ETH   | 100000000 |
      | trader3                                                          | ETH   | 10000     |
      | trader4                                                          | ETH   | 10000     |
      | lpprov                                                           | ETH   | 200000000 |
      | party1                                                           | ETH   | 100000    |
      | party2                                                           | ETH   | 100000    |


    And the parties deposit on staking account the following amount:
      | party   | asset | amount |
      | aux1    | VEGA  | 2000   |
      | aux2    | VEGA  | 1000   |
      | trader3 | VEGA  | 1500   |
      | trader4 | VEGA  | 1000   |
      | lpprov  | VEGA  | 10000  |
      | party1  | VEGA  | 2000   |
      | party2  | VEGA  | 2000   |


    Given time is updated to "2023-09-23T00:00:00Z"
    Given the average block duration is "1"

    #complete the epoch to advance to a meaningful epoch (can't setup transfer to start at epoch 0)
    Then the network moves ahead "1" epochs

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
      | lp2 | lpprov | ETH/DEC22 | 90000             | 0.1 | submission |
      | lp2 | lpprov | ETH/DEC22 | 90000             | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC21 | 90        | 1                    | buy  | BID              | 90     | 10     |
      | lpprov | ETH/DEC21 | 90        | 1                    | sell | ASK              | 90     | 10     |
      | lpprov | ETH/DEC22 | 90        | 1                    | buy  | BID              | 90     | 10     |
      | lpprov | ETH/DEC22 | 90        | 1                    | sell | ASK              | 90     | 10     |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1   | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2   | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux1   | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | aux2   | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |
      | party1 | ETH/DEC22 | buy  | 5      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | ETH/DEC22 | sell | 5      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux1   | ETH/DEC22 | buy  | 1      | 1800  | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
      | aux2   | ETH/DEC22 | sell | 1      | 2200  | 0                | TYPE_LIMIT | TIF_GTC | sell2     |

  Scenario: No trader is eligible - no transfer is made
    # setup recurring transfer to the reward account - this will start at the  end of this epoch 
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RELATIVE_RETURN | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RELATIVE_RETURN | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1500                | 50                   |

    Then the network moves ahead "1" epochs

    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "1000000" for asset "VEGA"

  Scenario: eligible party with staking less than threshold doesn't get a reward (0056-REWA-076)
    # setup recurring transfer to the reward account - this will start at the  end of this epoch 
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RELATIVE_RETURN | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RELATIVE_RETURN | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1500                | 0                    |

    Then the network moves ahead "1" epochs

    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 10     | 999   | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 10     | 999   | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    # M2M
    # party1 = -30
    # aux1 = 20
    # aux2 = 10
    # relative return metric for party1 = 0
    # relative return metric for aux1 = 20/5 = 4
    # aux2 is not eligible because they don't have sufficient staking
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "10000" for asset "VEGA"

  Scenario: eligible party with average notional less than threshold doesn't get a reward (0056-REWA-077)
    # setup recurring transfer to the reward account - this will start at the  end of this epoch 
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RELATIVE_RETURN | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RELATIVE_RETURN | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 5000                 |

    Then the network moves ahead "1" epochs

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 6      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 6      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | buy  | 6      | 999   | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | sell | 6      | 999   | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    # M2M
    # party1 = -18
    # aux1 = 8
    # aux2 = 10
    # aux1 has ~4000 average notional < 5000 therefore they're not eligible
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux2" should have vesting account balance of "10000" for asset "VEGA"

  Scenario: multiple eligible parties split the reward (0056-REWA-084,0056-REWA-085)
    # setup recurring transfer to the reward account - this will start at the  end of this epoch 
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RELATIVE_RETURN | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RELATIVE_RETURN | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 0                    |

    Then the network moves ahead "1" epochs

    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 10     | 999   | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 10     | 999   | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    # party1 is the loser and has negative metric therefore is not getting any reward.
    # aux1 and aux2 split the reward proportionally to their metric
    # both aux1 and aux2 are eligible
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "6666" for asset "VEGA"
    And "aux2" should have vesting account balance of "3333" for asset "VEGA"

  Scenario: multiple epochs multiple positions (0056-REWA-087)
    Given the network moves ahead "1" epochs

    # setup recurring transfer to the reward account - this will start at the  end of this epoch 
    And the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RELATIVE_RETURN | VEGA  | 10000  | 2           |           | 1      | DISPATCH_METRIC_RELATIVE_RETURN | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 0                    |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 5      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
      | aux2   | ETH/DEC21 | sell | 5      | 1001  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party2 | ETH/DEC21 | buy  | 5      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
      | aux1   | ETH/DEC21 | sell | 5      | 1002  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

    Then the network moves ahead "1" epochs

    # aux2 m2m -25 p=15 => ret = -1.6666666666666667
    # aux1 m2m 20 p=5  => ret = 4
    # party1 m2m 5 p=5 =>  ret = 1
    # metric over a window=2:
    # aux1 = 2
    # party1 = 0.5
    # aux1 gets 10000 * 2/2.5 = 8000
    # party1 gets 10000 * 0.5/2.5 = 2000
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "8000" for asset "VEGA"
    And "party1" should have vesting account balance of "2000" for asset "VEGA"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | aux1   | ETH/DEC21 | sell | 20     | 998   | 0                | TYPE_LIMIT | TIF_GTC | aux1-sell2 |
      | party1 | ETH/DEC21 | buy  | 20     | 998   | 1                | TYPE_LIMIT | TIF_GTC | aux2-buy2  |

    Then the network moves ahead "1" epochs

    # in this epoch:
    # aux2 m2m 60 p=15 => ret = 4
    # aux1 m2m -20 p=15 => ret = -1.3333333333333333
    # party1 m2m -20 p=25 => ret = -0.8
    # party2 m2m -20 p=5 => ret = -4
    # metric over a window=2:
    # aux1 = [4, -1.3333333333333333] => 1.3333333333
    # party1 = [1, -0.8] => 0.1
    # aux2 = [-1.6666666666666667, 4] => 1.1666666667
    # party2 = [0, -4] => 0
    # aux1 gets: 10000*1.3333333333/2.6 = 5128
    # party1 gets: 10000*0.1/2.6 = 384
    # aux2 gets: 10000*1.1666666667/2.6 = 4487
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "980000" for asset "VEGA"
    And "aux1" should have vesting account balance of "13128" for asset "VEGA"
    And "party1" should have vesting account balance of "2384" for asset "VEGA"
    And "aux2" should have vesting account balance of "4487" for asset "VEGA"

  Scenario: multiple multiple markets - only one in scope
    Given the network moves ahead "1" epochs
    And the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets   | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RELATIVE_RETURN | VEGA  | 10000  | 2           |           | 1      | DISPATCH_METRIC_RELATIVE_RETURN | ETH          | ETH/DEC21 | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 0                    |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 5      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
      | aux2   | ETH/DEC21 | sell | 5      | 1001  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party2 | ETH/DEC21 | buy  | 5      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
      | aux1   | ETH/DEC21 | sell | 5      | 1002  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC22 | buy  | 5      | 2001  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
      | aux2   | ETH/DEC22 | sell | 5      | 2001  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party2 | ETH/DEC22 | buy  | 5      | 2002  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
      | aux1   | ETH/DEC22 | sell | 5      | 2002  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

    Then the network moves ahead "1" epochs
    # aux2 m2m -25 p=15 => ret = -1.6666666666666667
    # aux1 m2m 20 p=5  => ret = 4
    # party1 m2m 5 p=5 =>  ret = 1
    # metric over a window=2:
    # aux1 = 2
    # party1 = 0.5
    # aux1 gets 10000 * 2/2.5 = 8000
    # party1 gets 10000 * 0.5/2.5 = 2000
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "8000" for asset "VEGA"
    And "party1" should have vesting account balance of "2000" for asset "VEGA"

  Scenario: If an eligible party is participating in multiple in-scope markets, their relative returns reward metric should be the sum of their relative returns from each market (0056-REWA-085,0056-REWA-086)
    Then the network moves ahead "1" epochs
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RELATIVE_RETURN | VEGA  | 10000  | 2           |           | 1      | DISPATCH_METRIC_RELATIVE_RETURN | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 0                    |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 5      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
      | aux2   | ETH/DEC21 | sell | 5      | 1001  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party2 | ETH/DEC21 | buy  | 5      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
      | aux1   | ETH/DEC21 | sell | 5      | 1002  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

    # let the position update be in the middle of the epoch
    Given time is updated to "2023-09-23T00:00:30Z"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC22 | buy  | 5      | 1999  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
      | aux2   | ETH/DEC22 | sell | 5      | 1999  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party2 | ETH/DEC22 | buy  | 5      | 1997  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
      | aux1   | ETH/DEC22 | sell | 5      | 1997  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

    Then the network moves ahead "1" epochs

    # market ETH/DEC21
    # aux2 m2m=-25 p=15 => ret=-1.6666666666666667
    # aux1 m2m=20 p=5 => ret=4
    # party1 m2m=5 p=5 =>  ret=1

    # market ETH/DEC22
    # party1 m2m=-25 p=7.5 => ret=-3.3333333333333333
    # party2 m2m=15 p=2.5 => ret=6
    # aux2 m2m=10 p=2.5 => ret=4

    # metric over a window=2
    # party1 = -1.6666666666666667-3.3333333333333333 = 0
    # party2 = 6/2 = 3
    # aux1 = 4/2 = 2
    # aux2 = (4-1.6666666666666667)/2 = 1.1666666667

    # reward
    # aux1 = 10000*2/6.1666666667 = 3243
    # aux2 = 10000*1.1666666667/6.1666666667 = 1891
    # party2 = 10000*3/6.1666666667 = 4864

    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "3243" for asset "VEGA"
    And "aux2" should have vesting account balance of "1891" for asset "VEGA"
    And "party2" should have vesting account balance of "4864" for asset "VEGA"