Feature: Test trader accounts

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu | r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading |
      | ETH/DEC19 | BTC      | ETH       | ETH   |      1000 | simple     |       0.11 |      0.1 |  0 | 0 |     0 |            1.4 |            1.2 |           1.1 |              42 | 0           | continuous   |        0 |                 0 |            0 |                 0  |                |             |                 | 0.1             |
  Scenario: a trader place a new order in the system, margin are calculated
    Given the following traders:
      | name    | amount |
      | trader1 | 10000  |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
    And "trader1" general accounts balance is "10000"
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |      1 |  1000 |                0 | TYPE_LIMIT | TIF_GTC |
    Then the margins levels for the traders are:
      | trader  | id        | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |         100 |    110 |     120 |     140 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    120 |    9880 |

  Scenario: an order is rejected if a trader have insufficient margin
    Given the following traders:
      | name    | amount |
      | trader1 |      1 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
    And "trader1" general accounts balance is "1"
    Then traders place following failing orders:
      | trader  | id        | type | volume | price | error               | type  |
      | trader1 | ETH/DEC19 | sell |      1 |  1000 | margin check failed | TYPE_LIMIT |
    Then the following orders are rejected:
      | trader  | id        | reason                          |
      | trader1 | ETH/DEC19 | ORDER_ERROR_MARGIN_CHECK_FAILED |
