Feature: Replicate failing system tests after changes to price monitoring (trigger lower bounds)

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
      | name                           | value |
      | market.auction.minimumDuration | 1     |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

  Scenario: Replicate test called test_TriggerWithLowerBounds
    Given the traders deposit on asset's general account the following amount:
      | trader   | asset | amount    |
      | trader1  | ETH   | 100000000 |
      | trader2  | ETH   | 100000000 |
      | trader3  | ETH   | 100000000 |
      | traderLP | ETH   | 100000000 |
      | aux      | ETH   | 100000000 |

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA |
      | trader2 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA |
      | trader1 | ETH/DEC20 | buy  | 5      | 95000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 5      | 107000 | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC20 | buy  | 1      | 95000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 1      | 107000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | order side | order reference | order proportion | order offset |
      | lp1 | trader1 | ETH/DEC20 | 16000000          | 0.3 | buy        | BID             | 2                | -10          |
      | lp1 | trader1 | ETH/DEC20 | 16000000          | 0.3 | sell       | ASK             | 13               | 10           |
    Then the mark price should be "0" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    When the opening auction period ends for market "ETH/DEC20"
    Then the mark price should be "100000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    ## price bounds are 99771 to 100290
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | buy  | 1      | 100150 | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 1      | 100150 | 1                | TYPE_LIMIT | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100150" for the market "ETH/DEC20"

    ## This order should breach one price bound (price bounds are 100156 and 100290)
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     |
      | trader2 | ETH/DEC20 | sell | 2      | 100160 | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC20 | buy  | 1      | 100160 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the mark price should be "100150" for the market "ETH/DEC20"

    # after opening auction, market time is 2020-10-16T00:00:02Z (00 seconds + 2x auction duration)
    # The price auction violated 1 bound, so the auction period was 6 seconds, meaning  the auction
    # lasts until 2020-10-16T00:00:08Z, update time to 1 second after:
    When time is updated to "2020-10-16T00:00:09Z"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100160" for the market "ETH/DEC20"
