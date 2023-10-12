Feature: Team Rewards

  Background:

    # Initialise the network
    Given time is updated to "2023-01-01T00:00:00Z"
    And the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.makerFee             | 0.001 |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.auction.minimumDuration          | 1     |
      | validators.epoch.length                 | 60s   |
      | limits.markets.maxPeggedOrders          | 4     |
      | referralProgram.minStakedVegaTokens     | 0     |

    # Initialise the markets
    And the following assets are registered:
      | id       | decimal places | quantum |
      | USD-1-10 | 1              | 10      |
    And the markets:
      | id           | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USD-1-10 | ETH        | USD-1-10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset    | amount      |
      | lpprov                                                           | USD-1-10 | 10000000000 |
      | aux1                                                             | USD-1-10 | 10000000    |
      | aux2                                                             | USD-1-10 | 10000000    |
      | aux3                                                             | USD-1-10 | 10000000000 |
      | aux4                                                             | USD-1-10 | 10000000000 |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USD-1-10 | 10000000000 |
      | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | USD-1-10 | 10000000000 |
      | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | USD-1-10 | 10000000000 |
      | party1                                                           | USD-1-10 | 10000000000 |
      | party2                                                           | USD-1-10 | 10000000000 |
      | party3                                                           | USD-1-10 | 10000000000 |
      | party4                                                           | USD-1-10 | 10000000000 |

    # Exit opening auctions
    Given the parties submit the following liquidity provision:
      | id  | party  | market id    | commitment amount | fee  | lp type    |
      | lp1 | lpprov | ETH/USD-1-10 | 1000000           | 0.01 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id    | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/USD-1-10 | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/USD-1-10 | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
    When the parties place the following orders:
      | party | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD-1-10 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD-1-10 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-10 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USD-1-10"
    When the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-1-10"

    # Create the teams, 2 members each ( + referrer)
    Given the following teams with referees are created:
      | referrer  | prefix | code            | team name | referees | balance  | asset    |
      | referrer1 | ref1   | referral-code-1 | team1     | 2        | 10000000 | USD-1-10 |
      | referrer2 | ref2   | referral-code-2 | team2     | 2        | 10000000 | USD-1-10 |

  @TeamStep
  Scenario: Create a situation where we have parties in different teams, and some not in teams, then change the teams

    # set up some recurring payments, one for both teams, one for individuals only, one for a single team only
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | entity_scope | individual_scope | teams       | ntop | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets      | lock_period |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | TEAMS        |                  | team1,team2 | 0.2  | USD-1-10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD-1-10     | ETH/USD-1-10 | 1           |
      | 2  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | INDIVIDUALS  | NOT_IN_TEAM      |             |      | USD-1-10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD-1-10     | ETH/USD-1-10 | 1           |
      | 3  | a7c4b181ef9bf5e9029a016f854e4ad471208020fd86187d07f0b420004f06a4 | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | TEAMS        |                  | team1       | 1    | USD-1-10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD-1-10     | ETH/USD-1-10 | 1           |
    ## Bunch of orders:
    # Team1: 20, 10
    # Team2: 10, 9
    # Non-team: 1, 21, 5
    # Non-team: 3, 1 fee of 10k to be distributed as x, 21x, and 5x (where x = 10,000/27 = 370.370370...)
    #           giving 370, 7777, 1851
    # Team2: 1 fee shared with team1 (n top % 0.2 where team1 == 20, team2 == 10): They split 1/3rd of 10k -> 1666 each
    # Team1: 2 fees, 1 is shared between 2 (5k each), the second is 2/3rds shared -> 8,333
    And the parties place the following orders:
      | party     | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1      | ETH/USD-1-10 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref1-0001 | ETH/USD-1-10 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref1-0002 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux3      | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref2-0001 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux3      | ETH/USD-1-10 | sell | 9      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref2-0002 | ETH/USD-1-10 | buy  | 9      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux4      | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1    | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux4      | ETH/USD-1-10 | sell | 21     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2    | ETH/USD-1-10 | buy  | 21     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux4      | ETH/USD-1-10 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3    | ETH/USD-1-10 | buy  | 5      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party     | asset    | balance |
      | ref1-0001 | USD-1-10 | 8333    |
      | ref1-0002 | USD-1-10 | 8333    |
      | ref2-0001 | USD-1-10 | 1666    |
      | ref2-0002 | USD-1-10 | 1666    |
      | party1    | USD-1-10 | 370     |
      | party2    | USD-1-10 | 7777    |
      | party3    | USD-1-10 | 1851    |

    # Move party2 to team2, the nop should now force a 5 way split of the fee
    When the parties apply the following referral codes:
      | party  | code            | is_team | team  |
      | party2 | referral-code-2 | true    | team2 |
    Then the team "team2" has the following members:
      | party     |
      | referrer2 |
      | ref2-0001 |
      | ref2-0002 |
      | party2    |
    ## Division as follows:
    # Team1: 5k each from transfer only applicable to them = 8333 + 5000 = 13333
    #       10k split equally across team1 and 2 = 2.5k each = 13333 + 2500 = 15833
    # Team2: 5k split 3 ways == 1666 each: ref2-0001 and ref2-0002 = 1666 + 1666 == 3333
    #       party2 gets 7777 + 1666 = 9443.66666 ≃ 9444
    # Party 1 and 3 + 5k each
    And the parties place the following orders:
      | party     | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1      | ETH/USD-1-10 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref1-0001 | ETH/USD-1-10 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref1-0002 | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux3      | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref2-0001 | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux3      | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref2-0002 | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux4      | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1    | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux4      | ETH/USD-1-10 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2    | ETH/USD-1-10 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux4      | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3    | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party     | asset    | balance |
      | ref1-0001 | USD-1-10 | 15833   |
      | ref1-0002 | USD-1-10 | 15833   |
      | ref2-0001 | USD-1-10 | 3333    |
      | ref2-0002 | USD-1-10 | 3333    |
      | party1    | USD-1-10 | 5371    |
      | party2    | USD-1-10 | 9444    |
      | party3    | USD-1-10 | 6852    |
