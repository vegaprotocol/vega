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
    
    # Set vesting parmaeters and disable multipliers
    And the following network parameters are set:
      | name                                | value                                                                                            |
      | rewards.vesting.baseRate            | 0.1                                                                                              |
      | rewards.vesting.minimumTransfer     | 1                                                                                                |
      | rewards.vesting.benefitTiers        | {"tiers": [{"minimum_quantum_balance": "1", "reward_multiplier": "1"}]}                          |
      | rewards.activityStreak.benefitTiers | {"tiers": [{"minimum_activity_streak": 1, "reward_multiplier": "1", "vesting_multiplier": "1"}]} |

    # Initialise the markets
    And the following assets are registered:
      | id   | decimal places | quantum |
      | COIN | 0              | 1       |
    And the markets:
      | id       | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/COIN | ETH        | COIN  | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
      | BTC/COIN | ETH        | COIN  | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
    And the network moves ahead "1" blocks

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
      | lp2 | lpprov | BTC/COIN  | 1000000           | 0.01 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/COIN  | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/COIN  | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
      | lpprov | BTC/COIN  | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | BTC/COIN  | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/COIN  | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/COIN  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/COIN  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/COIN  | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/COIN  | buy  | 10     | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/COIN  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/COIN  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/COIN  | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    # And the opening auction period ends for market "ETH/COIN"
    And the opening auction period ends for market "BTC/COIN"
    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/COIN"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/COIN"


  Scenario: Party earns rewards from different transfers scoping different markets (0085-VSPR-004)(0085-VSPR-005)
    # Expectation: rewards should be distributed into the same vesting account and transfered to the same vested account

    # Setup the recurring transfer
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets  |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | COIN  | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | COIN         | ETH/COIN |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | COIN  | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | COIN         | BTC/COIN |
    And the network moves ahead "1" blocks

    # Generate rewards from the two markets
    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/COIN  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/COIN  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1    | BTC/COIN  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | BTC/COIN  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then "trader1" should have vesting account balance of "20000" for asset "COIN"

    # Vest some rewards
    Given the network moves ahead "1" epochs
    # ISSUE: would expect only 2000 to be vested, instead of 5000
    And "trader1" should have vesting account balance of "15000" for asset "COIN"
    Then "trader1" should have vested account balance of "5000" for asset "COIN"


