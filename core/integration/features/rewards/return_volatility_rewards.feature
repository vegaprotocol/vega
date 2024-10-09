Feature: Return volatility rewards

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
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca4ffffffffff | VEGA  | 100000    |
      | party2                                                           | ETH   | 100000    |
      | aux1                                                             | ETH   | 100000000 |
      | aux2                                                             | ETH   | 100000000 |
      | trader3                                                          | ETH   | 10000     |
      | trader4                                                          | ETH   | 10000     |
      | lpprov                                                           | ETH   | 200000000 |
      | party1                                                           | ETH   | 100000    |
      | party2                                                           | ETH   | 100000    |


    And the parties deposit on staking account the following amount:
      | party   | asset | amount |
      | aux1    | VEGA  | 2500   |
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
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |
      | aux1  | ETH/DEC22 | buy  | 5      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC22 | sell | 5      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux1  | ETH/DEC22 | buy  | 1      | 1800  | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
      | aux2  | ETH/DEC22 | sell | 1      | 2200  | 0                | TYPE_LIMIT | TIF_GTC | sell2     |


  Scenario: No trader is eligible - no transfer is made
    # setup recurring transfer to the reward account - this will start at the end of this epoch
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RETURN_VOLATILITY | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1500                | 50                   |

    Then the network moves ahead "1" epochs

    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "1000000" for asset "VEGA"

  Scenario: variance is 0 no reward is given no transfer is made
    # setup recurring transfer to the reward account - this will start at the end of this epoch
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RETURN_VOLATILITY | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1500                | 0                    |

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
    # as we only have this returns in scope, for each party the variance is 0 and no transfer is made.
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "1000000" for asset "VEGA"

  Scenario: eligible party with staking less than threshold doesn't get a reward (0056-REWA-076)
    # setup recurring transfer to the reward account - this will start at the end of this epoch
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RETURN_VOLATILITY | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 2500                | 0                    |

    Then the network moves ahead "1" epochs

    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 10     | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 10     | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | sell | 2      | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | buy  | 2      | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | buy  | 3      | 1004  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | sell | 3      | 1004  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    # looking at the returns of all parties for a window of 2:
    # aux1 = [4,1] => variance = 4.5
    # aux2 = [0,0] => N/A
    # party1 = [2,1] => variance = 0.5 => however party1 has insufficient stake so isn't eligible
    # party2 = [0] => N/A
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "10000" for asset "VEGA"

  Scenario: eligible party with average notional less than threshold doesn't get a reward (0056-REWA-077)
    # setup recurring transfer to the reward account - this will start at the end of this epoch
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RETURN_VOLATILITY | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 10000                |

    Then the network moves ahead "1" epochs

    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 10     | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 10     | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | sell | 2      | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | buy  | 2      | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | buy  | 3      | 1004  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | sell | 3      | 1004  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    # looking at the returns of all parties for a window of 2:
    # aux1 = [4,1] => variance = 4.5
    # aux2 = [0,0] => N/A
    # party1 = [2,1] => variance = 0.5 => has insufficient average notional < 10000
    # party2 = [0] => N/A
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "10000" for asset "VEGA"

  Scenario: multiple eligible parties split the reward (0056-REWA-088, 0056-REWA-089, 0056-REWA-208)
    # setup recurring transfer to the reward account - this will start at the end of this epoch
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RETURN_VOLATILITY | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 0                    |

    Then the network moves ahead "1" epochs

    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 10     | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 10     | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | sell | 2      | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | buy  | 2      | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | buy  | 3      | 1004  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | sell | 3      | 1004  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    # looking at the returns of all parties for a window of 2:
    # aux1 = [4,1] => variance = 1/4.5 = 0.2222222222
    # aux2 = [0,0] => N/A
    # party1 = [2,1] => variance = 1/0.5 => 2 has sufficient average therefore is eligible
    # party2 = [0] => N/A
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "1000" for asset "VEGA"
    And "party1" should have vesting account balance of "9000" for asset "VEGA"

  Scenario: Given the following dispatch metrics, if an `eligible keys` list is specified in the recurring transfer, only parties included in the list and meeting other eligibility criteria should receive a score (0056-REWA-218)
    # setup recurring transfer to the reward account - this will start at the end of this epoch
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement | eligible_keys |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RETURN_VOLATILITY | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 0                    | party1        |

    Then the network moves ahead "1" epochs

    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 10     | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 10     | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | sell | 2      | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | buy  | 2      | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | buy  | 3      | 1004  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | sell | 3      | 1004  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    # looking at the returns of all parties for a window of 2:
    # aux1 = [4,1] => variance = 1/4.5 = 0.2222222222
    # aux2 = [0,0] => N/A
    # party1 = [2,1] => variance = 1/0.5 => 2 has sufficient average therefore is eligible
    # party2 = [0] => N/A
    # but only party1 is in eligible keys so they take the whole lot. 
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "party1" should have vesting account balance of "10000" for asset "VEGA"


  Scenario: multiple multiple markets - only one in scope
    # setup recurring transfer to the reward account - this will start at the end of this epoch
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets   | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RETURN_VOLATILITY | ETH          | ETH/DEC21 | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 0                    |

    Then the network moves ahead "1" epochs

    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 10     | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 10     | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC22 | buy  | 7      | 2002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC22 | sell | 7      | 2002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC22 | sell | 8      | 2003  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC22 | buy  | 8      | 2003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | sell | 2      | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | buy  | 2      | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | buy  | 3      | 1004  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | sell | 3      | 1004  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC22 | sell | 4      | 2003  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC22 | buy  | 4      | 2003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC22 | buy  | 6      | 2004  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC22 | sell | 6      | 2004  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    # looking at the returns of all parties for a window of 2 for only market ETH/DEC21
    # aux1 = [4,1] => variance = 4.5 => 1/4.5 = 0.2222222222
    # aux2 = [0,0] => N/A
    # party1 = [2,1] => variance = 0.5 => 1/0.5 = 2
    # party2 = [0] => N/A
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "1000" for asset "VEGA"
    And "party1" should have vesting account balance of "9000" for asset "VEGA"

  Scenario: multiple multiple (0056-REWA-090,0056-REWA-093,0056-REWA-094)
    # not that this test is also demonstrating multiple transfers to the same reward account but with different dispatch strategies, though same metric and same metric asset,
    # being handled sepearately, eventially contributing to the same accounts different amounts as calculated by the distribution strategy.
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement | ranks    |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RETURN_VOLATILITY | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 0                    |          |
      | 2  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca4ffffffffff | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RETURN_VOLATILITY | ETH          |         | 2           | 2             | RANK                  | INDIVIDUALS  | ALL              | 0                   | 0                    | 1:10,2:5 |

    Then the network moves ahead "1" epochs

    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 10     | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 10     | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC22 | buy  | 7      | 2002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC22 | sell | 7      | 2002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC22 | sell | 8      | 2003  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC22 | buy  | 8      | 2003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | sell | 2      | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | buy  | 2      | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | buy  | 3      | 1004  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | sell | 3      | 1004  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC22 | sell | 4      | 2003  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC22 | buy  | 4      | 2003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC22 | buy  | 6      | 2004  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC22 | sell | 6      | 2004  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    # looking at the returns of all parties for a window of 2 for only market ETH/DEC21

    # epoch1
    # market1:
    # aux1 return = 4
    # aux2 return = -6
    # party1 return = 2
    # market2:
    # aux1 return = 2.2857142857142857
    # aux2 return = -3.75
    # party1 return = 2

    # epoch2
    # market1:
    # aux1 return = 1
    # aux2 return = 0
    # party1 return = 1
    # party2 return = -1.4285714285714286
    # market2:
    # aux1 return = 1
    # aux2 return = 1
    # party1 return = 1
    # party2 return = -4

    # total:
    # aux1 = [6.2857142857142857, 2] => variance = 1/4.5918367346938775 = 0.2177777778
    # aux2 = [-9.75, 1] => 1/28.890625 = 0.03461330449
    # party1 = [4, 2] => variance = 1
    # party2 = [,-5.4285714285714286] => 0

    # pro rata rewards from transfer1:
    # aux1 = 10000 * 0.2177777778/1.2523910823 = 1738
    # aux2 = 10000 * 0.03461330449/1.2523910823 = 276
    # party1 = 10000 * 1/1.2523910823 = 7984

    # rank rewards from transfer2:
    # aux1 = 2500
    # aux2 = 2500
    # party1 = 5000

    # total
    # aux1 = 1738 + 2500 = 4238
    # aux2 = 276 + 2500 = 5276
    # party1 = 7984 + 5000 = 12984

    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca4ffffffffff" should have general account balance of "90000" for asset "VEGA"
    And "aux1" should have vesting account balance of "4238" for asset "VEGA"
    And "aux2" should have vesting account balance of "2776" for asset "VEGA"
    And "party1" should have vesting account balance of "12984" for asset "VEGA"


  Scenario: rank lottery (0056-REWA-190,0056-REWA-191)
    Given the following network parameters are set:
      | name                                         | value                                                                                            |
      | rewards.activityStreak.inactivityLimit       | 1                                                                                                |
      | rewards.activityStreak.minQuantumTradeVolume | 1000000000000000                                                                                 |
      | rewards.activityStreak.minQuantumOpenVolume  | 10000                                                                                            |
      | rewards.activityStreak.benefitTiers          | {"tiers": [{"minimum_activity_streak": 2, "reward_multiplier": "2", "vesting_multiplier": "2"}]} |

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement | ranks    |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca4ffffffffff | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RETURN_VOLATILITY | ETH          |         | 2           | 2             | RANK_LOTTERY          | INDIVIDUALS  | ALL              | 0                   | 0                    | 1:10,2:5 |

    Then the network moves ahead "1" epochs

    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 10     | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 10     | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC22 | buy  | 7      | 2002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC22 | sell | 7      | 2002  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC22 | sell | 8      | 2003  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC22 | buy  | 8      | 2003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | sell | 2      | 1003  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | buy  | 2      | 1003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | buy  | 3      | 1004  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | sell | 3      | 1004  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC22 | sell | 4      | 2003  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC22 | buy  | 4      | 2003  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC22 | buy  | 6      | 2004  | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC22 | sell | 6      | 2004  | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the network moves ahead "1" epochs

    # looking at the returns of all parties for a window of 2 for only market ETH/DEC21

    # epoch1
    # market1:
    # aux1 return = 4
    # aux2 return = -6
    # party1 return = 2
    # market2:
    # aux1 return = 2.2857142857142857
    # aux2 return = -3.75
    # party1 return = 2

    # epoch2
    # market1:
    # aux1 return = 1
    # aux2 return = 0
    # party1 return = 1
    # party2 return = -1.4285714285714286
    # market2:
    # aux1 return = 1
    # aux2 return = 1
    # party1 return = 1
    # party2 return = -4

    # total:
    # aux1 = [6.2857142857142857, 2] => variance = 1/4.5918367346938775 = 0.2177777778
    # aux2 = [-9.75, 1] => 1/28.890625 = 0.03461330449
    # party1 = [4, 2] => variance = 1
    # party2 = [,-5.4285714285714286] => 0

    # rank rewards from transfer1 (party1 has a multiplier of 2):
    # aux1 = 2500 (rank=5 * multiplier=1 = 5) => 1666
    # aux2 = 2500 (rank=5 * multiplier=1 = 5) => 1666
    # party1 = 5000 (rank=10 * multiplier=2 = 20) => 6666

    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca4ffffffffff" should have general account balance of "90000" for asset "VEGA"
    And "aux1" should have vesting account balance of "1666" for asset "VEGA"
    And "aux2" should have vesting account balance of "1666" for asset "VEGA"
    And "party1" should have vesting account balance of "6666" for asset "VEGA"
