Feature: Regression test for issue 596

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short              | mu | r     | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading | oracleSpecPubKeys     | oracleSpecProperty | oracleSpecPropertyType | oracleSpecBinding |
      | ETH/DEC19 | ETH      | BTC       | BTC   | 100       | forward    | 0.001     | 0.00011407711613050422 | 0  | 0.016 | 2.0   | 1.4            | 1.2            | 1.1           | 42              | 0           | continuous   | 0        | 0                 | 0            | 0                  |                |             |                 | 0.1             | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value   | TYPE_INTEGER           | prices.ETH.value  |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42   |

  Scenario: Traded out position but monies left in margin account
    Given the following traders:
      | name   | amount |
      | edd    |  10000 |
      | barney |  10000 |
      | chris  |  10000 |
      | tamlyn |  10000 |
    Then I Expect the traders to have new general account:
      | name   | asset |
      | edd    | BTC   |
      | barney | BTC   |
      | chris  | BTC   |
      | tamlyn | BTC   |
    And "edd" general accounts balance is "10000"
    And "barney" general accounts balance is "10000"
    And "chris" general accounts balance is "10000"
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
      | edd    | BTC   | ETH/DEC19 |    848 |    9152 |
      | barney | BTC   | ETH/DEC19 |    594 |    9406 |
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type   | tif |
      | chris  | ETH/DEC19 | buy  |     50 |     0 |                3 | TYPE_MARKET | TIF_IOC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    933 |    9007 |
      | chris  | BTC   | ETH/DEC19 |    790 |    9270 |
      | barney | BTC   | ETH/DEC19 |    594 |    9406 |
    And All balances cumulated are worth "40000"
# then chris is trading out
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type   | tif |
      | chris  | ETH/DEC19 | sell |     50 |     0 |                4 | TYPE_MARKET | TIF_IOC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |   1283 |    9007 |
      | chris  | BTC   | ETH/DEC19 |      0 |    9808 |
      | barney | BTC   | ETH/DEC19 |    630 |    9272 |
    And All balances cumulated are worth "40000"
# placing new orders to trade out
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type   | tif |
      | chris  | ETH/DEC19 | buy  |     5  |     0 |                1 | TYPE_MARKET | TIF_IOC |
# placing order which get cancelled
    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference            |
      | chris  | ETH/DEC19 | buy  |     60 |     1 |                0 | TYPE_LIMIT | TIF_GTC | chris-id-1-to-cancel |
# other traders trade together (tamlyn+barney)
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | tamlyn | ETH/DEC19 | sell |      12 |    95 |                1 | TYPE_LIMIT | TIF_GTC |
# cancel order
    Then traders cancels the following orders reference:
      | trader | reference            |
      | chris  | chris-id-1-to-cancel |
# then chris is trading out
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type   | tif |
      | chris  | ETH/DEC19 | sell |     5  |     0 |                2 | TYPE_MARKET | TIF_IOC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | chris  | BTC   | ETH/DEC19 |      0 |    9767 |
    And All balances cumulated are worth "40000"

  Scenario: Traded out position but monies left in margin account if trade which trade out do not update the markprice
    Given the following traders:
      | name   | amount |
      | edd    |  10000 |
      | barney |  10000 |
      | chris  |  10000 |
      | tamlyn |  10000 |
    Then I Expect the traders to have new general account:
      | name   | asset |
      | edd    | BTC   |
      | barney | BTC   |
      | chris  | BTC   |
      | tamlyn | BTC   |
    And "edd" general accounts balance is "10000"
    And "barney" general accounts balance is "10000"
    And "chris" general accounts balance is "10000"
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
      | edd    | BTC   | ETH/DEC19 |    848 |    9152 |
      | barney | BTC   | ETH/DEC19 |    594 |    9406 |
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type   | tif |
      | chris  | ETH/DEC19 | buy  |     50 |     0 |                3 | TYPE_MARKET | TIF_IOC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |    933 |    9007 |
      | chris  | BTC   | ETH/DEC19 |    790 |    9270 |
      | barney | BTC   | ETH/DEC19 |    594 |    9406 |
    And All balances cumulated are worth "40000"
# then chris is trading out
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type   | tif |
      | chris  | ETH/DEC19 | sell |     50 |     0 |                4 | TYPE_MARKET | TIF_IOC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | edd    | BTC   | ETH/DEC19 |   1283 |    9007 |
      | chris  | BTC   | ETH/DEC19 |      0 |    9808 |
      | barney | BTC   | ETH/DEC19 |    630 |    9272 |
    And All balances cumulated are worth "40000"
# placing new orders to trade out
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type   | tif |
      | chris  | ETH/DEC19 | buy  |     5  |     0 |                1 | TYPE_MARKET | TIF_IOC |
# placing order which get cancelled
    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference            |
      | chris  | ETH/DEC19 | buy  |     60 |     1 |                0 | TYPE_LIMIT | TIF_GTC | chris-id-1-to-cancel |
# other traders trade together (tamlyn+barney)
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type  | tif |
      | tamlyn | ETH/DEC19 | sell |      3  |    95 |                1 | TYPE_LIMIT | TIF_GTC |
# cancel order
    Then traders cancels the following orders reference:
      | trader | reference            |
      | chris  | chris-id-1-to-cancel |
# then chris is trading out
    Then traders place following orders:
      | trader | id        | type | volume | price | resulting trades | type   | tif |
      | chris  | ETH/DEC19 | sell |     5  |     0 |                1 | TYPE_MARKET | TIF_IOC |
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | chris  | BTC   | ETH/DEC19 |   0    |    9768 |
    And All balances cumulated are worth "40000"
