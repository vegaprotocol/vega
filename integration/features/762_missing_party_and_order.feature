Feature: Test crash on cancel of missing order

  Background:
    Given the insurance pool initial balance for the markets is "0":
 #   And the markets starts on "2019-11-30T00:00:00Z" and expires on "2019-12-31T23:59:59Z"
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       100 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 | 0           | continuous   |        0 |                 0 |            0 |

  Scenario: A non-existent party attempts to place an order
    Given missing traders place following orders with references:
      | trader        | id        | type | volume | price | resulting trades | type  | tif | reference     |
      | missingTrader | ETH/DEC19 | sell |   1000 |   120 |                0 | TYPE_LIMIT | TIF_GTC | missing-ref-1 |
    Then missing traders cancels the following orders reference:
      | trader        | reference     |
      | missingTrader | missing-ref-1 |
    Given missing traders place following orders with references:
      | trader        | id        | type | volume | price | resulting trades | type  | tif | reference     |
      | missingTrader | ETH/DEC19 | sell |   1000 |   120 |                0 | TYPE_LIMIT | TIF_GTC | missing-ref-2 |
    Then missing traders cancels the following orders reference:
      | trader        | reference     |
      | missingTrader | missing-ref-2 |
