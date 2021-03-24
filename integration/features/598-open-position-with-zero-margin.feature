Feature: Regression test for issue 598

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Open position but ZERO in margin account
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount  |
      | edd     | BTC   | 1000    |
      | barney  | BTC   | 1000    |
      | chris   | BTC   | 1000    |
      | trader1 | BTC   | 1000000 |
      | trader2 | BTC   | 1000000 |
      | aux     | BTC   | 1000    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 95    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 105   | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"

    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | edd    | ETH/DEC19 | sell | 10     | 101   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | edd    | ETH/DEC19 | sell | 12     | 102   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | edd    | ETH/DEC19 | sell | 13     | 103   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | edd    | ETH/DEC19 | sell | 14     | 104   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | edd    | ETH/DEC19 | sell | 15     | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | barney | ETH/DEC19 | buy  | 10     | 99    | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | barney | ETH/DEC19 | buy  | 12     | 98    | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | barney | ETH/DEC19 | buy  | 13     | 97    | 0                | TYPE_LIMIT | TIF_GTC | ref-8     |
      | barney | ETH/DEC19 | buy  | 14     | 96    | 0                | TYPE_LIMIT | TIF_GTC | ref-9     |
      | barney | ETH/DEC19 | buy  | 15     | 95    | 0                | TYPE_LIMIT | TIF_GTC | ref-10    |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 571    | 429     |
      | barney | BTC   | ETH/DEC19 | 535    | 465     |
# next instruction will trade with edd
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris  | ETH/DEC19 | buy  | 10     | 0     | 1                | TYPE_MARKET | TIF_IOC | ref-1     |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 571    | 429     |
      | chris  | BTC   | ETH/DEC19 | 109    | 891     |
# next instruction will trade with barney
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | chris  | ETH/DEC19 | sell | 10     | 0     | 1                | TYPE_MARKET | TIF_IOC | ref-1     |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | chris  | BTC   | ETH/DEC19 | 0      | 980     |
      | barney | BTC   | ETH/DEC19 | 535    | 465     |
      | edd    | BTC   | ETH/DEC19 | 591    | 429     |
    Then the margins levels for the traders are:
      | trader | market id | maintenance | search | initial | release |
      | edd    | ETH/DEC19 | 502         | 552    | 602     | 702     |
      | barney | ETH/DEC19 | 451         | 496    | 541     | 631     |
      | chris  | ETH/DEC19 | 0           | 0      | 0       | 0       |
