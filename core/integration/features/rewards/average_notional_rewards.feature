Feature: Average position metric rewards

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
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures |
      | ETH/DEC22 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures |

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

  # Scenario: No trader is eligible - no transfer is made
  #   # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
  #   Given the parties submit the following recurring transfers:
  #     | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
  #     | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_AVERAGE_NOTIONAL | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1500                | 50                   |

  #   Then the network moves ahead "1" epochs

  #   And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "1000000" for asset "VEGA"

  # Scenario: eligible party with staking less than threshold doesn't get a reward (0056-REWA-076)
  #   # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
  #   Given the parties submit the following recurring transfers:
  #     | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
  #     | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_AVERAGE_NOTIONAL | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1500                | 0                    |

  #   When the parties submit the following liquidity provision:
  #     | id  | party  | market id | commitment amount | fee | lp type    |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |

  #   And the parties place the following pegged iceberg orders:
  #     | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | buy  | BID              | 90     | 10     |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | sell | ASK              | 90     | 10     |

  #   Then the parties place the following orders:
  #     | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
  #     | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
  #     | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

  #   # let the position update be in the middle of the epoch
  #   Given time is updated to "2023-09-23T00:00:18Z"

  #   Then the network moves ahead "1" epochs

  #   # aux1 has a position of 10
  #   # aux2 has a position of -10
  #   # however aux1 has sufficient vega staked
  #   # and aux2 doesn't
  #   # therefore the transfer is made and the full amount goes to aux1
  #   And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
  #   And "aux1" should have vesting account balance of "10000" for asset "VEGA"

  # Scenario: eligible party with average notional less than threshold doesn't get a reward (0056-REWA-077)
  #   # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
  #   Given the parties submit the following recurring transfers:
  #     | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
  #     | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_AVERAGE_NOTIONAL | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 10000                |

  #   When the parties submit the following liquidity provision:
  #     | id  | party  | market id | commitment amount | fee | lp type    |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |

  #   And the parties place the following pegged iceberg orders:
  #     | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | buy  | BID              | 90     | 10     |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | sell | ASK              | 90     | 10     |

  #   Then the parties place the following orders:
  #     | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
  #     | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
  #     | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

  #   # let the position update be in the middle of the epoch
  #   Given time is updated to "2023-09-23T00:00:18Z"

  #   Then the network moves ahead "1" epochs

  #   # aux1 has a position of 10
  #   # aux2 has a position of -10
  #   # the average notional for both is 5*1000 = 5000 < minimum average notional = 10000 therefore no one gets a reward
  #   And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "1000000" for asset "VEGA"

  #   # advance one epoch
  #   Then the network moves ahead "1" epochs

  #   # now their average notional is 10*1000 = 10000 therefore they both are eligible
  #   And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
  #   And "aux1" should have vesting account balance of "5000" for asset "VEGA"
  #   And "aux2" should have vesting account balance of "5000" for asset "VEGA"

  # Scenario: multiple eligible parties split the reward equally
  #   # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
  #   Given the parties submit the following recurring transfers:
  #     | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
  #     | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_AVERAGE_NOTIONAL | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 0                    |

  #   When the parties submit the following liquidity provision:
  #     | id  | party  | market id | commitment amount | fee | lp type    |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |

  #   And the parties place the following pegged iceberg orders:
  #     | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | buy  | BID              | 90     | 10     |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | sell | ASK              | 90     | 10     |

  #   Then the parties place the following orders:
  #     | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
  #     | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
  #     | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

  #   # let the position update be in the middle of the epoch
  #   Given time is updated to "2023-09-23T00:00:18Z"

  #   Then the network moves ahead "1" epochs

  #   # aux1 has a position of 10
  #   # aux2 has a position of -10
  #   # however aux1 and aux2 have sufficient vega staked
  #   # therefore the transfer is made and the reward amount is split between aux1 and aux2
  #   And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
  #   And "aux1" should have vesting account balance of "5000" for asset "VEGA"
  #   And "aux2" should have vesting account balance of "5000" for asset "VEGA"

  # Scenario: multiple epochs multiple positions (0056-REWA-083)
  #   # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
  #   Given the parties submit the following recurring transfers:
  #     | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
  #     | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_AVERAGE_NOTIONAL | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 0                    |

  #   When the parties submit the following liquidity provision:
  #     | id  | party  | market id | commitment amount | fee | lp type    |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |

  #   And the parties place the following pegged iceberg orders:
  #     | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | buy  | BID              | 90     | 10     |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | sell | ASK              | 90     | 10     |

  #   Then the parties place the following orders:
  #     | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
  #     | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
  #     | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

  #   # let the position update be in the middle of the epoch
  #   Given time is updated to "2023-09-23T00:00:18Z"

  #   Then the network moves ahead "1" epochs

  #   # aux1 has a position of 10
  #   # aux2 has a position of -10
  #   # however aux1 and aux2 have sufficient vega staked
  #   # therefore the transfer is made and the reward amount is split between aux1 and aux2
  #   And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
  #   And "aux1" should have vesting account balance of "5000" for asset "VEGA"
  #   And "aux2" should have vesting account balance of "5000" for asset "VEGA"

  #   # 20% into the epoch, lets get a trade done
  #   Given time is updated to "2023-09-23T00:00:26Z"

  #   Then the parties place the following orders:
  #     | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
  #     | party1 | ETH/DEC21 | buy  | 5      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
  #     | aux2   | ETH/DEC21 | sell | 5      | 1001  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

  #   # 80% into the epoch do another trade
  #   Given time is updated to "2023-09-23T00:00:34Z"

  #   Then the parties place the following orders:
  #     | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
  #     | aux1   | ETH/DEC21 | sell | 20     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | aux1-sell2 |
  #     | party1 | ETH/DEC21 | buy  | 20     | 1002  | 1                | TYPE_LIMIT | TIF_GTC | aux2-buy2  |

  #   Then the network moves ahead "1" epochs

  #   And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "980000" for asset "VEGA"
  #   # lets calculate the expected scores:
  #   # at the beginning of the epoch:
  #   # aux1: time weighted average position = 5
  #   # aux2: time weighted average position = 5
  #   # party1 : time weighted average position = 0

  #   # 20% into the epoch:
  #   # aux2: time weighted average position = 5 * 0 + 10 * 1
  #   # party1 : time weighted average position = 0

  #   # 80% into the epoch:
  #   # aux2: time weighted average position = 5 * 0 + 10 * 1
  #   # party1 : time weighted average position = 0.8 * 5 = 4

  #   # end of epoch
  #   # aux1: time weighted average position = 10
  #   # aux2: (1-10/12) * 10 + 10/12 * 15 = 14.1666666667
  #   # party1: (1-2/12) * 4 + 2/12 * 25 = 7.5

  #   # considering both epochs as window is 2:
  #   # aux1: [5, 10] => metric for window = 7.5
  #   # aux2: [5, 14.1666666667] => metric for window = 9.5833333334
  #   # party1: [0, 7.5] => metric for window = 3.75

  #   # aux1 gets 10000 * 7.5 / 20.8333333334 = 3600
  #   # aux2 gets 10000 * 9.5833333334 / 20.8333333334 = 4600
  #   # party1 gets 10000 * 3.75 / 20.8333333334 = 1,799.9999999942 = 1799

  #   And "aux1" should have vesting account balance of "8600" for asset "VEGA"
  #   And "aux2" should have vesting account balance of "9600" for asset "VEGA"
  #   And "party1" should have vesting account balance of "1799" for asset "VEGA"

  # Scenario: multiple multiple markets - only one in scope
  #   # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
  #   Given the parties submit the following recurring transfers:
  #     | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets   | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
  #     | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_AVERAGE_NOTIONAL | ETH          | ETH/DEC21 | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 0                    |

  #   When the parties submit the following liquidity provision:
  #     | id  | party  | market id | commitment amount | fee | lp type    |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
  #     | lp2 | lpprov | ETH/DEC22 | 90000             | 0.1 | submission |
  #     | lp2 | lpprov | ETH/DEC22 | 90000             | 0.1 | submission |

  #   And the parties place the following pegged iceberg orders:
  #     | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | buy  | BID              | 90     | 10     |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | sell | ASK              | 90     | 10     |
  #     | lpprov | ETH/DEC22 | 90        | 1                    | buy  | BID              | 90     | 10     |
  #     | lpprov | ETH/DEC22 | 90        | 1                    | sell | ASK              | 90     | 10     |

  #   Then the parties place the following orders:
  #     | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
  #     | aux1   | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux2   | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux1   | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
  #     | aux2   | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |
  #     | party1 | ETH/DEC22 | buy  | 5      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | party2 | ETH/DEC22 | sell | 5      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux1   | ETH/DEC22 | buy  | 1      | 1800  | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
  #     | aux2   | ETH/DEC22 | sell | 1      | 2200  | 0                | TYPE_LIMIT | TIF_GTC | sell2     |


  #   # let the position update be in the middle of the epoch
  #   Given time is updated to "2023-09-23T00:00:18Z"

  #   Then the network moves ahead "1" epochs

  #   # only ETH/DEC21 is inscope
  #   # aux1 has a position of 10
  #   # aux2 has a position of -10
  #   # however aux1 and aux2 have sufficient vega staked
  #   # therefore the transfer is made and the reward amount is split between aux1 and aux2
  #   And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
  #   And "aux1" should have vesting account balance of "5000" for asset "VEGA"
  #   And "aux2" should have vesting account balance of "5000" for asset "VEGA"

  # Scenario: If an eligible party held positions in multiple in-scope markets, their average position reward metric should be the sum of the size of their time-weighted-average-position in each market (0056-REWA-082)
  #   # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
  #   Given the parties submit the following recurring transfers:
  #     | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
  #     | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_AVERAGE_NOTIONAL | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 0                    |

  #   When the parties submit the following liquidity provision:
  #     | id  | party  | market id | commitment amount | fee | lp type    |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
  #     | lp2 | lpprov | ETH/DEC22 | 90000             | 0.1 | submission |
  #     | lp2 | lpprov | ETH/DEC22 | 90000             | 0.1 | submission |

  #   And the parties place the following pegged iceberg orders:
  #     | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | buy  | BID              | 90     | 10     |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | sell | ASK              | 90     | 10     |
  #     | lpprov | ETH/DEC22 | 90        | 1                    | buy  | BID              | 90     | 10     |
  #     | lpprov | ETH/DEC22 | 90        | 1                    | sell | ASK              | 90     | 10     |

  #   Then the parties place the following orders:
  #     | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
  #     | aux1   | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux2   | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux1   | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
  #     | aux2   | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |
  #     | party1 | ETH/DEC22 | buy  | 20     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | party2 | ETH/DEC22 | sell | 20     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux1   | ETH/DEC22 | buy  | 1      | 910   | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
  #     | aux2   | ETH/DEC22 | sell | 1      | 1110  | 0                | TYPE_LIMIT | TIF_GTC | sell2     |

  #   # let the position update be in the middle of the epoch
  #   Given time is updated to "2023-09-23T00:00:18Z"

  #   Then the network moves ahead "1" epochs

  #   # only ETH/DEC21 is inscope
  #   # aux1 has a position of 10 => time weighted notional = 5, window = 2 => 2.5 => 10000 * 2.5/15 = 1666
  #   # aux2 has a position of -10 => time weighted notional = 5, window = 2 => 2.5 => 10000 * 2.5/15 = 1666
  #   # party1 has a position of 20 => time weighted notional = 10, window = 2 => 5 => 10000 * 5/15 = 3333
  #   # party2 has a position of -20 => time weighted notional = 10, window = 2 => 5 => 10000 * 5/15 = 3333
  #   And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
  #   And "aux1" should have vesting account balance of "1666" for asset "VEGA"
  #   And "aux2" should have vesting account balance of "1666" for asset "VEGA"
  #   And "party1" should have vesting account balance of "3333" for asset "VEGA"
  #   And "party2" should have vesting account balance of "3333" for asset "VEGA"

  #   # 20% into the epoch, lets get a trade done
  #   Given time is updated to "2023-09-23T00:00:26Z"

  #   Then the parties place the following orders:
  #     | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
  #     | party1 | ETH/DEC21 | buy  | 5      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | aux1-buy1  |
  #     | aux2   | ETH/DEC21 | sell | 5      | 1001  | 1                | TYPE_LIMIT | TIF_GTC | aux2-sell1 |

  #   # 80% into the epoch do another trade
  #   Given time is updated to "2023-09-23T00:00:34Z"

  #   Then the parties place the following orders:
  #     | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
  #     | aux1   | ETH/DEC21 | sell | 20     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | aux1-sell2 |
  #     | party1 | ETH/DEC21 | buy  | 20     | 1002  | 1                | TYPE_LIMIT | TIF_GTC | aux2-buy2  |

  #   Then the network moves ahead "1" epochs

  #   And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "980000" for asset "VEGA"

  #   # epoch1
  #   # market1: aux1 = 5, aux2 = 5
  #   # market2: party1 = 10, party2 = 10
  #   # epoch2
  #   # market1: aux1 = 10, aux2 = 14.1666666667, party1 = 7.5
  #   # market2: party1 = 20, party2 = 20
  #   # considering the window=2:
  #   # aux1 = [5,10] = 7.5
  #   # aux2 = [5,14.1666666667] = 9.5833333334
  #   # party1 = [10, 27.5] = 18.75
  #   # part2 = [10,20] = 15
  #   # aux1 reward = 10000*7.5/50.8333333334 = 1475
  #   # aux2 reward = 10000*9.5833333334/50.8333333334 = 1885
  #   # party1 reward = 10000*18.75/50.8333333334 = 3688
  #   # party2 reward = 10000*15/50.8333333334 = 2950
  #   And "aux1" should have vesting account balance of "3141" for asset "VEGA"
  #   And "aux2" should have vesting account balance of "3551" for asset "VEGA"
  #   And "party1" should have vesting account balance of "7022" for asset "VEGA"
  #   And "party2" should have vesting account balance of "6284" for asset "VEGA"

  # Scenario: If an eligible party opens a position at the beginning of the epoch, their average position reward metric should be equal to the size of the position at the end of the epoch (0056-REWA-078). If an eligible party held an open position at the start of the epoch, their average position reward metric should be equal to the size of the position at the end of the epoch (0056-REWA-079).
  #   When the parties submit the following liquidity provision:
  #     | id  | party  | market id | commitment amount | fee | lp type    |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
  #     | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |

  #   And the parties place the following pegged iceberg orders:
  #     | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | buy  | BID              | 90     | 10     |
  #     | lpprov | ETH/DEC21 | 90        | 1                    | sell | ASK              | 90     | 10     |

  #   Then the parties place the following orders:
  #     | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
  #     | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
  #     | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
  #     | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

  #   # leave opening auction
  #   Then the network moves ahead "1" epochs

  #   # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
  #   Given the parties submit the following recurring transfers:
  #     | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
  #     | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL | VEGA  | 10000  | 2           |           | 1      | DISPATCH_METRIC_AVERAGE_NOTIONAL | ETH          |         | 2           | 1             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 0                    |

  #   # the time is the beginning of the epoch
  #   Then the parties place the following orders:
  #     | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
  #     | party1 | ETH/DEC21 | buy  | 5      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
  #     | party2 | ETH/DEC21 | sell | 5      | 1001  | 1                | TYPE_LIMIT | TIF_GTC | p2-sell1  |

  #   Then the network moves ahead "1" epochs

  #   # aux1 and aux2 have a position of 10 - which is equal to their position held at the beginning of the epoch
  #   # party1 and party2 has position of 5 - which is equal to the position opened at the beginning of the epoch
  #   And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
  #   And "aux1" should have vesting account balance of "3333" for asset "VEGA"
  #   And "aux2" should have vesting account balance of "3333" for asset "VEGA"
  #   And "party1" should have vesting account balance of "1666" for asset "VEGA"
  #   And "party2" should have vesting account balance of "1666" for asset "VEGA"

  Scenario: If an eligible party opens a position half way through the epoch, their average position reward metric should be half the size of the position at the end of the epoch (0056-REWA-080). If an eligible party held an open position at the start of the epoch and closes it half-way through the epoch, their average position reward metric should be equal to the size of that position at the end of the epoch (0056-REWA-081).
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC21 | 90        | 1                    | buy  | BID              | 90     | 10     |
      | lpprov | ETH/DEC21 | 90        | 1                    | sell | ASK              | 90     | 10     |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    # leave opening auction
    Then the network moves ahead "1" epochs

    # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL | VEGA  | 10000  | 2           |           | 1      | DISPATCH_METRIC_AVERAGE_NOTIONAL | ETH          |         | 2           | 1             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 0                    |

    # let the position update be in the middle of the epoch
    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1001  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 5      | 999   | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 5      | 999   | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    # aux1 - sells all of their position mid epoch - so their notional metric is 0.0005
    # aux2 - sells half of its position mid epoch - so their notional metric is 0.0007497
    # party1 - got into position mid epoch so their notional metric is 0.0005005
    # party2 - got into position mid epoch so their notional metric is 0.0002497
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "2500" for asset "VEGA"
    And "aux2" should have vesting account balance of "3748" for asset "VEGA"
    And "party1" should have vesting account balance of "2502" for asset "VEGA"
    And "party2" should have vesting account balance of "1248" for asset "VEGA"

