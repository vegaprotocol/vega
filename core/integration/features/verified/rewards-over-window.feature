Feature: Maker fees paid reward metric calculated correctly for time window

  # Test checks bug where previous maker fees received were being stored in place of
  # maker fees paid in the market activity tracker.

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
      | referrer1                                                        | USD-1-10 | 10000000    |
      | referee1                                                         | USD-1-10 | 10000000    |
      | referee2                                                         | USD-1-10 | 10000000    |
      | referee3                                                         | USD-1-10 | 10000000    |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USD-1-10 | 10000000    |

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
      | aux1  | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-10 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USD-1-10"
    When the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-1-10"

    # Create the teams
    Given the parties create the following referral codes:
      | party     | code            | is_team | team  |
      | referrer1 | referral-code-1 | true    | team1 |
      | referrer2 | referral-code-2 | true    | team2 |
    And the parties apply the following referral codes:
      | party    | code            | is_team | team  |
      | referee1 | referral-code-1 | true    | team1 |
      | referee2 | referral-code-2 | true    | team2 |
      | referee3 | referral-code-2 | true    | team2 |
    And the team "team1" has the following members:
      | party     |
      | referrer1 |
      | referee1  |
    And the team "team2" has the following members:
      | party     |
      | referrer2 |
      | referee2  |
      | referee3  |


  Scenario: Party funds pool with recurring transfer scoping individuals

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | entity_scope | teams | ntop | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets      | window_length |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | INDIVIDUALS  |       | 1    | USD-1-10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD-1-10     | ETH/USD-1-10 | 2             |
    And the parties place the following orders:
      | party     | market id    | side | volume | price | resulting trades | type       | tif     |
      | referrer1 | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1  | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee2  | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USD-1-10 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee3  | ETH/USD-1-10 | buy  | 5      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs

    Then "referee1" should have vesting account balance of "4000" for asset "USD-1-10"
    And "referee2" should have vesting account balance of "4000" for asset "USD-1-10"
    And "referee3" should have vesting account balance of "2000" for asset "USD-1-10"


  Scenario: Party funds pool with recurring transfer scoping teams

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | entity_scope | teams | ntop | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets      | window_length |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | TEAMS        |       | 1    | USD-1-10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD-1-10     | ETH/USD-1-10 | 2             |
    And the parties place the following orders:
      | party     | market id    | side | volume | price | resulting trades | type       | tif     |
      | referrer1 | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1  | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee2  | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USD-1-10 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee3  | ETH/USD-1-10 | buy  | 5      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs

    Then "referee1" should have vesting account balance of "5000" for asset "USD-1-10"
    And "referee2" should have vesting account balance of "2500" for asset "USD-1-10"
    And "referee3" should have vesting account balance of "2500" for asset "USD-1-10"
