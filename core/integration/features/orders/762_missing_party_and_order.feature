Feature: Test crash on cancel of missing order

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-margin-calculator | 0                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |

  Scenario: A non-existent party attempts to place an order
    When the parties place the following orders:
      | party         | market id | side | volume | price | resulting trades | type       | tif     | reference     | error                                   |
      | missingTrader | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | missing-ref-1 | party has insufficient balance in asset |
    When the parties cancel the following orders:
      | party         | reference     | error                                  |
      | missingTrader | missing-ref-1 | unable to find the order in the market |
    When the parties place the following orders:
      | party         | market id | side | volume | price | resulting trades | type       | tif     | reference     | error                                   |
      | missingTrader | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | missing-ref-2 | party has insufficient balance in asset |
    When the parties cancel the following orders:
      | party         | reference     | error                                  |
      | missingTrader | missing-ref-2 | unable to find the order in the market |
