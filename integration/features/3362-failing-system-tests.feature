Feature: Replicate failing system tests after changes to price monitoring (not triggering with FOK orders, auction duration)

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

    ## price bounds are 99771 to 100290
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader1 | ETH/DEC20 | buy  | 1      | 100150 | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 1      | 100150 | 1                | TYPE_LIMIT  | TIF_GTC |
      # | trader1 | ETH/DEC20 | buy  | 1      | 100448 | 0                | TYPE_LIMIT  | TIF_GTC |
      # | trader2 | ETH/DEC20 | sell | 1      | 100448 | 1                | TYPE_LIMIT  | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100150" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader1 | ETH/DEC20 | buy  | 2      | 100213 | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader1 | ETH/DEC20 | buy  | 1      | 100050 | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    # Now place a FOK order that would trigger a price auction (trader 1 has a buy at 95,000 on the book

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | error                                                       |
      | trader2 | ETH/DEC20 | sell | 3      | 0     | 0                | TYPE_MARKET | TIF_FOK | OrderError: non-persistent order trades out of price bounds |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100150" for the market "ETH/DEC20"

    ## Now place the order for the same volume again, but set price to 100,000 -> the buy at 95,000 doesn't uncross
    ## We'll see the mark price move as we've uncrossed with the orders at 100213 and 100050 we've just placed
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader2 | ETH/DEC20 | sell | 1      | 100000 | 1                | TYPE_LIMIT  | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100213" for the market "ETH/DEC20"
