Feature: Setting and applying referee benefit factors
  
  Background:

    # Initialise timings
    Given time is updated to "2023-01-01T00:00:00Z"
    And the average block duration is "1"

    # Initialise the markets and network parameters
    Given the following network parameters are set:
      | name                                                    | value      |
      | market.fee.factors.infrastructureFee                    | 0.1        |
      | market.fee.factors.makerFee                             | 0.1        |
      | market.auction.minimumDuration                          | 1          |
      | limits.markets.maxPeggedOrders                          | 4          |
      | referralProgram.minStakedVegaTokens                     | 0          |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 1000000000 |

    # Initalise the referral program then move forwards an epoch to start the program
    Given the referral benefit tiers "rbt":
      | minimum running notional taker volume | minimum epochs | referral reward factor | referral discount factor |
      | 2000                                  | 2              | 0.02                   | 0.02                     |
      | 3000                                  | 3              | 0.03                   | 0.03                     |
    And the referral staking tiers "rst":
      | minimum staked tokens | referral reward multiplier |
      | 1                     | 1                          |
    And the referral program:
      | end of program       | window length | benefit tiers | staking tiers |
      | 2023-12-12T12:12:12Z | 7             | rbt           | rst           |
    And the network moves ahead "1" epochs

    # Initialse the assets and markets
    And the following assets are registered:
      | id   | decimal places |
      | USDT | 1              |
    And the markets:
      | id       | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USDT | ETH        | USDT  | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 1              | 1                       |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 3600s       | 1              |
    When the markets are updated:
      | id       | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/USDT | lqm-params           | 1e-3                   | 0                         |

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party     | asset | amount      |
      | lpprov    | USDT  | 10000000000 |
      | aux1      | USDT  | 10000000    |
      | aux2      | USDT  | 10000000    |
      | referrer1 | USDT  | 10000000    |
      | referee1  | USDT  | 10000000    |
      | referee2  | USDT  | 10000000    |

    # Exit the opening auction
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/USDT  | 1000000           | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/USDT  | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/USDT  | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USDT  | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USDT"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USDT"

    # Create the referral set and codes
    Given the parties create the following referral codes:
      | party     | code            |
      | referrer1 | referral-code-1 |
    And the parties apply the following referral codes:
      | party    | code            |
      | referee1 | referral-code-1 |


  Scenario Outline: Referral set generates variable running taker volume with a referee with variable epochs in set
    # Expectation: Referral reward factor and referral discount factor should be set correctly according to the spec

    Given the parties place the following orders:
      | party    | market id | side | volume           | price | resulting trades | type       | tif     |
      | aux1     | ETH/USDT  | buy  | <initial volume> | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USDT  | sell | <initial volume> | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead <time in set> epochs
    Then the referral set stats for code "referral-code-1" at epoch <time in set> should have a running volume of <running volume>:
      | party    | reward factor       | discount factor       |
      | referee1 | <max reward factor> | <max discount factor> |

    Given the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    # Test cases
    # - Referral set does not fulfill the smallest 'minimimum running taker volume' requirement and referee does not fulfill the smallest 'minimum epochs' requirement
    # - Referral set does not fulfill the smallest 'minimimum running taker volume' requirement but referee fulfulls the smallest 'minimum epochs' requirement
    # - Referral set fulfills the smallest 'minimimum running taker volume' requirement but referee does fulfill the smallest 'minimum epochs' requirement
    # - Referral set fulfills the 'minimum running taker volume' requirement tier for a higher tier than a referee fulfulls the 'minimum epochs' requirement for
    # - Referral set fulfills the 'minimum running taker volume' requirement tier for a lower tier than a referee fulfulls the 'minimum epochs' requirement for
    # - Referral set fulfills the 'minimum running taker volume' requirement tier for the same tier as a referee fulfulls the 'minimum epochs' requirement for
    Examples:
      | initial volume | running volume | time in set | max reward factor | max discount factor | seller maker fee referrer discount |
      | 10             | 1000           | "1"         | 0                 | 0                   | 0                                  |
      | 10             | 1000           | "4"         | 0                 | 0                   | 0                                  |
      | 20             | 2000           | "1"         | 0.02              | 0                   | 0                                  |
      | 30             | 3000           | "2"         | 0.03              | 0.02                | 2                                  |
      | 20             | 2000           | "3"         | 0.02              | 0.02                | 2                                  |
      | 30             | 3000           | "3"         | 0.03              | 0.03                | 3                                  |


  Scenario: Simple test for showing discounts not applied, will be merged into above scenario once issue closed
    # Expectation: Referee has non-zero benefit factors, fees should be discounted and comission paid to the referrer

    Given the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USDT  | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USDT  | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" epochs
    Then the referral set stats for code "referral-code-1" at epoch "2" should have a running volume of 2000:
      | party    | reward factor | discount factor |
      | referee1 | 0.02          | 0.02            |

    Given the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the following trades should be executed:
      | buyer | price | size | seller   |
      | aux1  | 1000  | 20   | referee1 |
    # ISSUE: referee1 still pays full fees despite active referral program and positive discount factor
    # ISSUE: referee1 pays no commision to referrer despite active referral program and positive reward factor
    Then the following transfers should happen:
      | from     | to     | from account         | to account                       | market id | amount | asset |
      | referee1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_MAKER          | ETH/USDT  | 100    | USDT  |
      | referee1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/USDT  | 100    | USDT  |
      | referee1 |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 100    | USDT  |
