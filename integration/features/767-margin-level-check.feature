Feature: Regression test for issue 767

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long |               tau/short | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading | oracleSpecPubKeys    | oracleSpecProperty | oracleSpecPropertyType | oracleSpecBinding |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       100 | forward    |      0.001 | 0.00011407711613050422 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 | 0           | continuous   |        0 |                 0 |            0 |                 0  |                |             |                 | 0.1             | 0xDEADBEEF,0xCAFEDOOD| prices.ETH.value   | TYPE_INTEGER           | prices.ETH.value  |

  Scenario: Traders place orders meeting the maintenance margin, but not the initial margin requirements, and can close out
    Given the following traders:
      | name   | amount |
      | edd    |   1000 |
      | barney |   1000 |
    Then I Expect the traders to have new general account:
      | name   | asset |
      | edd    | BTC   |
      | barney | BTC   |
    And "edd" general accounts balance is "1000"
    And "barney" general accounts balance is "1000"
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | edd    | ETH/DEC19 | sell |     20 |   101 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     20 |   102 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     10 |   103 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     15 |   104 |                0 | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell |     30 |   105 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     20 |    99 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     12 |    98 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     14 |    97 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     20 |    96 |                0 | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  |     5  |    95 |                0 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    848 |     152 |
      | barney | BTC   | ETH/DEC19 |    594 |     406 |
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | edd    | ETH/DEC19 | sell |     20 |   101 |                0 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |   1000 |       0 |
      | barney | BTC   | ETH/DEC19 |    594 |     406 |
    And All balances cumulated are worth "2000"
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | edd    | ETH/DEC19 | buy  |    115 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |   1000 |       0 |
      | barney | BTC   | ETH/DEC19 |    594 |     406 |
    And All balances cumulated are worth "2000"
