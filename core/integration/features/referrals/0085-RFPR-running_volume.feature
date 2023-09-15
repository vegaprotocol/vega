Feature: Calculating referral set running volumes

  Background:

    # Initialise the network
    Given time is updated to "2023-01-01T00:00:00Z"
    And the average block duration is "1"
    And the following network parameters are set:
      | name                                                    | value      |
      | market.fee.factors.infrastructureFee                    | 0.001      |
      | market.fee.factors.makerFee                             | 0.001      |
      | network.markPriceUpdateMaximumFrequency                 | 0s         |
      | market.auction.minimumDuration                          | 1          |
      | validators.epoch.length                                 | 20s        |
      | limits.markets.maxPeggedOrders                          | 4          |
      | referralProgram.minStakedVegaTokens                     | 0          |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 1000000000 |

    # Initalise the referral program
    Given the referral benefit tiers "rbt":
      | minimum running notional taker volume | minimum epochs | referral reward factor | referral discount factor |
      | 100                                   | 1              | 0.1                    | 0.1                      |
    And the referral staking tiers "rst":
      | minimum staked tokens | referral reward multiplier |
      | 1                     | 1                          |
    And the referral program:
      | end of program       | window length | benefit tiers | staking tiers |
      | 2023-12-12T12:12:12Z | 7             | rbt           | rst           |
    And the network moves ahead "1" epochs

    # Initialise the markets
    And the following assets are registered:
      | id   | decimal places |
      | USDT | 1              |
      | USDC | 2              |
    And the markets:
      | id       | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USDT | ETH        | USDT  | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
      | ETH/USDC | ETH        | USDC  | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 1              | 1                       |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 3600s       | 1              |
    When the markets are updated:
      | id       | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/USDT | lqm-params           | 1e-3                   | 0                         |
      | ETH/USDC | lqm-params           | 1e-3                   | 0                         |

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party     | asset | amount      |
      | lpprov    | USDC  | 10000000000 |
      | aux1      | USDC  | 10000000    |
      | aux2      | USDC  | 10000000    |
      | referrer1 | USDC  | 10000000    |
      | referee1  | USDC  | 10000000    |
      | referee2  | USDC  | 10000000    |
      | lpprov    | USDT  | 10000000000 |
      | aux1      | USDT  | 10000000    |
      | aux2      | USDT  | 10000000    |
      | referrer1 | USDT  | 10000000    |
      | referee1  | USDT  | 10000000    |
      | referee2  | USDT  | 10000000    |

    # Exit opening auctions
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | lp type    |
      | lp1 | lpprov | ETH/USDT  | 1000000           | 0.01 | submission |
      | lp2 | lpprov | ETH/USDC  | 10000000          | 0.01 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/USDT  | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/USDT  | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
      | lpprov | ETH/USDC  | 50000     | 10000                | buy  | BID              | 100000 | 10     |
      | lpprov | ETH/USDC  | 50000     | 10000                | sell | ASK              | 100000 | 10     |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USDT  | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USDC  | buy  | 1      | 9900  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USDC  | buy  | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDC  | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDC  | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USDT"
    And the opening auction period ends for market "ETH/USDC"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USDT"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USDC"

    # Create the referral set and codes
    Given the parties create the following referral codes:
      | party     | code            |
      | referrer1 | referral-code-1 |
    And the parties apply the following referral codes:
      | party    | code            |
      | referee1 | referral-code-1 |
      | referee2 | referral-code-1 |


  Scenario Outline: Referral set member is the maker of a trade during continuous trading
    # Expectation: the members taker volume is not increased, and thus we see no increase in the sets taker volume
    
    Given the parties place the following orders:
      | party   | market id | side         | volume | price | resulting trades | type       | tif     |
      | <party> | ETH/USDT  | <maker side> | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/USDT  | <taker side> | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch 1 should have a running volume of 0:
      | party    | reward factor | discount factor |
      | referee1 | 0             | 0               |
      | referee2 | 0             | 0               |

    # Check volume not counted for either referrer or referee contributions or for either buy or sell orders
    Examples:
      | party     | maker side | taker side |
      | referrer1 | buy        | sell       |
      | referrer1 | sell       | buy        |
      | referee1  | buy        | sell       |
      | referee1  | sell       | buy        |


  Scenario Outline: Referral set member is the taker of a trade during continuous trading
    # Expectation: the members taker volume is increased, and thus we see an increase in the sets taker volume
    
    Given the parties place the following orders:
      | party   | market id | side         | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USDT  | <maker side> | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USDT  | <taker side> | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch 1 should have a running volume of 100000:
      | party    | reward factor | discount factor |
      | referee1 | 0.1           | 0.1             |
      | referee2 | 0.1           | 0.1             |

    # Check volume counted for both referrer and referee contributions and both buy and sell orders
    Examples:
      | party     | maker side | taker side |
      | referrer1 | buy        | sell       |
      | referrer1 | sell       | buy        |
      | referee1  | buy        | sell       |
      | referee1  | sell       | buy        |


  Scenario Outline: Referral set member participates in a trade when exiting an auction
    # Expectation: the members taker volume is not increased, and thus we see no increase in the sets taker volume

    # Cancel the liquidity commitment triggering an auction
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type      |
      | lp1 | lpprov | ETH/USDT  | 0                 | 0.001 | cancellation |
    And the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/USDT"

    When the parties place the following orders:
      | party   | market id | side         | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USDT  | <maker side> | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USDT  | <taker side> | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp2 | lpprov | ETH/USDT  | 1000000           | 0.001 | submission |
    And the network moves ahead "1" epochs
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USDT"
    And the referral set stats for code "referral-code-1" at epoch 1 should have a running volume of 0:
      | party    | reward factor | discount factor |
      | referee1 | 0             | 0               |
      | referee2 | 0             | 0               |

    # Check volume not counted for either referrer or referee contributions or for either buy or sell orders
    Examples:
      | party     | maker side | taker side |
      | referrer1 | buy        | sell       |
      | referrer1 | sell       | buy        |
      | referee1  | buy        | sell       |
      | referee1  | sell       | buy        |


  Scenario Outline: Referral set member has an epoch taker volume greater than the cap
    # Expectation: the sets running volume is capped to the network parameter

    Given the following network parameters are set:
      | name                                                    | value |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 1000  |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch 1 should have a running volume of 1000:
      | party    | reward factor | discount factor |
      | referee1 | 0.1           | 0.1             |
      | referee2 | 0.1           | 0.1             |

    # Check the correct cap is applied for both referrer and referee contributions
    Examples:
      | party     |
      | referrer1 |
      | referee1  |


  Scenario: Referral set member has an epoch taker volume greater than the cap, but the cap is increased before the end of the epoch (AC CODE TO BE ADDED)
    # Expectation: the sets running volume is capped to the most recent network parameter

    Given the following network parameters are set:
      | name                                                    | value |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 1000  |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the following network parameters are set:
      | name                                                    | value |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 5000  |
    And the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch 1 should have a running volume of 5000:
      | party    | reward factor | discount factor |
      | referee1 | 0.1           | 0.1             |
      | referee2 | 0.1           | 0.1             |

    # Check the correct cap is applied for both referrer and referee contributions
    Examples:
      | party     |
      | referrer1 |
      | referee1  |


  Scenario Outline: Referral set member participates in multiple markets with different quantum settlement assets
    # Expectation: the members taker volume is increased correctly, and thus we see an increase in the sets taker volume

    Given the parties place the following orders:
      | party   | market id | side         | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/USDT  | <maker side> | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USDT  | <taker side> | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/USDC  | <maker side> | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | <party> | ETH/USDC  | <taker side> | 10     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch 1 should have a running volume of 110000:
      | party    | reward factor | discount factor |
      | referee1 | 0.1           | 0.1             |
      | referee2 | 0.1           | 0.1             |

    # Check volume is counted for both referrer and referee contributions and for both buy and sell orders
    Examples:
      | party     | maker side | taker side |
      | referrer1 | buy        | sell       |
      | referrer1 | sell       | buy        |
      | referee1  | buy        | sell       |
      | referee1  | sell       | buy        |


  Scenario Outline: Multiple referees generate volume in multiple markets with different settlement assets (and quantum)
    # Expectation: each members taker volume is increased correctly, and thus we see an increase in the sets taker volume

    Given the parties place the following orders:
      | party     | market id | side         | volume | price | resulting trades | type       | tif     |
      | aux1      | ETH/USDT  | <maker side> | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USDT  | <taker side> | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USDT  | <maker side> | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1  | ETH/USDT  | <taker side> | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USDT  | <maker side> | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee2  | ETH/USDT  | <taker side> | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USDC  | <maker side> | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USDC  | <taker side> | 10     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USDC  | <maker side> | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1  | ETH/USDC  | <taker side> | 10     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USDC  | <maker side> | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | referee2  | ETH/USDC  | <taker side> | 10     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch 1 should have a running volume of 330000:
      | party    | reward factor | discount factor |
      | referee1 | 0.1           | 0.1             |
      | referee2 | 0.1           | 0.1             |

    # Check volume is counted for both referrer and referee contributions and for both buy and sell orders
    Examples:
      | maker side | taker side |
      | buy        | sell       |
      | sell       | buy        |


  Scenario Outline: Multiple referees generate volume in multiple markets with different settlement assets (and quantum)
    # Expectation: each members taker volume is increased correctly, and thus we see an increase in the sets taker volume

    Given the parties place the following orders:
      | party     | market id | side         | volume | price | resulting trades | type       | tif     |
      | aux1      | ETH/USDT  | <maker side> | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USDT  | <taker side> | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USDT  | <maker side> | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1  | ETH/USDT  | <taker side> | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USDT  | <maker side> | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee2  | ETH/USDT  | <taker side> | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USDC  | <maker side> | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | referrer1 | ETH/USDC  | <taker side> | 10     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USDC  | <maker side> | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1  | ETH/USDC  | <taker side> | 10     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1      | ETH/USDC  | <maker side> | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | referee2  | ETH/USDC  | <taker side> | 10     | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch 1 should have a running volume of 330000:
      | party    | reward factor | discount factor |
      | referee1 | 0.1           | 0.1             |
      | referee2 | 0.1           | 0.1             |

    # Check volume is counted for both referrer and referee contributions and for both buy and sell orders
    Examples:
      | maker side | taker side |
      | buy        | sell       |
      | sell       | buy        |


  Scenario Outline: Members generate consistent taker volume over a number of epochs
    # Expectation: only epoch taker volume from the last n epochs should contribute to the sets running taker volume

    Given the referral program:
      | end of program       | window length   | benefit tiers | staking tiers |
      | 2023-12-12T12:12:12Z | <window length> | rbt           | rst           |

    Given the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USDT  | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch 1 should have a running volume of <running volume 1>:
      | party    | reward factor | discount factor |
      | referee1 | 0.1           | 0.1             |
      | referee2 | 0.1           | 0.1             |

    Given the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USDT  | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch 2 should have a running volume of <running volume 2>:
      | party    | reward factor | discount factor |
      | referee1 | 0.1           | 0.1             |
      | referee2 | 0.1           | 0.1             |

    Given the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USDT  | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch 3 should have a running volume of <running volume 3>:
      | party    | reward factor | discount factor |
      | referee1 | 0.1           | 0.1             |
      | referee2 | 0.1           | 0.1             |

    Given the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USDT  | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the referral set stats for code "referral-code-1" at epoch 4 should have a running volume of <running volume 4>:
      | party    | reward factor | discount factor |
      | referee1 | 0.1           | 0.1             |
      | referee2 | 0.1           | 0.1             |

    # Check running volume correctly calculated for a variety of window lengths
    Examples:
      | window length | running volume 1 | running volume 2 | running volume 3 | running volume 4 |
      | 1             | 10000            | 10000            | 10000            | 10000            |
      | 2             | 10000            | 20000            | 20000            | 20000            |
      | 3             | 10000            | 20000            | 30000            | 30000            |
