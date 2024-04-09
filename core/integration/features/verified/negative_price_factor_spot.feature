Feature: Simple negative price factor - spot markets

    Test simply checks if an asset and market is configured such that the market
    decimal places are greater than the asset decimal places, yielding a
    negative price factor, the market still operates correctly.

    This is validated by checking an LP meets their obligation when their orders
    are sized and priced correctly. i.e. they are not penalised at the end of
    the epoch.

  Background:

    Given time is updated to "2024-01-01T00:00:00Z"
    And the average block duration is "1"
    And the following network parameters are set:
      | name                                             | value |
      | validators.epoch.length                          | 10s   |
      | market.liquidity.providersFeeCalculationTimeStep | 5s    |

    Given the following assets are registered:
      | id       | decimal places | quantum |
      | USDT.0.1 | 0              | 1       |
      | BTC.0.1  | 0              | 1       |
    And the spot markets:
      | id       | name     | base asset | quote asset | risk model                    | auction duration | fees         | price monitoring | sla params    | decimal places | position decimal places |
      | BTC/USDT | ETH/USDT | BTC.0.1    | USDT.0.1    | default-log-normal-risk-model | 1                | default-none | default-none     | default-basic | 2              | -2                      |

      
    Given the parties deposit on asset's general account the following amount:
      | party | asset    | amount     |
      | lp1   | USDT.0.1 | 1000000000 |
      | aux1  | USDT.0.1 | 1000000000 |
      | aux2  | USDT.0.1 | 1000000000 |
      | lp1   | BTC.0.1  | 1000000000 |
      | aux1  | BTC.0.1  | 1000000000 |
      | aux2  | BTC.0.1  | 1000000000 |

    Given the parties deposit on asset's general account the following amount:
      | party | asset    | amount     |
      | lp1   | USDT.0.1 | 1000000000 |
      | aux1  | USDT.0.1 | 1000000000 |
      | aux2  | USDT.0.1 | 1000000000 |


  Scenario: Check if a futures / perpetual market operates correctly with a negative price factor

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/USDT  | 200               | 0.1 | submission |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1   | BTC/USDT  | buy  | 2      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1   | BTC/USDT  | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/USDT"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/USDT"

    Given the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 1      | 150   | 1                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                   | to account           | market id | amount | asset    |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL | BTC/USDT  | 10     | USDT.0.1 |


  Scenario: Check if a spot market operates correctly with a negative price factor

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/USDT  | 200               | 0.1 | submission |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1   | BTC/USDT  | buy  | 2      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1   | BTC/USDT  | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/USDT"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/USDT"

    Given the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 1      | 150   | 1                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                   | to account           | market id | amount | asset    |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL | BTC/USDT  | 10     | USDT.0.1 |
