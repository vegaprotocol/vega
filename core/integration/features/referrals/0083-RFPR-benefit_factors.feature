Feature: Setting and applying referee benefit factors

  Background:

    # Initialise timings
    Given time is updated to "2023-01-01T00:00:00Z"
    And the average block duration is "1"
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    And the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 3                 |

    # Initialise the markets and network parameters
    Given the following network parameters are set:
      | name                                                    | value      |
      | market.fee.factors.infrastructureFee                    | 0.01       |
      | market.fee.factors.makerFee                             | 0.01       |
      | market.auction.minimumDuration                          | 1          |
      | limits.markets.maxPeggedOrders                          | 4          |
      | referralProgram.minStakedVegaTokens                     | 0          |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 1000000000 |
      | referralProgram.maxReferralRewardProportion             | 0.1        |

    # Initalise the referral program then move forwards an epoch to start the program
    Given the referral benefit tiers "rbt":
      | minimum running notional taker volume | minimum epochs | referral reward infra factor | referral reward maker factor | referral reward liquidity factor | referral discount infra factor | referral discount maker factor | referral discount liquidity factor |
      | 2000                                  | 2              | 0.021                        | 0.022                        | 0.023                            | 0.024                          | 0.025                          | 0.026                              |
      | 3000                                  | 3              | 0.21                         | 0.22                         | 0.23                             | 0.24                           | 0.25                           | 0.26                               |
    And the referral staking tiers "rst":
      | minimum staked tokens | referral reward multiplier |
      | 1                     | 1                          |
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
      | ETH/USD.1.2 | ETH        | USD.1.1 | log-normal-risk-model         | margin-calculator-1       | 1                | default-none | price-monitoring | default-eth-for-future | 1e-3                   | 0                         | default-futures | 1              | 1                       |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 3600s       | 1              |
    When the markets are updated:
      | id          | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/USD.1.1 | lqm-params           | 1e-3                   | 0                         |
      | ETH/USD.1.2 | lqm-params           | 1e-3                   | 0                         |

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party     | asset   | amount     |
      | lpprov    | USD.1.1 | 1000000000 |
      | lpprov2   | USD.1.1 | 1000000000 |
      | aux1      | USD.1.1 | 1000000000 |
      | aux2      | USD.1.1 | 1000000000 |
      | aux3      | USD.1.1 | 1000000000 |
      | aux4      | USD.1.1 | 1000000000 |
      | referrer1 | USD.1.1 | 1000000000 |
      | referee1  | USD.1.1 | 1000000000 |
      | referee2  | USD.1.1 | 1000000000 |
      | ptbuy     | USD.1.1 | 1000000000 |
      | ptsell    | USD.1.1 | 1000000000 |

    # Exit the opening auction
    Given the parties submit the following liquidity provision:
      | id  | party   | market id   | commitment amount | fee  | lp type    |
      | lp1 | lpprov  | ETH/USD.1.1 | 1000000           | 0.01 | submission |
      | lp2 | lpprov2 | ETH/USD.1.2 | 1000000           | 0.01 | submission |
    And the parties place the following pegged iceberg orders:
      | party   | market id   | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov  | ETH/USD.1.1 | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov  | ETH/USD.1.1 | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
      | lpprov2 | ETH/USD.1.2 | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov2 | ETH/USD.1.2 | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
    When the parties place the following orders:
      | party | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD.1.1 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD.1.1 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD.1.1 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD.1.1 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/USD.1.2 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/USD.1.2 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux4  | ETH/USD.1.2 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux4  | ETH/USD.1.2 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USD.1.1"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD.1.1"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD.1.2"

    # Create the referral set and codes
    Given the parties create the following referral codes:
      | party     | code            | is_team | team  |
      | referrer1 | referral-code-1 | true    | team1 |
    And the parties apply the following referral codes:
      | party    | code            | is_team | team  |
      | referee1 | referral-code-1 | true    | team1 |

    And the team "team1" has the following members:
      | party     |
      | referrer1 |
      | referee1  |

  Scenario Outline: Referral set generates variable running taker volume with a referee with variable epochs in set (0083-RFPR-036)(0083-RFPR-037)(0083-RFPR-038)(0083-RFPR-039)(0083-RFPR-040)
    # Expectation: Referral reward factor and referral discount factor should be set correctly according to the spec

    # Test cases
    # - Referral set does not fulfill the smallest 'minimimum running taker volume' requirement and referee does not fulfill the smallest 'minimum epochs' requirement
    # - Referral set does not fulfill the smallest 'minimimum running taker volume' requirement but referee fulfulls the smallest 'minimum epochs' requirement
    # - Referral set fulfills the smallest 'minimimum running taker volume' requirement but referee does fulfill the smallest 'minimum epochs' requirement
    # - Referral set fulfills the 'minimum running taker volume' requirement tier for a higher tier than a referee fulfulls the 'minimum epochs' requirement for
    # - Referral set fulfills the 'minimum running taker volume' requirement tier for a lower tier than a referee fulfulls the 'minimum epochs' requirement for
    # - Referral set fulfills the 'minimum running taker volume' requirement tier for the same tier as a referee fulfulls the 'minimum epochs' requirement for

    Given the parties place the following orders:
      | party    | market id   | side | volume           | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | <initial volume> | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | <initial volume> | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead <time in set> epochs
    Then the referral set stats for code "referral-code-1" at epoch <time in set> should have a running volume of <running volume>:
      | party    | reward infra factor       | reward maker factor       | reward liquidity factor       | discount infra factor       | discount maker factor       | discount liquidity factor       |
      | referee1 | <max infra reward factor> | <max maker reward factor> | <max liquidity reward factor> | <max infra discount factor> | <max maker discount factor> | <max liquidity discount factor> |

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    Examples:
      | initial volume | running volume | time in set | max infra reward factor | max maker reward factor | max liquidity reward factor | max infra discount factor | max maker discount factor | max liquidity discount factor |
      | 10             | 1000           | "1"         | 0                       | 0                       | 0                           | 0                         | 0                         | 0                             |
      | 10             | 1000           | "4"         | 0                       | 0                       | 0                           | 0                         | 0                         | 0                             |
      | 20             | 2000           | "1"         | 0.021                   | 0.022                   | 0.023                       | 0                         | 0                         | 0                             |
      | 30             | 3000           | "2"         | 0.21                    | 0.22                    | 0.23                        | 0.024                     | 0.025                     | 0.026                         |
      | 20             | 2000           | "3"         | 0.021                   | 0.022                   | 0.023                       | 0.024                     | 0.025                     | 0.026                         |
      | 30             | 3000           | "3"         | 0.21                    | 0.22                    | 0.23                        | 0.24                      | 0.25                      | 0.26                          |


  Scenario: Referee incurs fees during continuous trading (0029-FEES-023)(0029-FEES-025)
    # Expectation: referral discount applied and referral reward calculated from resulting fee

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" epochs
    Then the referral set stats for code "referral-code-1" at epoch "2" should have a running volume of 2000:
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
      | from     | to        | from account                             | to account                               | market id   | amount | asset   |
      | referee1 | market    | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_FEES_MAKER                  | ETH/USD.1.1 | 96     | USD.1.1 |
      | referee1 | market    | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_FEES_LIQUIDITY              | ETH/USD.1.1 | 96     | USD.1.1 |
      | referee1 |           | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE         |             | 96     | USD.1.1 |
      | referee1 |           | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_PENDING_FEE_REFERRAL_REWARD |             | 6      | USD.1.1 |
      |          | referrer1 | ACCOUNT_TYPE_PENDING_FEE_REFERRAL_REWARD | ACCOUNT_TYPE_GENERAL                     |             | 6      | USD.1.1 |


  Scenario: Referrer incurs fees during continuous trading (0029-FEES-023)(0029-FEES-025)
    # Expectation: referral discount should not be applied and no referral rewards should be distributed

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" epochs
    Then the referral set stats for code "referral-code-1" at epoch "2" should have a running volume of 2000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.021               | 0.022               | 0.023                   | 0.024                 | 0.025                 | 0.026                     |

    Given the parties place the following orders:
      | party     | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1      | ETH/USD.1.1 | buy  | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USD.1.1 | sell | 100    | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the following trades should be executed:
      | buyer | price | size | seller    |
      | aux1  | 1000  | 100  | referrer1 |
    Then the following transfers should happen:
      | from      | to     | from account         | to account                       | market id   | amount | asset   |
      | referrer1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_MAKER          | ETH/USD.1.1 | 100    | USD.1.1 |
      | referrer1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/USD.1.1 | 100    | USD.1.1 |
      | referrer1 |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |             | 100    | USD.1.1 |


  Scenario: Referee incurs fees when exiting an auction (0029-FEES-024)(0029-FEES-026)
    # Expectation: fee should be split between buyer and seller, referral discount applied and referral reward calculated from resulting fee

    Given the parties place the following orders:
      | party     | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1      | ETH/USD.1.1 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USD.1.1 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" epochs
    Then the referral set stats for code "referral-code-1" at epoch "2" should have a running volume of 2000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.021               | 0.022               | 0.023                   | 0.024                 | 0.025                 | 0.026                     |

    # Cancel the liquidity commitment triggering an auction
    Given the parties submit the following liquidity provision:
      | id  | party  | market id   | commitment amount | fee | lp type      |
      | lp1 | lpprov | ETH/USD.1.1 | 0                 | 0.1 | cancellation |
    And the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD.1.1"
    When the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 200    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 200    | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer | price | size | seller   |
      | aux1  | 1000  | 200  | referee1 |
    And the parties submit the following liquidity provision:
      | id  | party  | market id   | commitment amount | fee  | lp type    |
      | lp2 | lpprov | ETH/USD.1.1 | 1000000           | 0.01 | submission |
    And the network moves ahead "1" epochs
    # fees are split between referee1 and aux1
    # aux1 has no discounts or rewards so is paying 100, 100
    # referee1 has a discount of 1 and pays a reward of 2 to the referrer
    And the following transfers should happen:
      | from     | to        | from account                             | to account                               | market id   | amount | asset   |
      | referee1 | market    | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_FEES_LIQUIDITY              | ETH/USD.1.1 | 0      | USD.1.1 |
      | referee1 |           | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE         |             | 192    | USD.1.1 |
      | referee1 |           | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_PENDING_FEE_REFERRAL_REWARD |             | 8      | USD.1.1 |
      |          | referrer1 | ACCOUNT_TYPE_PENDING_FEE_REFERRAL_REWARD | ACCOUNT_TYPE_GENERAL                     |             | 8      | USD.1.1 |


  Scenario: Referrer incurs fees when exiting an auction (0029-FEES-024)(0029-FEES-026)
    # Expectation: fee should be split between buyer and seller, referral discount applied and referral reward calculated from resulting fee

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux3     | ETH/USD.1.2 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.2 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" epochs
    Then the referral set stats for code "referral-code-1" at epoch "2" should have a running volume of 2000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.021               | 0.022               | 0.023                   | 0.024                 | 0.025                 | 0.026                     |
    And the market data for the market "ETH/USD.1.2" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | horizon | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS | 7469         | 1000000        | 21            | 3600    | 973       | 1027      |

    # Submit ourders out of price range to trigger auction
    When the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     |
      | ptbuy  | ETH/USD.1.2 | buy  | 2      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
      | ptsell | ETH/USD.1.2 | sell | 2      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/USD.1.2" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 7935         | 1000000        | 21            | 3           |
    When the parties place the following orders:
      | party     | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux3      | ETH/USD.1.2 | buy  | 200    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USD.1.2 | sell | 200    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" epochs
    And the network moves ahead "3" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD.1.2"
    And the following trades should be executed:
      | buyer | price | size | seller    |
      | aux3  | 1000  | 198  | referrer1 |
      | aux3  | 1000  | 2    | ptsell    |
    And the following transfers should happen:
      | from      | to     | from account         | to account                       | market id   | amount | asset   |
      | referrer1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/USD.1.2 | 99     | USD.1.1 |
      | referrer1 |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |             | 99     | USD.1.1 |


  Scenario: Insufficent fees after discounts applied to pay a referral commision to the referrer (0029-FEES-029)
    # Expectation: rewards (<1) should be floored and therefore no reward paid

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" epochs
    Then the referral set stats for code "referral-code-1" at epoch "2" should have a running volume of 2000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.021               | 0.022               | 0.023                   | 0.024                 | 0.025                 | 0.026                     |

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 50     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the following trades should be executed:
      | buyer | price | size | seller   |
      | aux1  | 1000  | 50   | referee1 |
    Then the following transfers should happen:
      | from     | to     | from account         | to account                       | market id   | amount | asset   |
      | referee1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_MAKER          | ETH/USD.1.1 | 48     | USD.1.1 |
      | referee1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/USD.1.1 | 48     | USD.1.1 |
      | referee1 |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |             | 48     | USD.1.1 |


  Scenario: Insufficent fees to be able to apply a referral discount discount for the referee (0029-FEES-030)
    # Expectation: discounts (<1) should be floored and therefore no discount applied

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" epochs
    Then the referral set stats for code "referral-code-1" at epoch "2" should have a running volume of 2000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.021               | 0.022               | 0.023                   | 0.024                 | 0.025                 | 0.026                     |

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 40     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the following trades should be executed:
      | buyer | price | size | seller   |
      | aux1  | 1000  | 40   | referee1 |
    Then the following transfers should happen:
      | from     | to     | from account         | to account                       | market id   | amount | asset   |
      | referee1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_MAKER          | ETH/USD.1.1 | 39     | USD.1.1 |
      | referee1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/USD.1.1 | 39     | USD.1.1 |
      | referee1 |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |             | 40     | USD.1.1 |


  Scenario: Referal reward factor set greater than referralProgram.maxReferralRewardProportion (0029-FEES-029, 0029-FEES-031)
    # Expectation: the maximum reward proportion should be adhered to

    Given the parties place the following orders:
      | party     | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1      | ETH/USD.1.1 | buy  | 30     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USD.1.1 | sell | 30     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "3" epochs
    Then the referral set stats for code "referral-code-1" at epoch "3" should have a running volume of 3000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.21                | 0.22                | 0.23                    | 0.24                  | 0.25                  | 0.26                      |

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 100    | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the following trades should be executed:
      | buyer | price | size | seller   |
      | aux1  | 1000  | 100  | referee1 |

    # reward factor is 0.21, multiplier is 1 however the maxReferralRewardProportion is set to 0.1 therefore
    # the actual reward given is 0.1 * 240 = 24
    Then the following transfers should happen:
      | from     | to        | from account                             | to account                               | market id   | amount | asset   |
      | referee1 | market    | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_FEES_MAKER                  | ETH/USD.1.1 | 68     | USD.1.1 |
      | referee1 | market    | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_FEES_LIQUIDITY              | ETH/USD.1.1 | 67     | USD.1.1 |
      | referee1 |           | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE         |             | 69     | USD.1.1 |
      | referee1 |           | ACCOUNT_TYPE_GENERAL                     | ACCOUNT_TYPE_PENDING_FEE_REFERRAL_REWARD |             | 21     | USD.1.1 |
      |          | referrer1 | ACCOUNT_TYPE_PENDING_FEE_REFERRAL_REWARD | ACCOUNT_TYPE_GENERAL                     |             | 21     | USD.1.1 |
