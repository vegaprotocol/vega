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
      | party3                                                           | ETH   | 100000    |
      | party4                                                           | ETH   | 100000    |

    And the parties deposit on staking account the following amount:
      | party   | asset | amount |
      | aux1    | VEGA  | 2000   |
      | aux2    | VEGA  | 2000   |
      | trader3 | VEGA  | 1500   |
      | trader4 | VEGA  | 1000   |
      | lpprov  | VEGA  | 10000  |
      | party1  | VEGA  | 800    |
      | party2  | VEGA  | 2000   |
      | party3  | VEGA  | 2000   |
      | party4  | VEGA  | 2000   |

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
    And the mark price should be "0" for the market "ETH/DEC21"

  Scenario: Given a recurring transfer is setup such that all eligible parties have a positive reward score, each parties metric is not offset and parties receive the correct rewards. (0056-REWA-116). Given the following dispatch metrics, if no `eligible keys` list is specified in the recurring transfer, all parties meeting other eligibility criteria should receive a score (0056-REWA-207).
    # setup recurring transfer to the reward account - this will start at the  end of this epoch
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RELATIVE_RETURN | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RELATIVE_RETURN | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 0                    |

    Then the network moves ahead "1" epochs
    And the mark price should be "1000" for the market "ETH/DEC21"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |
    And the mark price should be "1000" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 10     | 999   | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 10     | 999   | 1                | TYPE_LIMIT | TIF_GTC |           |
    And the mark price should be "1000" for the market "ETH/DEC21"

    Then the network moves ahead "1" epochs
    And the mark price should be "999" for the market "ETH/DEC21"

    # M2M
    # party1 = -30
    # aux1 = 20
    # aux2 = 10
    # party1 is not eligible because they don't have sufficient staking
    # relative return metric for aux1 = 20/5 = 4
    # relative return metric for aux2 = 10/5 = 2
    # aux1 gets 10000 * 4/6 = 6666
    # aux2 gets 10000 * 2/6 = 3333

    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "6666" for asset "VEGA"
    And "aux2" should have vesting account balance of "3333" for asset "VEGA"

    Then debug trades


  Scenario: Given the following dispatch metrics, if an `eligible keys` list is specified in the recurring transfer, only parties included in the list and meeting other eligibility criteria should receive a score (0056-REWA-217).
    # setup recurring transfer to the reward account - this will start at the  end of this epoch
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement | eligible_keys |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_RELATIVE_RETURN | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_RELATIVE_RETURN | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 0                    | aux1          |

    Then the network moves ahead "1" epochs
    And the mark price should be "1000" for the market "ETH/DEC21"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    Given time is updated to "2023-09-23T00:00:30Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 10     | 1002  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | aux1   | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |           |
    And the mark price should be "1000" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 10     | 999   | 0                | TYPE_LIMIT | TIF_GTC | p2-sell   |
      | aux2   | ETH/DEC21 | buy  | 10     | 999   | 1                | TYPE_LIMIT | TIF_GTC |           |
    And the mark price should be "1000" for the market "ETH/DEC21"

    Then the network moves ahead "1" epochs
    And the mark price should be "999" for the market "ETH/DEC21"

    # M2M
    # party1 = -30
    # aux1 = 20
    # aux2 = 10
    # party1 is not eligible because they don't have sufficient staking
    # relative return metric for aux1 = 20/5 = 4
    # relative return metric for aux2 = 10/5 = 2
    # aux1 gets 10000 * 4/6 = 6666
    # aux2 gets 10000 * 2/6 = 3333
    # but only aux1 is in the eligible keys so they get 10k

    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "990000" for asset "VEGA"
    And "aux1" should have vesting account balance of "10000" for asset "VEGA"

    Then debug trades


