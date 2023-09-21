Feature: Setting and applying activity streak benefits

  Background:

    # Initialise the network
    Given time is updated to "2023-01-01T00:00:00Z"
    And the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.makerFee             | 0.001 |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.auction.minimumDuration          | 1     |
      | validators.epoch.length                 | 20s   |
      | limits.markets.maxPeggedOrders          | 4     |
    
    # Set vesting parameters and disable multipliers
    And the following network parameters are set:
      | name                                | value                                                                                            |
      | rewards.vesting.baseRate            | 0.1                                                                                              |
      | rewards.vesting.minimumTransfer     | 1                                                                                                |
      | rewards.vesting.benefitTiers        | {"tiers": [{"minimum_quantum_balance": "1", "reward_multiplier": "1"}]}                          |
      | rewards.activityStreak.benefitTiers | {"tiers": [{"minimum_activity_streak": 1, "reward_multiplier": "1", "vesting_multiplier": "1"}]} |

    # Initialise the markets
    And the following assets are registered:
      | id       | decimal places | quantum |
      | USD.1.10 | 1              | 10      |

    And the markets:
      | id           | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USD.1.10 | ETH        | USD.1.10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
      | BTC/USD.1.10 | ETH        | USD.1.10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
    And the network moves ahead "1" blocks

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset    | amount       |
      | lpprov                                                           | USD.1.10 | 10000000000  |
      | aux1                                                             | USD.1.10 | 10000000     |
      | aux2                                                             | USD.1.10 | 10000000     |
      | trader1                                                          | USD.1.10 | 10000000     |
      | trader2                                                          | USD.1.10 | 10000000     |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USD.1.10 | 100000000000 |

    # Exit opening auctions
    Given the parties submit the following liquidity provision:
      | id  | party  | market id    | commitment amount | fee  | lp type    |
      | lp1 | lpprov | ETH/USD.1.10 | 1000000           | 0.01 | submission |
      | lp2 | lpprov | BTC/USD.1.10 | 1000000           | 0.01 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id    | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/USD.1.10 | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/USD.1.10 | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
      | lpprov | BTC/USD.1.10 | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | BTC/USD.1.10 | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
    When the parties place the following orders:
      | party | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD.1.10 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD.1.10 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD.1.10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD.1.10 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/USD.1.10 | buy  | 10     | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/USD.1.10 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USD.1.10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USD.1.10 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    # And the opening auction period ends for market "ETH/USD.1.10"
    And the opening auction period ends for market "BTC/USD.1.10"
    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD.1.10"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/USD.1.10"


  Scenario Outline: Parties rewards vest at the correct rate and adhere to the minimum quantum transfer (0085-RVST-009)(0085-RVST-010)
    # Expectation: rewards should be distributed into the same vesting account and transferred to the same vested account

    # Test cases:
    # - the expected base transfer is greater than the minimum transfer
    # - the expected base transfer is smaller than the minimum transfer
    # - the minimum transfer is smaller than the full balance

    Given the following network parameters are set:
      | name                            | value              |
      | rewards.vesting.baseRate        | <base rate>        |
      | rewards.vesting.minimumTransfer | <minimum transfer> |

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets      |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | USD.1.10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD.1.10     | ETH/USD.1.10 |
    And the network moves ahead "1" blocks

    Given the parties place the following orders:
      | party   | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD.1.10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.1.10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1    | BTC/USD.1.10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | BTC/USD.1.10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then "trader1" should have vesting account balance of "10000" for asset "USD.1.10"

    When  the network moves ahead "1" epochs
    Then "trader1" should have vested account balance of <vested amount> for asset "USD.1.10"

    Examples:
      | base rate | minimum transfer | vested amount |
      | 0.1       | 1                | "1000"        |
      | 0.1       | 200              | "2000"        |
      | 0.1       | 2000             | "10000"       |
      | 1.0       | 1                | "10000"       |


  Scenario: Parties rewards from different markets (using the same settlement asset) are transferred into the same vesting account (0085-VSPR-004)(0085-VSPR-005)
    # Expectation: rewards should be distributed into the same vesting account and t to the same vested account

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets      |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | USD.1.10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD.1.10     | ETH/USD.1.10 |
      | 2  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | USD.1.10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD.1.10     | BTC/USD.1.10 |
    And the network moves ahead "1" blocks

    Given the parties place the following orders:
      | party   | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD.1.10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.1.10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1    | BTC/USD.1.10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | BTC/USD.1.10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then "trader1" should have vesting account balance of "20000" for asset "USD.1.10"

    When  the network moves ahead "1" epochs
    Then "trader1" should have vesting account balance of "18000" for asset "USD.1.10"
    And "trader1" should have vested account balance of "2000" for asset "USD.1.10"


  Scenario: Party receive rewards from pool funded with a transfer with a non-zero lock period (0085-VSPR-011)
    # Expectation: rewards should only start vesting once the lock-period has expired

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets      | lock_period |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | USD.1.10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD.1.10     | ETH/USD.1.10 | 2           |
    And the network moves ahead "1" blocks

    Given the parties place the following orders:
      | party   | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD.1.10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.1.10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then "trader1" should have vesting account balance of "10000" for asset "USD.1.10"

    # Lock period of first batch of rewards expired - no vesting occurs
    When the network moves ahead "1" epochs
    Then "trader1" should have vesting account balance of "10000" for asset "USD.1.10"
    
    # Lock period of first batch of rewards expired - full vesting occurs
    When the network moves ahead "1" epochs
    And "trader1" should have vesting account balance of "9000" for asset "USD.1.10"
    Then "trader1" should have vested account balance of "1000" for asset "USD.1.10"

    # Generate some more rewards
    Given the parties place the following orders:
      | party   | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD.1.10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.1.10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then "trader1" should have vesting account balance of "18100" for asset "USD.1.10"
    And "trader1" should have vested account balance of "1900" for asset "USD.1.10"

    # Lock period of second batch of rewards not expired - partial vesting occurs
    When the network moves ahead "1" epochs
    Then "trader1" should have vesting account balance of "17290" for asset "USD.1.10"
    And "trader1" should have vested account balance of "2710" for asset "USD.1.10"

    # Lock period of second batch of rewards has expired - full vesting occurs
    When the network moves ahead "1" epochs
    And "trader1" should have vesting account balance of "15561" for asset "USD.1.10"
    Then "trader1" should have vested account balance of "4439" for asset "USD.1.10"


  Scenario Outline: Party receives rewards but does not withdraw them in order to receive a bonus multiplier (0085-VSPR-012)(0085-VSPR-013)
    # Expectation: if the party meets the minimum quantum balance requirement, they should receive a multiplier and a greater share of future rewards

    # Test Cases:
    # - party does meet the minimum quantum balance requirement, they receive a multiplier and a greater share of future rewards (3:1)
    # - party does meet the minimum quantum balance requirement, they receive a multiplier and a greater share of future rewards (4:1)
    # - party does not meet the minimum quantum balance requirement, they receive no multipliers and the same share of future rewards (1:1)

    Given the following network parameters are set:
      | name                         | value                                                                                                         |
      | rewards.vesting.baseRate     | 1.0                                                                                                           |
      | rewards.vesting.benefitTiers | {"tiers": [{"minimum_quantum_balance": <minimum quantum balance>, "reward_multiplier": <reward multiplier>}]} |

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets      |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | USD.1.10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD.1.10     | ETH/USD.1.10 |
    And the network moves ahead "1" blocks

    Given the parties place the following orders:
      | party   | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD.1.10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.1.10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" epochs
    Then "trader1" should have vesting account balance of "10000" for asset "USD.1.10"

    Given the parties place the following orders:
      | party   | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD.1.10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/USD.1.10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/USD.1.10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/USD.1.10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then "trader1" should have vesting account balance of <trader1 rewards> for asset "USD.1.10"
    And "trader2" should have vesting account balance of <trader2 rewards> for asset "USD.1.10"

    Examples:
      | minimum quantum balance | reward multiplier | trader1 rewards | trader2 rewards |
      | "500"                   | "3"               | "7500"          | "2500"          |
      | "500"                   | "4"               | "8000"          | "2000"          |
      | "5000000"               | "3"               | "5000"          | "5000"          |


  Scenario: Parties attempt to transfer rewards to or from vesting and vested accounts (0085-VSPR-006)(0085-VSPR-007)(0085-VSPR-008)
    # Expectation: only transfers from the vested account should be valid, all other transfers should be rejected

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets      | lock_period |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | USD.1.10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD.1.10     | ETH/USD.1.10 | 0           |
    And the network moves ahead "1" blocks

    Given the parties place the following orders:
      | party                                                            | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1                                                             | ETH/USD.1.10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ETH/USD.1.10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have vesting account balance of "9000" for asset "USD.1.10"
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have vested account balance of "1000" for asset "USD.1.10"

    # Check rewards cannot be transferred to or from the following accounts
    Given the parties submit the following one off transfers:
      | id  | from                                                             | from_account_type            | to                                                               | to_account_type              | asset    | amount | delivery_time        | error                         |
      | oo1 | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_VESTED_REWARDS  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL         | USD.1.10 | 1      | 2023-01-01T01:00:00Z |                               |
      | oo2 | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_VESTING_REWARDS | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL         | USD.1.10 | 1      | 2023-01-01T01:00:00Z | unsupported from account type |
      | oo3 | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL         | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_VESTED_REWARDS  | USD.1.10 | 1      | 2023-01-01T01:00:00Z | unsupported to account type   |
      | oo4 | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL         | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_VESTING_REWARDS | USD.1.10 | 1      | 2023-01-01T01:00:00Z | unsupported to account type   |
    And "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have vested account balance of "999" for asset "USD.1.10"