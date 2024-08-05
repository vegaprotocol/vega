Feature: Discounts from multiple sources

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
      | 3600    | 0.99        | 15                |

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
      | validators.epoch.length                                 | 10s        |


    # Initalise the referral program
    Given the referral benefit tiers "rbt":
      | minimum running notional taker volume | minimum epochs | referral reward infra factor | referral reward maker factor | referral reward liquidity factor | referral discount infra factor | referral discount maker factor | referral discount liquidity factor |
      | 1000                                  | 2              | 0.0                          | 0.0                          | 0.0                              | 0.11                           | 0.12                           | 0.13                               |
      | 10000                                 | 2              | 0.0                          | 0.0                          | 0.0                              | 1.0                            | 1.0                            | 1.0                                |

    And the referral staking tiers "rst":
      | minimum staked tokens | referral reward multiplier |
      | 1                     | 1                          |
    And the referral program:
      | end of program       | window length | benefit tiers | staking tiers |
      | 2023-12-12T12:12:12Z | 7             | rbt           | rst           |
    # Initialise the volume discount program
    And the volume discount program tiers named "vdt":
      | volume | infra factor | liquidity factor | maker factor |
      | 1000   | 0.11         | 0.12             | 0.13         |
      | 1000   | 0.11         | 0.12             | 0.13         |

    And the volume discount program:
      | id  | tiers | closing timestamp | window length |
      | id1 | vdt   | 0                 | 7             |
    # Move to the next epoch to start the programs
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
    Then the opening auction period ends for market "ETH/USD.1.1"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD.1.1"
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


  Scenario: When in continuous trading, fees discounted correctly when party has non-zero referral and volume discount factors (0029-FEES-032)
    # Expectation: referral discount applied before volume discount

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" epochs
    Then the referral set stats for code "referral-code-1" at epoch "2" should have a running volume of 2000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0                   | 0                   | 0                       | 0.11                  | 0.12                  | 0.13                      |
    And the party "referee1" has the following taker notional "2000"
    And the party "referee1" has the following discount infra factor "0.11"
    And the party "referee1" has the following discount liquidity factor "0.12"
    And the party "referee1" has the following discount maker factor "0.13"

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 100    | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the following trades should be executed:
      | buyer | price | size | seller   |
      | aux1  | 1000  | 100  | referee1 |
    Then the following transfers should happen:
      | from     | to     | from account         | to account                       | market id   | amount | asset   |
      | referee1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_MAKER          | ETH/USD.1.1 | 77     | USD.1.1 |
      | referee1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/USD.1.1 | 77     | USD.1.1 |
      | referee1 |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |             | 80     | USD.1.1 |

    # Additionally check if referral discount fully discounts fees, core does not panic when trying to apply volume discounts
    Given the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch "3" should have a running volume of 12000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0                   | 0                   | 0                       | 1.0                   | 1.0                   | 1.0                       |
    And the party "referee1" has the following taker notional "12000"
    And the party "referee1" has the following discount infra factor "0.11"
    And the party "referee1" has the following discount liquidity factor "0.12"
    And the party "referee1" has the following discount maker factor "0.13"

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD.1.1 | buy  | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.1 | sell | 100    | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the following trades should be executed:
      | buyer | price | size | seller   |
      | aux1  | 1000  | 100  | referee1 |
    Then the following transfers should happen:
      | from     | to     | from account         | to account                       | market id   | amount | asset   |
      | referee1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_MAKER          | ETH/USD.1.1 | 0      | USD.1.1 |
      | referee1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/USD.1.1 | 0      | USD.1.1 |
      | referee1 |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |             | 0      | USD.1.1 |

  Scenario: When exiting an auction, fees discounted correctly when party has non-zero referral and volume discount factors (0029-FEES-033)
    # Expectation: fee should be split between buyer and seller, referral discount applied before volume discount

    When the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux3     | ETH/USD.1.2 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.2 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/USD.1.2" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | horizon | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS | 7469         | 1000000        | 21            | 3600    | 973       | 1027      |
    When the network moves ahead "2" epochs
    Then the referral set stats for code "referral-code-1" at epoch "2" should have a running volume of 2000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0                   | 0                   | 0                       | 0.11                  | 0.12                  | 0.13                      |
    And the party "referee1" has the following taker notional "2000"
    And the party "referee1" has the following discount infra factor "0.11"
    And the party "referee1" has the following discount liquidity factor "0.12"
    And the party "referee1" has the following discount maker factor "0.13"

    When the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     |
      | ptbuy  | ETH/USD.1.2 | buy  | 2      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
      | ptsell | ETH/USD.1.2 | sell | 2      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/USD.1.2" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 7935         | 1000000        | 21            | 15          |
    ## Triger price auction
    # Cancel the liquidity commitment triggering an auction
    And the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/USD.1.2"
    When the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux3     | ETH/USD.1.2 | buy  | 200    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD.1.2 | sell | 200    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" epochs
    Then debug trades
    Then the following trades should be executed:
      | buyer | price | size | seller   |
      | aux3  | 1000  | 198  | referee1 |
      | aux3  | 1000  | 2    | ptsell   |
    And the following transfers should happen:
      | from     | to     | from account         | to account                       | market id   | amount | asset   |
      | referee1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/USD.1.2 | 77     | USD.1.1 |
      | referee1 |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |             | 80     | USD.1.1 |
