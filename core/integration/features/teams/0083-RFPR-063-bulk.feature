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
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USD-1-10 | 10000000000 |
      | another1                                                         | USD-1-10 | 10000000000 |
      | another2                                                         | USD-1-10 | 10000000000 |

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

    # Create the teams
    Given the following teams with referees are created:
      | referrer  | prefix | code            | team name | referees | balance  | asset    | closed | allow list |
      | referrer1 | ref1   | referral-code-1 | team1     | 20       | 10000000 | USD-1-10 | true   | another1   |
      | referrer2 | ref2   | referral-code-2 | team2     | 20       | 10000000 | USD-1-10 | true   |            |

  @Closed
  Scenario: 0083-RFPR-063 with bulk team creation

    When the parties apply the following referral codes:
      | party    | code            | is_team | team  | reference | error                                     |
      | another2 | referral-code-2 | true    | team2 | some ref  | party is not allowed to join team "team2" |
    When the parties apply the following referral codes:
      | party    | code            | is_team | team  | reference |
      | another1 | referral-code-1 | true    | team1 | some ref  |
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | entity_scope | teams       | ntop | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets      |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | TEAMS        | team1,team2 | 0.2  | USD-1-10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD-1-10     | ETH/USD-1-10 |
    ## Bunch of orders:
    # Team1: 20, 10, 10, 10, 1 (out of 20)
    # Team2: 21, 19, 10, 5 (out of 10)
    # TopN 0.2 should be based on 50/4 and 40/2 -> 12.5 and 20
    # This is then divided between 5 parties in team1, 4 in team2 respectively,
    # so team1 parties receive 12.5/5 (or 2.5), whereas team2 referees receive 20/4 (or 5)
    # We expect to see the vesting account balances of team2 to be 2x those of team1
    And the parties place the following orders:
      | party     | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1      | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref1-0001 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux3      | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref1-0002 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux3      | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref1-0003 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux3      | ETH/USD-1-10 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref1-0004 | ETH/USD-1-10 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux3      | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref1-0005 | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USD-1-10 | sell | 21     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref2-0001 | ETH/USD-1-10 | buy  | 21     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USD-1-10 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref2-0002 | ETH/USD-1-10 | buy  | 5      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USD-1-10 | sell | 19     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref2-0003 | ETH/USD-1-10 | buy  | 19     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | ref2-0004 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party     | asset    | balance |
      | ref1-0001 | USD-1-10 | 952     |
      | ref1-0002 | USD-1-10 | 952     |
      | ref1-0003 | USD-1-10 | 952     |
      | ref1-0004 | USD-1-10 | 952     |
      | ref1-0005 | USD-1-10 | 952     |
      | ref1-0006 | USD-1-10 | 0       |
      | ref2-0001 | USD-1-10 | 1309    |
      | ref2-0002 | USD-1-10 | 1309    |
      | ref2-0003 | USD-1-10 | 1309    |
      | ref2-0004 | USD-1-10 | 1309    |
      | ref2-0005 | USD-1-10 | 0       |
    #And "ref1-0002" should have vesting account balance of "0" for asset "USD-1-10"
