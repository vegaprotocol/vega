Feature: Calculating referral set running volumes

  Background:

    # Initialise the network
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
      | 3600    | 0.99        | 3                 |
    And the following network parameters are set:
      | name                                                    | value      |
      | market.fee.factors.infrastructureFee                    | 0.001      |
      | market.fee.factors.makerFee                             | 0.001      |
      | network.markPriceUpdateMaximumFrequency                 | 0s         |
      | market.auction.minimumDuration                          | 1          |
      | validators.epoch.length                                 | 60s        |
      | limits.markets.maxPeggedOrders                          | 4          |
      | referralProgram.minStakedVegaTokens                     | 0          |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 1000000000 |

    # Initalise the referral program
    Given the referral benefit tiers "rbt":
      | minimum running notional taker volume | minimum epochs | referral reward infra factor | referral reward maker factor | referral reward liquidity factor | referral discount infra factor | referral discount maker factor | referral discount liquidity factor |
      | 100                                   | 1              | 0.11                         | 0.12                         | 0.13                             | 0.14                           | 0.15                           | 0.16                               |
    And the referral staking tiers "rst":
      | minimum staked tokens | referral reward multiplier |
      | 1                     | 1                          |
    And the referral program:
      | end of program       | window length | benefit tiers | staking tiers |
      | 2023-12-12T12:12:12Z | 7             | rbt           | rst           |
    And the network moves ahead "1" epochs

    # Initialise the markets
    And the following assets are registered:
      | id       | decimal places | quantum |
      | USD-1-1  | 1              | 1       |
      | USD-2-10 | 2              | 10      |
    And the markets:
      | id           | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USD-1-1  | ETH        | USD-1-1  | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
      | ETH/USD-2-10 | ETH        | USD-2-10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 1              | 1                       |
      | ETH/USD-1-2  | ETH        | USD-1-1  | log-normal-risk-model         | margin-calculator-1       | 1                | default-none | price-monitoring | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 3600s       | 1              |
    When the markets are updated:
      | id           | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/USD-1-1  | lqm-params           | 1e-3                   | 0                         |
      | ETH/USD-2-10 | lqm-params           | 1e-3                   | 0                         |
      | ETH/USD-1-2  | lqm-params           | 1e-3                   | 0                         |
    And the network moves ahead "1" blocks

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party     | asset    | amount      |
      | lpprov    | USD-2-10 | 10000000000 |
      | aux1      | USD-2-10 | 10000000    |
      | aux2      | USD-2-10 | 10000000    |
      | referrer1 | USD-2-10 | 10000000    |
      | referee1  | USD-2-10 | 10000000    |
      | referee2  | USD-2-10 | 10000000    |
      | lpprov    | USD-1-1  | 10000000000 |
      | lpprov2   | USD-1-1  | 10000000000 |
      | aux1      | USD-1-1  | 10000000    |
      | aux2      | USD-1-1  | 10000000    |
      | aux3      | USD-1-1  | 10000000    |
      | aux4      | USD-1-1  | 10000000    |
      | referrer1 | USD-1-1  | 10000000    |
      | referee1  | USD-1-1  | 10000000    |
      | referee2  | USD-1-1  | 10000000    |
      | ptbuy     | USD-1-1  | 10000000    |
      | ptsell    | USD-1-1  | 10000000    |

    # Exit opening auctions
    Given the parties submit the following liquidity provision:
      | id  | party   | market id    | commitment amount | fee  | lp type    |
      | lp1 | lpprov  | ETH/USD-1-1  | 1000000           | 0.01 | submission |
      | lp2 | lpprov  | ETH/USD-2-10 | 10000000          | 0.01 | submission |
      | lp3 | lpprov2 | ETH/USD-1-2  | 10000000          | 0.01 | submission |
    And the parties place the following pegged iceberg orders:
      | party   | market id    | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov  | ETH/USD-1-1  | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov  | ETH/USD-1-1  | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
      | lpprov  | ETH/USD-2-10 | 50000     | 10000                | buy  | BID              | 100000 | 10     |
      | lpprov  | ETH/USD-2-10 | 50000     | 10000                | sell | ASK              | 100000 | 10     |
      | lpprov2 | ETH/USD-1-2  | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov2 | ETH/USD-1-2  | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
    When the parties place the following orders:
      | party | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD-1-1  | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD-1-1  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-1  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-1  | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD-2-10 | buy  | 1      | 9900  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD-2-10 | buy  | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-2-10 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-2-10 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/USD-1-2  | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/USD-1-2  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux4  | ETH/USD-1-2  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux4  | ETH/USD-1-2  | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/USD-1-1"
    And the opening auction period ends for market "ETH/USD-2-10"
    Then the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-1-1"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-1-2"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-2-10"

    # Create the referral set and codes
    Given the parties create the following referral codes:
      | party     | code            |
      | referrer1 | referral-code-1 |
    And the parties apply the following referral codes:
      | party    | code            |
      | referee1 | referral-code-1 |
      | referee2 | referral-code-1 |


  Scenario Outline: Referral set member is the maker of a trade during continuous trading (0083-RFPR-048)
    # Expectation: the members taker volume is not increased, and thus we see no increase in the sets taker volume

    Given the parties place the following orders:
      | party   | market id   | side         | volume | price | resulting trades | type       | tif     |
      | <party> | ETH/USD-1-1 | <maker side> | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/USD-1-1 | <taker side> | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch "1" should have a running volume of 0:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0                   | 0                   | 0                       | 0                     | 0                     | 0                         |
      | referee2 | 0                   | 0                   | 0                       | 0                     | 0                     | 0                         |

    # Check volume not counted for either referrer or referee contributions or for either buy or sell orders
    Examples:
      | party     | maker side | taker side |
      | referrer1 | buy        | sell       |
      | referrer1 | sell       | buy        |
      | referee1  | buy        | sell       |
      | referee1  | sell       | buy        |


  Scenario Outline: Referral set member is the taker of a trade during continuous trading (0083-RFPR-031)
    # Expectation: the members taker volume is increased, and thus we see an increase in the sets taker volume

    Given the parties place the following orders:
      | party   | market id   | side         | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD-1-1 | <maker side> | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USD-1-1 | <taker side> | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch "1" should have a running volume of 100000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |
      | referee2 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |

    # Check volume counted for both referrer and referee contributions and both buy and sell orders
    Examples:
      | party     | maker side | taker side |
      | referrer1 | buy        | sell       |
      | referrer1 | sell       | buy        |
      | referee1  | buy        | sell       |
      | referee1  | sell       | buy        |


  Scenario Outline: Referral set member participates in a trade when exiting an auction (0083-RFPR-032)
    # Expectation: the members taker volume is not increased, and thus we see no increase in the sets taker volume

    # Cancel the liquidity commitment triggering an auction

    Given the market data for the market "ETH/USD-1-2" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | horizon | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS | 35569        | 10000000       | 1             | 3600    | 973       | 1027      |
    When the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     |
      | ptbuy  | ETH/USD-1-2 | buy  | 2      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
      | ptsell | ETH/USD-1-2 | sell | 2      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/USD-1-2" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 103505       | 10000000       | 1             | 3           |
    And the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/USD-1-2"

    When the parties place the following orders:
      | party   | market id   | side         | volume | price | resulting trades | type       | tif     |
      | aux3    | ETH/USD-1-2 | <maker side> | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USD-1-2 | <taker side> | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" epochs
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-1-2"
    And the referral set stats for code "referral-code-1" at epoch "1" should have a running volume of 0:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0                   | 0                   | 0                       | 0                     | 0                     | 0                         |
      | referee2 | 0                   | 0                   | 0                       | 0                     | 0                     | 0                         |

    # Check volume not counted for either referrer or referee contributions or for either buy or sell orders
    Examples:
      | party     | maker side | taker side |
      | referrer1 | buy        | sell       |
      | referrer1 | sell       | buy        |
      | referee1  | buy        | sell       |
      | referee1  | sell       | buy        |


  Scenario Outline: Referral set member has an epoch taker volume greater than the cap (0083-RFPR-034)
    # Expectation: the sets running volume is capped to the network parameter

    Given the following network parameters are set:
      | name                                                    | value |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 1000  |
    And the parties place the following orders:
      | party   | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD-1-1 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USD-1-1 | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch "1" should have a running volume of 1000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |
      | referee2 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |

    # Check the correct cap is applied for both referrer and referee contributions
    Examples:
      | party     |
      | referrer1 |
      | referee1  |


  Scenario: Referral set member has an epoch taker volume greater than the cap, but the cap is increased before the end of the epoch (0083-RFPR-050)
    # Expectation: the sets running volume is capped to the most recent network parameter

    Given the following network parameters are set:
      | name                                                    | value |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 1000  |
    And the parties place the following orders:
      | party   | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD-1-1 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USD-1-1 | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the following network parameters are set:
      | name                                                    | value |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 5000  |
    And the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch "1" should have a running volume of 5000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |
      | referee2 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |

    # Check the correct cap is applied for both referrer and referee contributions
    Examples:
      | party     |
      | referrer1 |
      | referee1  |


  Scenario Outline: Referral set member participates in multiple markets with different quantum settlement assets (0083-RFPR-049)
    # Expectation: the members taker volume is increased correctly, and thus we see an increase in the sets taker volume

    Given the parties place the following orders:
      | party   | market id    | side         | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USD-1-1  | <maker side> | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USD-1-1  | <taker side> | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/USD-2-10 | <maker side> | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USD-2-10 | <taker side> | 10     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch "1" should have a running volume of 20000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |
      | referee2 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |

    # Check volume is counted for both referrer and referee contributions and for both buy and sell orders
    Examples:
      | party     | maker side | taker side |
      | referrer1 | buy        | sell       |
      | referrer1 | sell       | buy        |
      | referee1  | buy        | sell       |
      | referee1  | sell       | buy        |


  Scenario Outline: Multiple referees generate volume in multiple markets with different settlement assets (and quantum) (0083-RFPR-033)
    # Expectation: each members taker volume is increased correctly, and thus we see an increase in the sets taker volume

    Given the parties place the following orders:
      | party     | market id    | side         | volume | price | resulting trades | type       | tif     |
      | aux1      | ETH/USD-1-1  | <maker side> | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USD-1-1  | <taker side> | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USD-1-1  | <maker side> | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1  | ETH/USD-1-1  | <taker side> | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USD-1-1  | <maker side> | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee2  | ETH/USD-1-1  | <taker side> | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USD-2-10 | <maker side> | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USD-2-10 | <taker side> | 10     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USD-2-10 | <maker side> | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1  | ETH/USD-2-10 | <taker side> | 10     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USD-2-10 | <maker side> | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | referee2  | ETH/USD-2-10 | <taker side> | 10     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch "1" should have a running volume of 60000:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |
      | referee2 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |

    # Check volume is counted for both referrer and referee contributions and for both buy and sell orders
    Examples:
      | maker side | taker side |
      | buy        | sell       |
      | sell       | buy        |


  Scenario Outline: Members generate consistent taker volume over a number of epochs (0083-RFPR-035)
    # Expectation: only epoch taker volume from the last n epochs should contribute to the sets running taker volume

    Given the referral program:
      | end of program       | window length   | benefit tiers | staking tiers |
      | 2023-12-12T12:12:12Z | <window length> | rbt           | rst           |

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD-1-1 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD-1-1 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch "1" should have a running volume of <running volume 1>:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |
      | referee2 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD-1-1 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD-1-1 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch "2" should have a running volume of <running volume 2>:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |
      | referee2 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD-1-1 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD-1-1 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch "3" should have a running volume of <running volume 3>:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |
      | referee2 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |

    Given the parties place the following orders:
      | party    | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USD-1-1 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USD-1-1 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch "4" should have a running volume of <running volume 4>:
      | party    | reward infra factor | reward maker factor | reward liquidity factor | discount infra factor | discount maker factor | discount liquidity factor |
      | referee1 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |
      | referee2 | 0.11                | 0.12                | 0.13                    | 0.14                  | 0.15                  | 0.16                      |

    # Check running volume correctly calculated for a variety of window lengths
    Examples:
      | window length | running volume 1 | running volume 2 | running volume 3 | running volume 4 |
      | 1             | 10000            | 20000            | 10000            | 10000            |
      | 2             | 10000            | 30000            | 30000            | 20000            |
      | 3             | 10000            | 30000            | 40000            | 40000            |
