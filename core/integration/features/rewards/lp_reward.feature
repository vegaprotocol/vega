Feature: Rewards for liquidity fees recieved

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
      | aux2  | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-10 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USD-1-10"
    When the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-1-10"

    # Create the teams
    Given the parties create the following referral codes:
      | party     | code            | is_team | team  |
      | referrer1 | referral-code-1 | true    | team1 |
    And the parties apply the following referral codes:
      | party  | code            | is_team | team  |
      | lpprov | referral-code-1 | true    | team1 |
    And the team "team1" has the following members:
      | party     |
      | referrer1 |
      | lpprov    |

  Scenario: Party funds reward pool with lp received fees and dispatch metric scoping individuals

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | entity_scope | asset    | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets      |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES | INDIVIDUALS  | USD-1-10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_LP_FEES_RECEIVED | USD-1-10     | ETH/USD-1-10 |
    And the parties place the following orders:
      | party    | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1     | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee2 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1     | ETH/USD-1-10 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee3 | ETH/USD-1-10 | buy  | 5      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    # Confirm lpprov did indeed receive fees
    Then the following transfers should happen:
      | from   | to     | from account                   | to account                     | market id    | amount | asset    |
      |        | lpprov | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/USD-1-10 | 2500   | USD-1-10 |
      | lpprov | lpprov | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | ETH/USD-1-10 | 2500   | USD-1-10 |
    # We would expect the lp to recieve the full reward
    Then parties should have the following vesting account balances:
      | party  | asset    | balance |
      | lpprov | USD-1-10 | 10000   |


  Scenario: Given the following dispatch metrics, if an `eligible keys` list is specified in the recurring transfer, only parties included in the list and meeting other eligibility criteria should receive a score 0056-REWA-213

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | entity_scope | asset    | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets      | eligible_keys |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES | INDIVIDUALS  | USD-1-10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_LP_FEES_RECEIVED | USD-1-10     | ETH/USD-1-10 | aux1          |
    And the parties place the following orders:
      | party    | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1     | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee2 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1     | ETH/USD-1-10 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee3 | ETH/USD-1-10 | buy  | 5      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    # Confirm lpprov did indeed receive fees
    Then the following transfers should happen:
      | from   | to     | from account                   | to account                     | market id    | amount | asset    |
      |        | lpprov | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/USD-1-10 | 2500   | USD-1-10 |
      | lpprov | lpprov | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | ETH/USD-1-10 | 2500   | USD-1-10 |
    # We would expect the lp to recieve the full reward but it's not in eligible keys so no. 
    Then parties should have the following vesting account balances:
      | party  | asset    | balance |
      | lpprov | USD-1-10 | 0       |

    # nothing transferred
    Then "a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf" should have general account balance of "10000000" for asset "USD-1-10"


  Scenario: Party funds reward pool with lp received fees dispatch metric and scoping teams

    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | entity_scope | teams | ntop | asset    | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets      |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES | TEAMS        | team1 | 1    | USD-1-10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_LP_FEES_RECEIVED | USD-1-10     | ETH/USD-1-10 |
    And the parties place the following orders:
      | party    | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1     | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee2 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1     | ETH/USD-1-10 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee3 | ETH/USD-1-10 | buy  | 5      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    # Confirm lpprov did indeed receive fees
    Then the following transfers should happen:
      | from   | to     | from account                   | to account                     | market id    | amount | asset    |
      |        | lpprov | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/USD-1-10 | 2500   | USD-1-10 |
      | lpprov | lpprov | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | ETH/USD-1-10 | 2500   | USD-1-10 |

    # We would expect the lp to recieve the full reward
    Then parties should have the following vesting account balances:
      | party  | asset    | balance |
      | lpprov | USD-1-10 | 10000   |







