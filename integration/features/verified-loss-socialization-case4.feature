Feature: Test loss socialization case 4

  Background:
    Given the insurance pool initial balance for the markets is "2900":
    And the execution engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading | oracleSpecPubKeys    | oracleSpecProperty | oracleSpecPropertyType | oracleSpecBinding |
      | ETH/DEC19 | ETH      | BTC       | BTC   |       100 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |            1.4 |            1.2 |           1.1 |              42 | 0           | continuous   |        0 |                 0 |            0 |                 0  |                |             |                 | 0.1             | 0xDEADBEEF,0xCAFEDOOD| prices.ETH.value   | TYPE_INTEGER           | prices.ETH.value  |

  Scenario: case 4 from https://docs.google.com/spreadsheets/d/1CIPH0aQmIKj6YeFW9ApP_l-jwB4OcsNQ/edit#gid=1555964910
# setup accounts
    Given the following traders:
      | name             |    amount |
      | sellSideProvider | 100000000 |
      | buySideProvider  | 100000000 |
      | trader1          |      2000 |
      | trader2          |     10000 |
      | trader3          |      3000 |
      | trader4          |     10000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |
      | trader3          | BTC   |
      | sellSideProvider | BTC   |
      | buySideProvider  | BTC   |
# setup orderbook
    Then traders place following orders with references:
      | trader           | id        | type | volume | price | resulting trades | type  | tif | reference       |
      | sellSideProvider | ETH/DEC19 | sell |   1000 |   120 |                0 | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  |   1000 |    80 |                0 | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
# trade 1 occur
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     30 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  |     30 |   100 |                1 | TYPE_LIMIT | TIF_GTC |
# trade 2 occur
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader3 | ETH/DEC19 | sell |     60 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  |     60 |   100 |                1 | TYPE_LIMIT | TIF_GTC |
# trade 3 occur
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader3 | ETH/DEC19 | sell |     10 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader4 | ETH/DEC19 | buy  |     10 |   100 |                1 | TYPE_LIMIT | TIF_GTC |

# order book volume change
    Then traders cancels the following orders reference:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
      | buySideProvider  | buy-provider-1  |
    Then traders place following orders with references:
      | trader           | id        | type | volume | price | resulting trades | type  | tif | reference       |
      | sellSideProvider | ETH/DEC19 | sell |   1000 |   300 |                0 | TYPE_LIMIT | TIF_GTC | sell-provider-2 |
      | buySideProvider  | ETH/DEC19 | buy  |   1000 |    80 |                0 | TYPE_LIMIT | TIF_GTC | buy-provider-2  |

# trade 4 occur
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader2 | ETH/DEC19 | buy  |     10 |   180 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader4 | ETH/DEC19 | sell |     10 |   180 |                1 | TYPE_LIMIT | TIF_GTC |

# check positions
    Then position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |      0 |             0 |       -2000 |
      | trader2 |    100 |          7200 |         -90 |
      | trader3 |      0 |             0 |       -3000 |
      | trader4 |      0 |               |         790 |
    And the insurance pool balance is "0" for the market "ETH/DEC19"
