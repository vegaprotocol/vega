Feature: Test mark to market settlement

  Background:
    Given the markets starts on "2019-11-30T00:00:00Z" and expires on "2019-12-31T23:59:59Z"
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/short | tau/long | mu | r | sigma | release factor | initial factor | search factor | settlement price |
      | ETH/DEC19 | BTC      | ETH       | ETH   |      1000 | simple     |       0.11 |      0.1 |  0 | 0 |     0 |            1.4 |            1.2 |           1.1 |               42 |

  Scenario: Order cannot be placed once the market is expired
    Given the following traders:
      | name    | amount |
      | trader1 |  10000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
    And "trader1" general accounts balance is "10000"
    Then the time is updated to "2020-01-01T01:01:01Z"
    Then traders cannot place the following orders anymore:
      | trader  | id        | type | volume | price | resulting trades | error           |
      | trader1 | ETH/DEC19 | sell |      1 |  1000 |                0 | OrderError: Invalid Market ID   |

  Scenario: Settlement happened when market is being closed
    Given the following traders:
      | name    | amount |
      | trader1 |  10000 |
      | trader2 |   1000 |
      | trader3 |   5000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader2 | ETH   |
      | trader3 | ETH   |
    And "trader1" general accounts balance is "10000"
    And "trader2" general accounts balance is "1000"
    And "trader3" general accounts balance is "5000"
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |      2 |  1000 |                0 | LIMIT | GTC |
      | trader2 | ETH/DEC19 | buy  |      1 |  1000 |                1 | LIMIT | GTC |
      | trader3 | ETH/DEC19 | buy  |      1 |  1000 |                1 | LIMIT | GTC |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    240 |    9760 |
      | trader2 | ETH   | ETH/DEC19 |    132 |     868 |
      | trader3 | ETH   | ETH/DEC19 |    132 |    4868 |
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    And All balances cumulated are worth "16000"
    Then the time is updated to "2020-01-01T01:01:01Z"
    Then traders cannot place the following orders anymore:
      | trader  | id        | type | volume | price | resulting trades | error                         |
      | trader1 | ETH/DEC19 | sell |      1 |  1000 |                0 | OrderError: Invalid Market ID |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |      0 |   10718 |
      | trader2 | ETH   | ETH/DEC19 |      0 |      42 |
      | trader3 | ETH   | ETH/DEC19 |      0 |    4042 |
    And All balances cumulated are worth "14802"
