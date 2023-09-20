Feature: Evaluating trader activity

  Background:

    # Initialise the network
    Given time is updated to "2023-01-01T00:00:00Z"
    And the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.auction.minimumDuration          | 1     |
      | validators.epoch.length                 | 20s   |
      | limits.markets.maxPeggedOrders          | 4     |

    And the following network parameters are set:
      | name                                         | value                                                                                            |
      | rewards.activityStreak.inactivityLimit       | 1                                                                                                |
      | rewards.activityStreak.minQuantumOpenVolume  | 1                                                                                                |
      | rewards.activityStreak.minQuantumTradeVolume | 1                                                                                                |
      | rewards.activityStreak.benefitTiers          | {"tiers": [{"minimum_activity_streak": 1, "reward_multiplier": "2", "vesting_multiplier": "2"}]} |

    # Initialise the markets
    And the following assets are registered:
      | id       | decimal places | quantum |
      | USD.0.1  | 0              | 1       |
      | USD.1.10 | 1              | 10      |
    And the markets:
      | id           | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USD.0.1  | ETH        | USD.0.1  | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
      | ETH/USD.1.10 | ETH        | USD.1.10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 1              | 1                       |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 3600s       | 1              |
    When the markets are updated:
      | id           | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/USD.0.1  | lqm-params           | 1e-3                   | 0                         |
      | ETH/USD.1.10 | lqm-params           | 1e-3                   | 0                         |
    Then the network moves ahead "1" blocks

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party   | asset    | amount      |
      | lpprov  | USD.1.10 | 10000000000 |
      | aux1    | USD.1.10 | 10000000    |
      | aux2    | USD.1.10 | 10000000    |
      | trader1 | USD.1.10 | 10000000    |
      | lpprov  | USD.0.1  | 10000000000 |
      | aux1    | USD.0.1  | 10000000    |
      | aux2    | USD.0.1  | 10000000    |
      | trader1 | USD.0.1  | 10000000    |

    # Exit opening auctions
    Given the parties submit the following liquidity provision:
      | id  | party  | market id    | commitment amount | fee  | lp type    |
      | lp1 | lpprov | ETH/USD.0.1  | 1000000           | 0.01 | submission |
      | lp2 | lpprov | ETH/USD.1.10 | 10000000          | 0.01 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id    | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/USD.0.1  | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/USD.0.1  | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
      | lpprov | ETH/USD.1.10 | 50000     | 10000                | buy  | BID              | 100000 | 10     |
      | lpprov | ETH/USD.1.10 | 50000     | 10000                | sell | ASK              | 100000 | 10     |
    When the parties place the following orders:
      | party | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD.0.1  | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD.0.1  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD.0.1  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD.0.1  | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD.1.10 | buy  | 1      | 9900  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD.1.10 | buy  | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD.1.10 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD.1.10 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USD.0.1"
    And the opening auction period ends for market "ETH/USD.1.10"
    And the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD.0.1"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD.1.10"


  Scenario Outline: Party trades during continuous trading (0086-ASPR-004)(0086-ASPR-006)
    # Expectation: traders activity streak should be incremented if they fulfill the trade volume requirement

    # Test Cases:
    # - single trade (as taker) in continuous trading does not fulfill the trade volume requirement
    # - single trade (as maker) in continuous trading does not fulfill the trade volume requirement
    # - single trade (as taker) in continuous trading does fulfill the trade volume requirement
    # - single trade (as maker) in continuous trading does fulfill the trade volume requirement

    # Test cares about trade volume so set open volume requirement high
    Given the following network parameters are set:
      | name                                         | value            |
      | rewards.activityStreak.minQuantumTradeVolume | 10000            |
      | rewards.activityStreak.minQuantumOpenVolume  | 1000000000000000 |

    When the parties place the following orders:
      | party   | market id   | side | volume | price | resulting trades | type       | tif     |
      | <maker> | ETH/USD.0.1 | buy  | <size> | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | <taker> | ETH/USD.0.1 | sell | <size> | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the network moves ahead "1" epochs
    Given the activity streaks at epoch "1" should be:
      | party   | active for   | inactive for   | reward multiplier | vesting multiplier |
      | trader1 | <active for> | <inactive for> | <multipliers>     | <multipliers>      |
    
    Examples:
      | maker   | taker   | size | active for | inactive for | multipliers |
      | aux1    | trader1 | 1    | 0          | 1            | 1           |
      | trader1 | aux1    | 1    | 0          | 1            | 1           |
      | aux1    | trader1 | 11   | 1          | 0            | 2           |
      | trader1 | aux1    | 11   | 1          | 0            | 2           |


  Scenario Outline: Party trades when exiting an auction (0086-ASPR-004)(0086-ASPR-006)
    # Expectation: traders activity streak should be incremented if they fulfill the trade volume requirement

    # Test Cases:
    # - single trade (as taker) on auction exit does not fulfill the trade volume requirement
    # - single trade (as maker) on auction exit does not fulfill the trade volume requirement
    # - single trade (as taker) on auction exit does fulfill the trade volume requirement
    # - single trade (as maker) on auction exit does fulfill the trade volume requirement

    # Test cares about trade volume so set open volume requirement high
    Given the following network parameters are set:
      | name                                         | value            |
      | rewards.activityStreak.minQuantumTradeVolume | 10000            |
      | rewards.activityStreak.minQuantumOpenVolume  | 1000000000000000 |

    Given the parties submit the following liquidity provision:
      | id  | party  | market id   | commitment amount | fee   | lp type      |
      | lp1 | lpprov | ETH/USD.0.1 | 0                 | 0.001 | cancellation |
    And the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/USD.0.1"

    Given the parties place the following orders:
      | party   | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD.0.1 | buy  | <size> | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.0.1 | sell | <size> | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit the following liquidity provision:
      | id  | party  | market id   | commitment amount | fee   | lp type    |
      | lp2 | lpprov | ETH/USD.0.1 | 1000000           | 0.001 | submission |
    When the network moves ahead "1" epochs
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD.0.1"
    And the activity streaks at epoch "2" should be:
      | party   | active for   | inactive for   | reward multiplier | vesting multiplier |
      | trader1 | <active for> | <inactive for> | <multipliers>     | <multipliers>      |
    
    Examples:
      | maker   | taker   | size | active for | inactive for | multipliers |
      | aux1    | trader1 | 1    | 0          | 1            | 1           |
      | trader1 | aux1    | 1    | 0          | 1            | 1           |
      | aux1    | trader1 | 11   | 1          | 0            | 2           |
      | trader1 | aux1    | 11   | 1          | 0            | 2           |


  Scenario Outline: Party opens position in market (0086-ASPR-005)(0086-ASPR-006)
    # Expectation: traders activity streak should be incremented if they fulfill the open volume requirement

    # Test Cases:
    # - long position does not meet the open volume requirement
    # - short position does not meet the open volume requirement
    # - long position does meet the open volume requirement and position open for a single epoch
    # - short position does meet the open volume requirement and position open for a single epoch
    # - long position does meet the open volume requirement and position open across multiple epochs
    # - short position does meet the open volume requirement and position open across multiple epochs

    # Test cares about open volume so set trade volume requirement high
    Given the following network parameters are set:
      | name                                         | value            |
      | rewards.activityStreak.minQuantumTradeVolume | 1000000000000000 |
      | rewards.activityStreak.minQuantumOpenVolume  | 10000            |

    Given the current epoch is "1"
    And the parties place the following orders:
      | party   | market id   | side                | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD.0.1 | <counterparty side> | <size> | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.0.1 | <trader side>       | <size> | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead <epochs> epochs
    Then the activity streaks at epoch <epochs> should be:
      | party   | active for   | inactive for   | reward multiplier | vesting multiplier |
      | trader1 | <active for> | <inactive for> | <multipliers>     | <multipliers>      |
    
    Examples:
      | trader side | counterparty side | epochs | size | active for | inactive for | multipliers |
      | buy         | sell              | "1"    | 1    | 0          | 1            | 1           |
      | sell        | buy               | "1"    | 1    | 0          | 1            | 1           |
      | buy         | sell              | "1"    | 11   | 1          | 0            | 2           |
      | sell        | buy               | "1"    | 11   | 1          | 0            | 2           |
      | buy         | sell              | "2"    | 11   | 2          | 0            | 2           |
      | sell        | buy               | "2"    | 11   | 2          | 0            | 2           |


  Scenario Outline: Party splits trading between two markets using different settlement assets (0086-ASPR-004)(0086-ASPR-005)
    # Expectation: parties activity streak should be incremented if they fulfill the trade volume or open volume requirements

    # Test Cases:
    # - party meets the open volume requirement but not the trade volume requirement
    # - party meets the trade volume requirement but not the open volume requirement
    # - party meets both the trade volume requirement and the open volume requirement

    Given the following network parameters are set:
      | name                                         | value              |
      | rewards.activityStreak.minQuantumOpenVolume  | <min open volume>  |
      | rewards.activityStreak.minQuantumTradeVolume | <min trade volume> |
    Then the network moves ahead "1" blocks
    And the parties place the following orders:
      | party   | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD.0.1  | buy  | 6      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.0.1  | sell | 6      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/USD.1.10 | buy  | 60     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.1.10 | sell | 60     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the activity streaks at epoch "1" should be:
      | party   | active for | inactive for | reward multiplier | vesting multiplier |
      | trader1 | 1          | 0            | 2                 | 2                  |
    
    Examples:
      | min open volume  | min trade volume |
      | 10000            | 1000000000000000 |
      | 1000000000000000 | 10000            |
      | 10000            | 10000            |


  Scenario Outline: Party builds an activity streak and is then inactive for n epochs (0086-ASPR-007)
    # Expectation: parties activity streak should be reset if their inactivity streak is greater or equal than the network parameter

    # Test Cases:
    # - inactivity streak limit does not allow inactivity, as soon as party is inactive their streak is reset
    # - inactivity streak allows inactivity, party is inactive for less than the inactivity limit
    # - inactivity streak allows inactivity, party is inactive for the inactivity limit

    Given the following network parameters are set:
      | name                                   | value   |
      | rewards.activityStreak.inactivityLimit | <param> |

    # Open a position to be considered active then close it
    Given the parties place the following orders:
      | party   | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD.0.1 | buy  | 11     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.0.1 | sell | 11     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party   | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD.0.1 | sell | 11     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.0.1 | buy  | 11     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Given the network moves ahead <forward to epoch> epochs
    Then the activity streaks at epoch <forward to epoch> should be:
      | party   | active for   | inactive for   | reward multiplier | vesting multiplier |
      | trader1 | <active for> | <inactive for> | <multipliers>     | <multipliers>      |
    
    Examples:
      | param | forward to epoch | active for | inactive for | multipliers |
      | 1     | "2"              | 0          | 1            | 1           |
      | 5     | "5"              | 1          | 4            | 2           |
      | 5     | "6"              | 0          | 5            | 1           |