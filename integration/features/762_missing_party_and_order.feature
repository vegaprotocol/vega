Feature: Test crash on cancel of missing order

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-margin-calculator | 0                | default-none | default-none     | default-eth-for-future |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: A non-existent party attempts to place an order
    When the traders place the following orders:
      | trader        | market id | side | volume | price | resulting trades | type       | tif     | reference     | error                 |
      | missingTrader | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | missing-ref-1 | trader does not exist |
    When the traders cancel the following orders:
      | trader        | reference     | error |
      | missingTrader | missing-ref-1 | unable to find the order in the market |
    When the traders place the following orders:
      | trader        | market id | side | volume | price | resulting trades | type       | tif     | reference     | error                 |
      | missingTrader | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | missing-ref-2 | trader does not exist |
    When the traders cancel the following orders:
      | trader        | reference     | error |
      | missingTrader | missing-ref-2 | unable to find the order in the market |
