Feature: Setting and applying activity streak benefits

  Background:

    # Initialise the network
    Given time is updated to "2023-01-01T00:00:00Z"
    And the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.makerFee             | 0.01  |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.auction.minimumDuration          | 1     |
      | validators.epoch.length                 | 20s   |
      | limits.markets.maxPeggedOrders          | 4     |
    And the following network parameters are set:
      | name                                         | value                                                                                                                                                                                 |
      | rewards.vesting.baseRate                     | 0.1                                                                                                                                                                                   |
      | rewards.vesting.minimumTransfer              | 1                                                                                                                                                                                     |
      | rewards.vesting.benefitTiers                 | {"tiers": [{"minimum_quantum_balance": "1", "reward_multiplier": "1"}]}                                                                                                               |
      | rewards.activityStreak.minQuantumOpenVolume  | 1                                                                                                                                                                                     |
      | rewards.activityStreak.minQuantumTradeVolume | 1                                                                                                                                                                                     |
      | rewards.activityStreak.benefitTiers          | {"tiers": [{"minimum_activity_streak": 3, "reward_multiplier": "2", "vesting_multiplier": "2"}, {"minimum_activity_streak": 6, "reward_multiplier": "3", "vesting_multiplier": "3"}]} |

    # Initialise the markets
    And the following assets are registered:
      | id   | decimal places | quantum |
      | COIN | 0              | 1       |
    And the markets:
      | id       | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/COIN | ETH        | COIN  | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 3600s       | 1              |
    When the markets are updated:
      | id       | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/COIN | lqm-params           | 1e-3                   | 0                         |

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset | amount       |
      | lpprov                                                           | COIN  | 10000000000  |
      | aux1                                                             | COIN  | 10000000     |
      | aux2                                                             | COIN  | 10000000     |
      | trader1                                                          | COIN  | 10000000     |
      | trader2                                                          | COIN  | 10000000     |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | COIN  | 100000000000 |


    # Exit opening auctions
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | lp type    |
      | lp1 | lpprov | ETH/COIN  | 1000000           | 0.01 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/COIN  | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/COIN  | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/COIN  | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/COIN  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/COIN  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/COIN  | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/COIN"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/COIN"


  Scenario Outline: Party builds an activity streak and receives greater benefits than an infrequently active trader (0086-ASPR-008)(0086-ASPR-009)(0086-ASPR-010)
    # Expectation: parties activity streak should be incremented if they fulfill the trade volume or open volume requirements

    # Test Cases:
    # - party does not meet the lowest activity streak requirement, there multiplier is set to 1
    # - party meets the lowest activity streak requirement, party receives a larger share of the rewards
    # - party meets the highest activity streak requirement, party receives a larger share of the rewards

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | COIN  | 10000  | 2           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | COIN         |         |

    Given the network moves ahead "1" epochs
    And the current epoch is "1"
    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/COIN  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/COIN  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead <forward to epoch> epochs
    Then the activity streaks at epoch <forward to epoch> should be:
      | party   | active for   | inactive for | reward multiplier | vesting multiplier |
      | trader1 | <active for> | 0            | <multipliers>     | <multipliers>      |

    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/COIN  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/COIN  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/COIN  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/COIN  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the network moves ahead "1" epochs
    Then "trader1" should have vesting account balance of <vesting balance> for asset "COIN"
    Then the network moves ahead "1" epochs
    Then "trader1" should have vested account balance of <vested balance> for asset "COIN"

    Examples:
      | forward to epoch | active for | multipliers | vesting balance | vested balance |
      | "1"              | 1          | 1           | "5000"          | "1000"         |
      | "3"              | 3          | 2           | "6666"          | "1333"         |
      | "6"              | 6          | 3           | "7500"          | "2250"         |