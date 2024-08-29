Feature: Setting and applying referral reward multipliers

  Background:

    # Initialise timings
    Given time is updated to "2023-01-01T00:00:00Z"
    And the average block duration is "1"

    # Initialise network parameters
    Given the following network parameters are set:
      | name                                                    | value      |
      | market.fee.factors.infrastructureFee                    | 0.01       |
      | market.fee.factors.makerFee                             | 0.01       |
      | market.auction.minimumDuration                          | 1          |
      | limits.markets.maxPeggedOrders                          | 4          |
      | referralProgram.minStakedVegaTokens                     | 0          |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 1000000000 |
      | referralProgram.maxReferralRewardProportion             | 0.1        |

    # Initialise the staking parameters and the initial staking setup
    Given the following network parameters are set:
      | name                                       | value |
      | validators.delegation.minAmount            | 1     |
      | reward.staking.delegation.competitionLevel | 1.1   |
      | reward.staking.delegation.minValidators    | 1     |
    And the validators:
      | id    | staking account balance | pub_key |
      | node1 | 1000000                 | pk1     |
    And the parties submit the following delegations:
      | party | node id | amount |
      | pk1   | node1   | 10000  |

    # Initalise the referral program then move forwards an epoch to start the program
    Given the referral benefit tiers "rbt":
      | minimum running notional taker volume | minimum epochs | referral reward infra factor | referral reward maker factor | referral reward liquidity factor | referral discount infra factor | referral discount maker factor | referral discount liquidity factor |
      | 1                                     | 1              | 0.021                        | 0.022                        | 0.023                            | 0.024                          | 0.025                          | 0.026                              |
    And the referral staking tiers "rst":
      | minimum staked tokens | referral reward multiplier |
      | 2000                  | 2                          |
      | 3000                  | 10                         |
    And the referral program:
      | end of program       | window length | benefit tiers | staking tiers |
      | 2023-12-12T12:12:12Z | 7             | rbt           | rst           |
    And the network moves ahead "1" epochs

    # Initialse the assets and markets
    And the following assets are registered:
      | id      | decimal places | quantum |
      | USD.1.1 | 1              | 1       |
    And the markets:
      | id          | quote name | asset   | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USD.1.1 | ETH        | USD.1.1 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 1              | 1                       |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 3600s       | 1              |
    When the markets are updated:
      | id          | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/USD.1.1 | lqm-params           | 1e-3                   | 0                         |

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party     | asset   | amount     |
      | lpprov    | USD.1.1 | 1000000000 |
      | aux1      | USD.1.1 | 1000000000 |
      | aux2      | USD.1.1 | 1000000000 |
      | referrer1 | USD.1.1 | 1000000000 |
      | referee1  | USD.1.1 | 1000000000 |
      | referee2  | USD.1.1 | 1000000000 |

    # Exit the opening auction
    Given the parties submit the following liquidity provision:
      | id  | party  | market id   | commitment amount | fee  | lp type    |
      | lp1 | lpprov | ETH/USD.1.1 | 1000000           | 0.01 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id   | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/USD.1.1 | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/USD.1.1 | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
    When the parties place the following orders:
      | party | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD.1.1 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD.1.1 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD.1.1 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD.1.1 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USD.1.1"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD.1.1"

    # Create the referral set and codes
    Given the parties create the following referral codes:
      | party     | code            |
      | referrer1 | referral-code-1 |
    And the parties apply the following referral codes:
      | party    | code            |
      | referee1 | referral-code-1 |


  Scenario Outline: Referrer stakes a variable amount and receives a multiplier on their referral rewards (0083-RFPR-046)(0083-RFPR-047)(0029-FEES-023)(0029-FEES-025)
    # Expectation: Referral reward multiplier from staking should be set correctly according to the spec

    # Test cases
    # - Referrer does not fulfill the smallest 'minimum staking requirement', multiplier should default to 1
    # - Referrer does fulfill the smallest 'minimum staking requirement', multiplier should default to the correct value
    # - Referrer does fulfill the greatest 'minimum staking requirement', reward should be capped by the network parameter "referralProgram.maxReferralRewardProportion"

    Then the parties deposit on staking account the following amount:
      | party     | asset | amount           |
      | referrer1 | VEGA  | <staking amount> |
    Then the parties submit the following delegations:
      | party     | node id | amount           |
      | referrer1 | node1   | <staking amount> |

    Given the parties place the following orders:
      | party     | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1      | ETH/USD.1.1 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USD.1.1 | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" epochs
    Then the referral set stats for code "referral-code-1" at epoch "2" should have a running volume of 1000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.021               | 0.022               | 0.023                   | 0.024                 | 0.025                 | 0.026                     |

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 100    | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the following trades should be executed:
      | buyer | price | size | seller   |
      | aux1  | 1000  | 100  | referee1 |
    Then the following transfers should happen:
      | from     | to        | from account                             | to account                               | market id   | amount                     | asset   |
      | referee1 | market    | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_FEES_MAKER                  | ETH/USD.1.1 | <expected discounted fees> | USD.1.1 |
      | referee1 | market    | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_FEES_LIQUIDITY              | ETH/USD.1.1 | <expected discounted fees> | USD.1.1 |
      | referee1 |           | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE         |             | <expected discounted fees> | USD.1.1 |
      | referee1 |           | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_PENDING_FEE_REFERRAL_REWARD |             | <expected reward>          | USD.1.1 |
      |          | referrer1 | ACCOUNT_TYPE_PENDING_FEE_REFERRAL_REWARD | ACCOUNT_TYPE_GENERAL                     |             | <expected reward>          | USD.1.1 |

    Examples:
      | staking amount | expected discounted fees | expected reward |
      | 1001           | 96                       | 6               |
      | 2001           | 94                       | 12              |
      | 3001           | 89                       | 27              |