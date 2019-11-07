Feature: Test mark to market settlement

  Background:
    Given the executon engine have these markets:
      | name      |baseName | quoteName | asset | markprice | risk model | lamd/short | tau/long | mu | r | sigma | release factor | initial factor | search factor |
      | ETH/DEC19 |BTC      | ETH       | ETH   | 1000      | simple     | 0.11       | 0.1      | 0  | 0 | 0     | 1.4            | 1.2            | 1.1           |
      
  Scenario: a trader is added to the system. A general account is created for each asset
    Given the following traders:
      | name    | amount |
      | trader1 |  10000 |
      | trader2 |  10000 |
      | trader3 |  10000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader2 | ETH   |
      | trader3 | ETH   |
    And "trader1" general accounts balance is "10000"
    And "trader2" general accounts balance is "10000"
    And "trader3" general accounts balance is "10000"
    Then traders place following orders:
      | trader  | id         | type | volume | price | resulting trades |
      | trader1 |  ETH/DEC19 | sell |     1  |  1000 |                0 |
      | trader2 |  ETH/DEC19 | buy  |     1  |  1000 |                1 |
    Then I expect the trader to have a margin:
      | trader  | margin | general |
      | trader1 |   1100 |    8900 |
      | trader2 |   1100 |    8900 |
