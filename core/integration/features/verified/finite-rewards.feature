Feature: Team Rewards

    Setup a maker fees received team game with a fee cap.

  We want to make it so that one team is allocated rewards and the other team is allocated rewards.

  - Team A should have also paid rewards and not have their rewards capped.
  - Team B should not have paid rewards and have their rewards capped.

  Question is what happens to the left over rewards.


  Background:

    And the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | referralProgram.minStakedVegaTokens     | 0     |
      | market.fee.factors.makerFee             | 0.01  |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | validators.epoch.length                 | 60s   |

    # Initialise the markets
    And the following assets are registered:
      | id      | decimal places | quantum |
      | USD-0-1 | 0              | 1       |
    And the markets:
      | id          | quote name | asset   | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USD-0-1 | ETH        | USD-0-1 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset   | amount        |
      | aux1                                                             | USD-0-1 | 1000000000000 |
      | aux2                                                             | USD-0-1 | 1000000000000 |
      | party1                                                           | USD-0-1 | 1000000000000 |
      | party2                                                           | USD-0-1 | 1000000000000 |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USD-0-1 | 1000000000000 |

    # Exit opening auctions
    When the parties place the following orders:
      | party | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD-0-1 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-0-1 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USD-0-1"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-0-1"

    Given the parties create the following referral codes:
      | party  | code            | is_team | team  |
      | party1 | referral-code-1 | true    | team1 |
    Given the parties create the following referral codes:
      | party  | code            | is_team | team  |
      | party2 | referral-code-1 | true    | team2 |

  Scenario: Check a one-off pay out can be done with start epoch = end epoch

    Given the current epoch is "0"
    When the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                         | entity_scope | asset   | amount | start_epoch | end_epoch | factor | metric                              | metric_asset | markets     | lock_period | window_length | ntop | cap_reward_fee_multiple |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | TEAMS        | USD-0-1 | 100    | 1           | 2         | 1      | DISPATCH_METRIC_MAKER_FEES_RECEIVED | USD-0-1      | ETH/USD-0-1 | 10          | 1             | 1    | 1                       |
    Then the network moves ahead "1" epochs

    ## pary1
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1   | ETH/USD-0-1 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/USD-0-1 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/USD-0-1 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1   | ETH/USD-0-1 | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the following trades should be executed:
      | buyer  | size | price | seller | aggressor side | buyer maker fee | seller maker fee |
      | party1 | 10   | 1000  | aux1   | buy            | 100             | 0                |
      | party1 | 10   | 1000  | aux1   | sell           | 0               | 100              |

    # party2
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1   | ETH/USD-0-1 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/USD-0-1 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the following trades should be executed:
      | buyer  | size | price | seller | aggressor side | buyer maker fee |
      | party2 | 10   | 1000  | aux1   | buy            | 100             |

    When the network moves ahead "1" epochs
    And parties should have the following vesting account balances:
      | party  | asset   | balance |
      | party1 | USD-0-1 | 100     |
      | party2 | USD-0-1 | 0       |


    ## pary1
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1   | ETH/USD-0-1 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/USD-0-1 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/USD-0-1 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1   | ETH/USD-0-1 | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the following trades should be executed:
      | buyer  | size | price | seller | aggressor side | buyer maker fee | seller maker fee |
      | party1 | 10   | 1000  | aux1   | buy            | 100             | 0                |
      | party1 | 10   | 1000  | aux1   | sell           | 0               | 100              |

    # party2
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1   | ETH/USD-0-1 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/USD-0-1 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the following trades should be executed:
      | buyer  | size | price | seller | aggressor side | buyer maker fee |
      | party2 | 10   | 1000  | aux1   | buy            | 100             |

    When the network moves ahead "1" epochs
    And parties should have the following vesting account balances:
      | party  | asset   | balance |
      | party1 | USD-0-1 | 200     |
      | party2 | USD-0-1 | 0       |

    Then debug transfers
