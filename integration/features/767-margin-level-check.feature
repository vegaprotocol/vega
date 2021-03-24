Feature: Regression test for issue 767

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
   
  Scenario: Traders place orders meeting the maintenance margin, but not the initial margin requirements, and can close out
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount  |
      | edd     | BTC   | 1000    |
      | barney  | BTC   | 1000    |
      | trader1 | BTC   | 1000000 |
      | trader2 | BTC   | 1000000 |
      | aux     | BTC   | 100000  |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    When traders place the following orders:
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
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 848    | 152     |
      | barney | BTC   | ETH/DEC19 | 594    | 406     |
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | edd    | ETH/DEC19 | sell | 20     | 101   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 1000   | 0       |
      | barney | BTC   | ETH/DEC19 | 594    | 406     |
    And Cumulated balance for all accounts is worth "2102000"
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | edd    | ETH/DEC19 | buy  | 115    | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 1000   | 0       |
      | barney | BTC   | ETH/DEC19 | 594    | 406     |
    And Cumulated balance for all accounts is worth "2102000"
