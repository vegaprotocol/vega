Feature: Replicate failing system tests after changes to price monitoring (trigger with GTT order)

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring updated every "4" seconds named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.95        | 6                 |
      | 10      | 0.99        | 8                 |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the markets:
      | id        | quote name | asset | maturity date        | risk model               | margin calculator         | auction duration | fees         | price monitoring    | oracle config          |
      | ETH/DEC20 | ETH        | ETH   | 2020-12-31T23:59:59Z | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value  |
      | market.auction.minimumDuration | 1      |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

  Scenario: Replicate test called test_TriggerWithMarketOrder
    Given the traders deposit on asset's general account the following amount:
      | trader   | asset | amount    |
      | trader1  | ETH   | 100000000 |
      | trader2  | ETH   | 100000000 |
      | trader3  | ETH   | 100000000 |
      | traderLP | ETH   | 100000000 |
      | aux      | ETH   | 100000000 |

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader1 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT  | TIF_GFA |
      | trader2 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT  | TIF_GFA |
      | trader1 | ETH/DEC20 | buy  | 5      | 95000  | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 5      | 107000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader1 | ETH/DEC20 | buy  | 1      | 95000  | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 1      | 107000 | 0                | TYPE_LIMIT  | TIF_GTC |
    And the traders submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | order side | order reference | order proportion | order offset |
      | lp1 | trader1  | ETH/DEC20 | 16000000          | 0.3 | buy        | BID             | 2                | -10          |
      | lp1 | trader1  | ETH/DEC20 | 16000000          | 0.3 | sell       | ASK             | 13               | 10           |
    Then the mark price should be "0" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    When the opening auction period ends for market "ETH/DEC20"
    Then the mark price should be "100000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader1 | ETH/DEC20 | buy  | 1      | 100150 | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 1      | 100150 | 1                | TYPE_LIMIT  | TIF_GTC |
    ## price bounds are 99771 to 100290 (99845 and 100156)
    And the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | expires in |
      | trader3 | ETH/DEC20 | sell | 1      | 99770  | 0                | TYPE_LIMIT  | TIF_GTT | 6          |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100150" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader2 | ETH/DEC20 | sell | 5      | 99000  | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader1 | ETH/DEC20 | buy  | 1      | 99950  | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the mark price should be "100150" for the market "ETH/DEC20"

    ## We're violating both price ranges, so expect full auction of 14 seconds + 2 of the opening auction
    ## Update time by 15 seconds -> :17Z in total
    ## The mid price uncrossing the highest volume will be 99475
    When time is updated to "2020-10-16T00:00:17Z"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "99475" for the market "ETH/DEC20"
