Feature: Eligible parties metric rewards

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
            | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2dda | VEGA  | 1000000   |
            | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb | VEGA  | 1000000   |
            | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc | VEGA  | 1000000   |
            | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd | VEGA  | 1000000   |
            | aux1                                                             | ETH   | 100000000 |
            | aux2                                                             | ETH   | 100000000 |
            | trader3                                                          | ETH   | 10000     |
            | trader4                                                          | ETH   | 10000     |


        And the parties deposit on staking account the following amount:
            | party   | asset | amount |
            | aux1    | VEGA  | 2000   |
            | aux2    | VEGA  | 1000   |
            | trader3 | VEGA  | 1500   |
            | trader4 | VEGA  | 1000   |

        Given time is updated to "2023-09-23T00:00:00Z"
        Given the average block duration is "1"

        # Initalise the referral program then move forwards an epoch to start the program
        Given the referral benefit tiers "rbt":
            | minimum running notional taker volume | minimum epochs | referral reward infra factor | referral reward maker factor | referral reward liquidity factor | referral discount infra factor | referral discount maker factor | referral discount liquidity factor |
            | 500                                   | 1              | 0.021                        | 0.022                        | 0.023                            | 0.024                          | 0.025                          | 0.026                              |
            | 1000                                  | 1              | 0.21                         | 0.22                         | 0.23                             | 0.24                           | 0.25                           | 0.26                               |
        And the referral staking tiers "rst":
            | minimum staked tokens | referral reward multiplier |
            | 1                     | 1                          |
        And the referral program:
            | end of program       | window length | benefit tiers | staking tiers |
            | 2023-12-12T12:12:12Z | 7             | rbt           | rst           |

        #complete the epoch to advance to a meaningful epoch (can't setup transfer to start at epoch 0)
        Then the network moves ahead "1" epochs

    Scenario: Given a recurring transfer using the eligible entities metric and specifying only a staking requirement. If an entity meets the staking requirement they will receive rewards. (0056-REWA-182). Given a recurring transfer using the eligible entities metric and specifying only a staking requirement. If an entity does not meet the staking requirement they will receive no rewards. (0056-REWA-183).Given a recurring transfer using the eligible entities metric and specifying only a position requirement (assume all markets within scope). If an entity meets the position requirement they will receive rewards. (0056-REWA-184). Given a recurring transfer using the eligible entities metric and specifying only a position requirement (assume all markets within scope). If an entity does not meet the position requirement they will receive no rewards. (0056-REWA-185).Given a recurring transfer using the eligible entities metric and specifying both a staking and position requirement. If an entity meets neither the staking or position requirement, they will receive no rewards. (0056-REWA-186).Given a recurring transfer using the eligible entities metric and specifying both a staking and position requirement. If an entity meets the staking but not the position requirement, they will receive no rewards. (0056-REWA-187).Given a recurring transfer using the eligible entities metric and specifying both a staking and position requirement. If an entity meets the position requirement but not the staking requirement, they will receive no rewards. (0056-REWA-188).Given a recurring transfer using the eligible entities metric and specifying both a staking and position requirement. If an entity meets both the staking and position requirement, they will receive rewards. (0056-REWA-189).
        # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
        Given the parties submit the following recurring transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
            | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2dda | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES |              |         | 2           | 1             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 0                    |
            | 2  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES |              |         | 2           | 1             | PRO_RATA              | INDIVIDUALS  | ALL              | 1200                | 0                    |
            | 3  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES | ETH          |         | 2           | 1             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 10000                |
            | 4  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES | ETH          |         | 2           | 1             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 10000                |

        Then the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

        Then the opening auction period ends for market "ETH/DEC21"
        And the market data for the market "ETH/DEC21" should be:
            | mark price | trading mode            |
            | 1000       | TRADING_MODE_CONTINUOUS |

        When the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader3 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

        Then the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader4 | ETH/DEC21 | sell | 4      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

        Then the network moves ahead "1" epochs
        # no requirement so surely distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2dda" should have general account balance of "990000" for asset "VEGA"
        # only staking requirement so distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb" should have general account balance of "990000" for asset "VEGA"
        # not distributed as the notional requirement not met
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc" should have general account balance of "1000000" for asset "VEGA"
        # not distributed as the notional requirement not met
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd" should have general account balance of "1000000" for asset "VEGA"

        # they get 1/8 of the reward with no requirements + 1/2 of the reward with staking minimum = 1250+5000=6250
        And "aux1" should have vesting account balance of "6250" for asset "VEGA"
        # they get 1/8 of the reward with no requirements = 1250
        And "aux2" should have vesting account balance of "1250" for asset "VEGA"
        # they get 1/8 of the reward with no requirements + 1/2 of the reward with staking minimum = 1250+5000=6250
        And "trader3" should have vesting account balance of "6250" for asset "VEGA"
        # they get 1/8 of the reward with no requirements =  1250
        And "trader4" should have vesting account balance of "1250" for asset "VEGA"

        # now lets get some notional so we can satisfy the notional requirement
        When the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader3 | ETH/DEC21 | buy  | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |
            | trader4 | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

        Then the network moves ahead "1" epochs
        # no requirement so surely distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2dda" should have general account balance of "980000" for asset "VEGA"
        # only staking requirement so distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb" should have general account balance of "980000" for asset "VEGA"
        # we have trade3 statisfying the notional requirement so distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc" should have general account balance of "990000" for asset "VEGA"
        # we have trade3 statisfying the notional requirement so distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd" should have general account balance of "990000" for asset "VEGA"

        # for reward (1) each gets a 1/8
        # for reward (2) aux1 and trader3 split the reward
        # for reward (3) each gets a quarter (all have sufficient notional)
        # for reward (4) each gets a quarter
        # aux1 = 6250 + 1250 + 5000 + 2500 + 2500 = 17500
        And "aux1" should have vesting account balance of "17500" for asset "VEGA"
        # aux2 = 1250 + 1250 + 2500 + 2500 = 7500
        And "aux2" should have vesting account balance of "7500" for asset "VEGA"
        # trader3 = 6250 + 1250 + 5000 + 2500 + 2500 = 17500
        And "trader3" should have vesting account balance of "17500" for asset "VEGA"
        # trader4 = 1250 + 1250 + 2500 + 2500 = 7500
        And "trader4" should have vesting account balance of "7500" for asset "VEGA"

    Scenario: Given a recurring transfer using the eligible entries metric and scoping individuals. If multiple parties meet all eligibility they should receive rewards proportional to any reward multipliers. (0056-REWA-178)
        Given the following network parameters are set:
            | name                                         | value                                                                                            |
            | network.markPriceUpdateMaximumFrequency      | 0s                                                                                               |
            | market.auction.minimumDuration               | 1                                                                                                |
            | validators.epoch.length                      | 20s                                                                                              |
            | limits.markets.maxPeggedOrders               | 4                                                                                                |
            | rewards.activityStreak.inactivityLimit       | 1                                                                                                |
            | rewards.activityStreak.minQuantumTradeVolume | 1000000000000000                                                                                 |
            | rewards.activityStreak.minQuantumOpenVolume  | 10000                                                                                            |
            | rewards.activityStreak.benefitTiers          | {"tiers": [{"minimum_activity_streak": 2, "reward_multiplier": "2", "vesting_multiplier": "2"}]} |

        Then the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

        Then the opening auction period ends for market "ETH/DEC21"
        And the market data for the market "ETH/DEC21" should be:
            | mark price | trading mode            |
            | 1000       | TRADING_MODE_CONTINUOUS |

        Given the current epoch is "1"
        And the parties place the following orders:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | aux1    | ETH/DEC21 | buy  | 11     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | trader3 | ETH/DEC21 | sell | 11     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

        When the network moves ahead "2" epochs
        Then the activity streaks at epoch "2" should be:
            | party   | active for | inactive for | reward multiplier | vesting multiplier |
            | trader3 | 2          | 0            | 2                 | 2                  |
            | aux1    | 2          | 0            | 2                 | 2                  |

        # now we know that trader3 has >1 multipliers so lets set up a reward
        Given the parties submit the following recurring transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
            | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2dda | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 3           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES |              |         | 2           | 1             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 0                    |

        Then the network moves ahead "1" epochs
        # no requirement so surely distributed
        # trader3 and aux1 have multipliers of 2
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2dda" should have general account balance of "990000" for asset "VEGA"
        And "aux1" should have vesting account balance of "2000" for asset "VEGA"
        And "aux2" should have vesting account balance of "1000" for asset "VEGA"
        And "trader3" should have vesting account balance of "2000" for asset "VEGA"
        And "trader4" should have vesting account balance of "1000" for asset "VEGA"
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2dda" should have vesting account balance of "1000" for asset "VEGA"
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb" should have vesting account balance of "1000" for asset "VEGA"
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc" should have vesting account balance of "1000" for asset "VEGA"
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd" should have vesting account balance of "1000" for asset "VEGA"

    Scenario: Given a recurring transfer using the eligible entities metric and a reward window length N greater than one, a party who met the eligibility requirements in the current epoch as well as the previous N-1 epochs will receive rewards at the end of the epoch. (0056-REWA-179)
        # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
        Given the parties submit the following recurring transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
            | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2dda | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES |              |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 0                    |
            | 2  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES |              |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1200                | 0                    |
            | 3  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 10000                |
            | 4  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 10000                |

        Then the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

        Then the opening auction period ends for market "ETH/DEC21"
        And the market data for the market "ETH/DEC21" should be:
            | mark price | trading mode            |
            | 1000       | TRADING_MODE_CONTINUOUS |

        When the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader3 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

        Then the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader4 | ETH/DEC21 | sell | 4      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

        Then the network moves ahead "1" epochs
        # no requirement so surely distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2dda" should have general account balance of "990000" for asset "VEGA"
        # only staking requirement so distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb" should have general account balance of "990000" for asset "VEGA"
        # not distributed as the notional requirement not met
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc" should have general account balance of "1000000" for asset "VEGA"
        # not distributed as the notional requirement not met
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd" should have general account balance of "1000000" for asset "VEGA"

        # they get 1/8 of the reward with no requirements + 1/2 of the reward with staking minimum = 1250+5000=6250
        And "aux1" should have vesting account balance of "6250" for asset "VEGA"
        # they get 1/8 of the reward with no requirements = 1250
        And "aux2" should have vesting account balance of "1250" for asset "VEGA"
        # they get 1/8 of the reward with no requirements + 1/2 of the reward with staking minimum = 1250+5000=6250
        And "trader3" should have vesting account balance of "6250" for asset "VEGA"
        # they get 1/8 of the reward with no requirements =  1250
        And "trader4" should have vesting account balance of "1250" for asset "VEGA"

        # now lets get some notional so we can satisfy the notional requirement
        When the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader3 | ETH/DEC21 | buy  | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |
            | trader4 | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

        Then the network moves ahead "1" epochs
        # no requirement so surely distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2dda" should have general account balance of "980000" for asset "VEGA"
        # only staking requirement so distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb" should have general account balance of "980000" for asset "VEGA"
        # we have trade3 statisfying the notional requirement so distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc" should have general account balance of "990000" for asset "VEGA"
        # we have trade3 statisfying the notional requirement so distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd" should have general account balance of "990000" for asset "VEGA"

        # for reward (1) each gets a 1/8
        # for reward (2) aux1 and trader3 split the reward
        # for reward (3) each gets a quarter (all have sufficient notional)
        # for reward (4) each gets a quarter
        # aux1 = 6250 + 1250 + 5000 + 2500 + 2500 = 17500
        And "aux1" should have vesting account balance of "17500" for asset "VEGA"
        # aux2 = 1250 + 1250 + 2500 + 2500 = 7500
        And "aux2" should have vesting account balance of "7500" for asset "VEGA"
        # trader3 = 6250 + 1250 + 5000 + 2500 + 2500 = 17500
        And "trader3" should have vesting account balance of "17500" for asset "VEGA"
        # trader4 = 1250 + 1250 + 2500 + 2500 = 7500
        And "trader4" should have vesting account balance of "7500" for asset "VEGA"

    Scenario: Given a recurring transfer using the eligible entities metric and a reward window length N greater than one, a party who met the eligibility requirements in the current epoch only will receive no rewards at the end of the epoch.
        # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
        Given the parties submit the following recurring transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
            | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES |              |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1200                | 0                    |
            | 2  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 10000                |
            | 3  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 10000                |

        Then the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

        Then the opening auction period ends for market "ETH/DEC21"
        And the market data for the market "ETH/DEC21" should be:
            | mark price | trading mode            |
            | 1000       | TRADING_MODE_CONTINUOUS |

        When the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader3 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

        Then the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader4 | ETH/DEC21 | sell | 4      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

        Then the network moves ahead "1" epochs
        # only staking requirement so distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb" should have general account balance of "990000" for asset "VEGA"
        # not distributed as the notional requirement not met
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc" should have general account balance of "1000000" for asset "VEGA"
        # not distributed as the notional requirement not met
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd" should have general account balance of "1000000" for asset "VEGA"

        # they get 1/2 of the reward with staking minimum = 5000
        And "aux1" should have vesting account balance of "5000" for asset "VEGA"
        # they get 1/2 of the reward with staking minimum = 5000
        And "trader3" should have vesting account balance of "5000" for asset "VEGA"

        # now lets get some notional so we can satisfy the notional requirement
        When the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader3 | ETH/DEC21 | buy  | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |
            | trader4 | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

        # now lets make aux2 and trader 4 satisfy the staking requirement
        And the parties deposit on staking account the following amount:
            | party   | asset | amount |
            | aux2    | VEGA  | 1200   |
            | trader4 | VEGA  | 1200   |

        Then the network moves ahead "1" epochs
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb" should have general account balance of "980000" for asset "VEGA"
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc" should have general account balance of "990000" for asset "VEGA"
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd" should have general account balance of "990000" for asset "VEGA"

        # for reward (1) aux1 and trader3 split the reward (because they satisfied the requirement for both epochs)
        # for reward (2) split 4 ways - all have notional, no staking requirement
        # for reward (3) split 4 ways - all have notional, lower staking requirement met in both epochs
        And "aux1" should have vesting account balance of "15000" for asset "VEGA"
        And "aux2" should have vesting account balance of "5000" for asset "VEGA"
        And "trader3" should have vesting account balance of "15000" for asset "VEGA"
        And "trader4" should have vesting account balance of "5000" for asset "VEGA"

        Then the network moves ahead "1" epochs
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb" should have general account balance of "970000" for asset "VEGA"
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc" should have general account balance of "980000" for asset "VEGA"
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd" should have general account balance of "980000" for asset "VEGA"

        # all 3 rewards are now split 4 ways
        # aux1 = 15000 + 2500 + 2500 + 2500 = 22500
        And "aux1" should have vesting account balance of "22500" for asset "VEGA"
        # aux2 = 5000 + 2500 + 2500 + 2500 = 12500
        And "aux2" should have vesting account balance of "12500" for asset "VEGA"
        # trader3 = 15000 + 2500 + 2500 + 2500 = 22500
        And "trader3" should have vesting account balance of "22500" for asset "VEGA"
        # trader4 = 5000 + 2500 + 2500 + 2500 = 7500
        And "trader4" should have vesting account balance of "12500" for asset "VEGA"

    Scenario: Given a recurring transfer using the eligible entities metric and a reward window length greater than one, a party who met the eligibility requirements in a previous epoch in the window, but not the current epoch will receive no rewards at the end of the epoch. (0056-REWA-181)
        # setup recurring transfer to the reward account - this will start at the end of this epoch (1)
        Given the parties submit the following recurring transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type                       | asset | amount | start_epoch | end_epoch | factor | metric                            | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement |
            | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES |              |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1200                | 0                    |
            | 2  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 0                   | 10000                |
            | 3  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_ELIGIBLE_ENTITIES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_ELIGIBLE_ENTITIES | ETH          |         | 2           | 2             | PRO_RATA              | INDIVIDUALS  | ALL              | 1000                | 10000                |

        Then the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

        Then the opening auction period ends for market "ETH/DEC21"
        And the market data for the market "ETH/DEC21" should be:
            | mark price | trading mode            |
            | 1000       | TRADING_MODE_CONTINUOUS |

        When the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader3 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

        Then the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader4 | ETH/DEC21 | sell | 4      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

        Then the network moves ahead "1" epochs
        # only staking requirement so distributed
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb" should have general account balance of "990000" for asset "VEGA"
        # not distributed as the notional requirement not met
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc" should have general account balance of "1000000" for asset "VEGA"
        # not distributed as the notional requirement not met
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd" should have general account balance of "1000000" for asset "VEGA"

        # they get 1/2 of the reward with staking minimum = 5000
        And "aux1" should have vesting account balance of "5000" for asset "VEGA"
        # they get 1/2 of the reward with staking minimum = 5000
        And "trader3" should have vesting account balance of "5000" for asset "VEGA"

        # now lets get some notional so we can satisfy the notional requirement
        When the parties place the following orders with ticks:
            | party   | market id | side | volume | price | resulting trades | type       | tif     |
            | trader3 | ETH/DEC21 | buy  | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |
            | trader4 | ETH/DEC21 | sell | 10     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

        # lets withdraw the staking to make them ineligible in the following window
        And the parties withdraw from staking account the following amount:
            | party   | asset | amount |
            | aux1    | VEGA  | 1200   |
            | aux2    | VEGA  | 800    |
            | trader4 | VEGA  | 1000   |

        Then the network moves ahead "1" epochs
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddb" should have general account balance of "980000" for asset "VEGA"
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddc" should have general account balance of "990000" for asset "VEGA"
        And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddd" should have general account balance of "990000" for asset "VEGA"

        # for reward (1) is paid only to trader 3 
        # for reward (2) split 4 ways - all have notional, no staking requirement
        # for reward (3) is paid only to trader 3 
        And "aux1" should have vesting account balance of "7500" for asset "VEGA"
        And "aux2" should have vesting account balance of "2500" for asset "VEGA"
        And "trader3" should have vesting account balance of "27500" for asset "VEGA"
        And "trader4" should have vesting account balance of "2500" for asset "VEGA"
