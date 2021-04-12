Feature: Regression test for issue 630

  Background:

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Trader is being closed out.
# setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader           | asset | amount  |
      | sellSideProvider | BTC   | 1000000 |
      | buySideProvider  | BTC   | 1000000 |
      | traderGuy        | BTC   | 240000  |
      | trader1          | BTC   | 1000000 |
      | trader2          | BTC   | 1000000 |
      | aux              | BTC   | 100000  |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"

# setup orderbook
    When the traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 200    | 10000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 200    | 1     | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
    And the cumulated balance for all accounts should be worth "4340000"
    Then the traders should have the following margin levels:
      | trader           | market id | maintenance | search | initial | release |
      | sellSideProvider | ETH/DEC19 | 2000        | 2200   | 2400    | 2800    |
    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the traders should have the following account balances:
      | trader           | asset | market id | margin | general |
      | traderGuy        | BTC   | ETH/DEC19 | 0      | 0       |
      | sellSideProvider | BTC   | ETH/DEC19 | 240000 | 760000  |
    And the insurance pool balance should be "240000" for the market "ETH/DEC19"
    And the cumulated balance for all accounts should be worth "4340000"
