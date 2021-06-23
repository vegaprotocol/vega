Feature: Regression test for issue 596

  Background:

    And the markets:
      | id        | quote name | asset | auction duration | risk model                    | margin calculator         | fees         | oracle config          | price monitoring |
      | ETH/DEC19 | BTC        | BTC   | 1                | default-log-normal-risk-model | default-margin-calculator | default-none | default-eth-for-future | default-none     |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Traded out position but monies left in margin account
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount  |
      | edd     | BTC   | 10000   |
      | barney  | BTC   | 10000   |
      | chris   | BTC   | 10000   |
      | tamlyn  | BTC   | 10000   |
      | trader1 | BTC   | 1000000 |
      | trader2 | BTC   | 1000000 |
      | aux     | BTC   | 1000    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | edd    | ETH/DEC19 | sell | 20     | 101   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | edd    | ETH/DEC19 | sell | 20     | 102   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | edd    | ETH/DEC19 | sell | 10     | 103   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | edd    | ETH/DEC19 | sell | 15     | 104   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | edd    | ETH/DEC19 | sell | 30     | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | barney | ETH/DEC19 | buy  | 20     | 99    | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | barney | ETH/DEC19 | buy  | 12     | 98    | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | barney | ETH/DEC19 | buy  | 14     | 97    | 0                | TYPE_LIMIT | TIF_GTC | ref-8     |
      | barney | ETH/DEC19 | buy  | 20     | 96    | 0                | TYPE_LIMIT | TIF_GTC | ref-9     |
      | barney | ETH/DEC19 | buy  | 5      | 95    | 0                | TYPE_LIMIT | TIF_GTC | ref-10    |
    Then the traders should have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 848    | 9152    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris  | ETH/DEC19 | buy  | 50     | 0     | 3                | TYPE_MARKET | TIF_IOC | ref-1     |
    Then the traders should have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 933    | 9007    |
      | chris  | BTC   | ETH/DEC19 | 790    | 9270    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    And the cumulated balance for all accounts should be worth "2041000"
# then chris is trading out
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris  | ETH/DEC19 | sell | 50     | 0     | 4                | TYPE_MARKET | TIF_IOC | ref-1     |
    Then the traders should have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 1283   | 9007    |
      | chris  | BTC   | ETH/DEC19 | 0      | 9808    |
      | barney | BTC   | ETH/DEC19 | 630    | 9272    |
    And the cumulated balance for all accounts should be worth "2041000"
# placing new orders to trade out
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris  | ETH/DEC19 | buy  | 5      | 0     | 1                | TYPE_MARKET | TIF_IOC | ref-1     |
# placing order which get cancelled
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference            |
      | chris  | ETH/DEC19 | buy  | 60     | 1     | 0                | TYPE_LIMIT | TIF_GTC | chris-id-1-to-cancel |
# other traders trade together (tamlyn+barney)
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tamlyn | ETH/DEC19 | sell | 12     | 95    | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
# cancel order
    Then the traders cancel the following orders:
      | trader | reference            |
      | chris  | chris-id-1-to-cancel |
# then chris is trading out
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris  | ETH/DEC19 | sell | 5      | 0     | 2                | TYPE_MARKET | TIF_IOC | ref-1     |
    Then the traders should have the following account balances:
      | trader | asset | market id | margin | general |
      | chris  | BTC   | ETH/DEC19 | 0      | 9767    |
    And the cumulated balance for all accounts should be worth "2041000"

  Scenario: Traded out position but monies left in margin account if trade which trade out do not update the markprice
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount  |
      | edd     | BTC   | 10000   |
      | barney  | BTC   | 10000   |
      | chris   | BTC   | 10000   |
      | tamlyn  | BTC   | 10000   |
      | trader1 | BTC   | 1000000 |
      | trader2 | BTC   | 1000000 |
      | aux     | BTC   | 1000    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | edd    | ETH/DEC19 | sell | 20     | 101   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | edd    | ETH/DEC19 | sell | 20     | 102   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | edd    | ETH/DEC19 | sell | 10     | 103   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | edd    | ETH/DEC19 | sell | 15     | 104   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | edd    | ETH/DEC19 | sell | 30     | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | barney | ETH/DEC19 | buy  | 20     | 99    | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | barney | ETH/DEC19 | buy  | 12     | 98    | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | barney | ETH/DEC19 | buy  | 14     | 97    | 0                | TYPE_LIMIT | TIF_GTC | ref-8     |
      | barney | ETH/DEC19 | buy  | 20     | 96    | 0                | TYPE_LIMIT | TIF_GTC | ref-9     |
      | barney | ETH/DEC19 | buy  | 5      | 95    | 0                | TYPE_LIMIT | TIF_GTC | ref-10    |
    Then the traders should have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 848    | 9152    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris  | ETH/DEC19 | buy  | 50     | 0     | 3                | TYPE_MARKET | TIF_IOC | ref-1     |
    Then the traders should have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 933    | 9007    |
      | chris  | BTC   | ETH/DEC19 | 790    | 9270    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    And the cumulated balance for all accounts should be worth "2041000"
# then chris is trading out
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris  | ETH/DEC19 | sell | 50     | 0     | 4                | TYPE_MARKET | TIF_IOC | ref-1     |
    Then the traders should have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 1283   | 9007    |
      | chris  | BTC   | ETH/DEC19 | 0      | 9808    |
      | barney | BTC   | ETH/DEC19 | 630    | 9272    |
    And the cumulated balance for all accounts should be worth "2041000"
# placing new orders to trade out
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris  | ETH/DEC19 | buy  | 5      | 0     | 1                | TYPE_MARKET | TIF_IOC | ref-1     |
# placing order which get cancelled
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference            |
      | chris  | ETH/DEC19 | buy  | 60     | 1     | 0                | TYPE_LIMIT | TIF_GTC | chris-id-1-to-cancel |
# other traders trade together (tamlyn+barney)
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tamlyn | ETH/DEC19 | sell | 3      | 95    | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
# cancel order
    Then the traders cancel the following orders:
      | trader | reference            |
      | chris  | chris-id-1-to-cancel |
# then chris is trading out
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris  | ETH/DEC19 | sell | 5      | 0     | 1                | TYPE_MARKET | TIF_IOC | ref-1     |
    Then the traders should have the following account balances:
      | trader | asset | market id | margin | general |
      | chris  | BTC   | ETH/DEC19 | 0      | 9768    |
    And the cumulated balance for all accounts should be worth "2041000"
