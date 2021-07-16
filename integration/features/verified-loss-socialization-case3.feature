Feature: Test loss socialization case 3

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: case 3 from https://docs.google.com/spreadsheets/d/1CIPH0aQmIKj6YeFW9ApP_l-jwB4OcsNQ/edit#gid=1555964910
# setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader           | asset | amount    |
      | sellSideProvider | BTC   | 100000000 |
      | buySideProvider  | BTC   | 100000000 |
      | trader1          | BTC   | 2000      |
      | trader2          | BTC   | 10000     |
      | trader3          | BTC   | 3000      |
      | trader4          | BTC   | 10000     |
      | trader5          | BTC   | 100000000 |
      | trader6          | BTC   | 100000000 |
      | aux1             | BTC   | 100000000 |
      | aux2             | BTC   | 100000000 |

# setup order book
    When the traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | trader5          | ETH/DEC19 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GFA | buy-provider-t5 |
      | trader6          | ETH/DEC19 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GFA | buy-provider-t6 |
      | aux1             | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux2             | ETH/DEC19 | buy  | 1      | 80    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |

    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

# trade 1 occur
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 30     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 30     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
# trade 2 occur
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | sell | 60     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 60     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
# trade 3 occur
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader4 | ETH/DEC19 | buy  | 10     | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# order book volume change
    Then the traders cancel the following orders:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
      | buySideProvider  | buy-provider-1  |
    When the traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 300   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |
    Then the traders cancel the following orders:
      | trader | reference |
      | aux1   | aux-s-1   |
      | aux2   | aux-b-1   |
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

# trade 4 occur
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC19 | buy  | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader4 | ETH/DEC19 | sell | 10     | 180   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

# check positions
    Then the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 0      | 0              | -2000        |
      | trader2 | 100    | 7200           | -2455        |
      | trader3 | 0      | 0              | -3000        |
      | trader4 | 0      | 0              | 528          |
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
