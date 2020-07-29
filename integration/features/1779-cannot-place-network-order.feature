Feature: Cannot place an network order

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu | r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee |
      | ETH/DEC19 | BTC      | ETH       | ETH   |      1000 | simple     |      0.11 |       0.1 |  0 | 0 |     0 |            1.4 |            1.2 |           1.1 |              42 |           0 | continuous   |        0 |                 0 |            0 |

  Scenario: an order is rejected if a trader try to place an order with type NETWORK
    Given the following traders:
      | name    | amount |
      | trader1 |      1 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
    And "trader1" general accounts balance is "1"
    Then traders place following failing orders:
      | trader  | id        | type | volume | price | error              | type    |
      | trader1 | ETH/DEC19 | sell |      1 |  1000 | invalid order type | TYPE_NETWORK |
