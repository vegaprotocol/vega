Feature: Referral benefits set correctly

  Background:

    Given time is updated to "2023-01-01T00:00:00Z"

    # Initalise the referral program then move forwards an epoch to start the program
    Given the referral benefit tiers "rbt":
      | minimum running notional taker volume | minimum epochs | referral reward factor | referral discount factor |
      | 100                                   | 1              | 0.1                    | 0.1                      |
      | 500                                   | 2              | 0.2                    | 0.2                      |
      | 3000                                  | 3              | 0.3                    | 0.3                      |
    And the referral staking tiers "rst":
      | minimum staked tokens | referral reward multiplier |
      | 1                     | 1                          |
      | 2                     | 2                          |
      | 3                     | 3                          |
    And the referral program:
      | end of program       | window length | benefit tiers | staking tiers |
      | 2023-12-12T12:12:12Z | 7             | rbt           | rst           |
    And the network moves ahead "1" epochs

    # Initialise the markets and network parameters
    Given the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.infrastructureFee    | 0.001 |
      | market.fee.factors.makerFee             | 0.001 |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.auction.minimumDuration          | 1     |
      | validators.epoch.length                 | 5s    |
      | limits.markets.maxPeggedOrders          | 4     |
    And the following assets are registered:
      | id   | decimal places |
      | USDT | 1              |
      | USDC | 2              |
    And the markets:
      | id       | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/USDT | ETH        | USDT  | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures |
      | ETH/USDC | ETH        | USDC  | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures |
    
    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party     | asset | amount      |
      | lpprov    | USDC  | 10000000000 |
      | aux1      | USDC  | 100000      |
      | aux2      | USDC  | 100000      |
      | referrer1 | USDC  | 100000      |
      | referee1  | USDC  | 100000      |
      | referee2  | USDC  | 100000      |
      | lpprov    | USDT  | 10000000000 |
      | aux1      | USDT  | 100000      |
      | aux2      | USDT  | 100000      |
      | referrer1 | USDT  | 100000      |
      | referee1  | USDT  | 100000      |
      | referee2  | USDT  | 100000      |

    # Exit the opening auction
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | lp type    |
      | lp1 | lpprov | ETH/USDT  | 1000000           | 0.01 | submission |
      | lp2 | lpprov | ETH/USDC  | 10000000          | 0.01 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/USDT  | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/USDT  | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
      | lpprov | ETH/USDC  | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/USDC  | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USDT  | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USDC  | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USDC  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDC  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDC  | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USDT"
    And the opening auction period ends for market "ETH/USDC"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USDT"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USDC"


  Scenario: Referee is the taker of a buy order during continuous trading
    
    # Set up a referral set in the 1st epoch
    Given the current epoch is "1"
    And the parties create the following referral codes:
      | party     | code            |
      | referrer1 | referral-code-1 |
    And the parties apply the following referral codes:
      | party    | code            |
      | referee1 | referral-code-1 |
    # In the 1st epoch generate taker volume, move to the end of the epoch and check the volume
    When the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | referee1 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" epochs
    And the current epoch is "2"
    # ISSUE: Would expect to see the referees taker volume contribute to the sets running volume here
    Then the referral set stats for code "referral-code-1" at epoch 1 should have a running volume of 0:
      | party    | reward factor | discount factor |
      | referee1 | 0             | 0               |
    When the network moves ahead "1" epochs
    # ISSUE: Would expect to see the referees taker volume contribute to the sets running volume here
    Then the referral set stats for code "referral-code-1" at epoch 2 should have a running volume of 0:
      | party    | reward factor | discount factor |
      | referee1 | 0             | 0               |