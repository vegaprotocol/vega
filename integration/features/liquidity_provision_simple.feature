Feature: Test LP orders

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu | r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading | oracleSpecPubKeys    | oracleSpecProperty | oracleSpecPropertyType | oracleSpecBinding |
      | ETH/DEC19 | BTC      | ETH       | ETH   |      1000 | simple     |       0.11 |      0.1 |  0 | 0 |     0 |            1.4 |            1.2 |           1.1 |              42 | 0           | continuous   |        0 |                 0 |            0 |                 0  |                |             |                 | 0.1             | 0xDEADBEEF,0xCAFEDOOD| prices.ETH.value   | TYPE_INTEGER           | prices.ETH.value  |
  Scenario: create liquidity provisions
    Given the following traders:
      | name             | amount    |
      | trader1          | 100000000 |
      | sellSideProvider | 100000000 |
      | buySideProvider  | 100000000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | ETH   |
      | sellSideProvider | ETH   |
      | buySideProvider  | ETH   |
    And "trader1" general accounts balance is "100000000"
    Then traders place following orders with references:
      | trader           | id        | type | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell |   1000 |   120 |                0 | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  |   1000 |    80 |                0 | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | trader1          | ETH/DEC19 | buy  |    500 |   110 |                0 | TYPE_LIMIT | TIF_GTC | lp-ref-1        |
      | trader1          | ETH/DEC19 | sell |    500 |   120 |                0 | TYPE_LIMIT | TIF_GTC | lp-ref-2        |
    And dump orders
    Then I see the following order events:
      | trader            | id        | side | volume | reference | offset | price | status        |
      | sellSideProvider  | ETH/DEC19 | sell |   1000 |           | 0      | 120   | STATUS_ACTIVE |
      | buySideProvider   | ETH/DEC19 | buy  |   1000 |           | 0      | 80    | STATUS_ACTIVE |
    And clear order events
    Then the trader submits LP:
      | id  | party   | market    | commitment amount  | fee | order side | order reference | order proportion | order offset |
      | lp1 | trader1 | ETH/DEC19 | 10000              | 0.1   | buy        | BID             | 500              | -10          |
      | lp1 | trader1 | ETH/DEC19 | 10000              | 0.1  | sell       | ASK             | 500              | 10           |
    Then I see the LP events:
      | id  | party   | market    | commitment amount |
      | lp1 | trader1 | ETH/DEC19 | 10000              |
    And dump orders
    Then I see the following order events:
      | trader  | id        | side | volume | reference | offset | price | status        |
      | trader1 | ETH/DEC19 | buy  |    450 |           | 0      | 100   | STATUS_ACTIVE |
      | trader1 | ETH/DEC19 | sell |    308 |           | 0      | 130   | STATUS_ACTIVE |
