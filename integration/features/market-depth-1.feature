Feature: Test market depth events for pegged orders

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | mark price | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading |
      | ETH/DEC19 | BTC        | BTC   | 100        | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 42               | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              |

  Scenario: Ensure the expected order events for pegged orders are produced when mid price changes
# setup accounts
    Given the following traders:
      | name             |    amount |
      | sellSideProvider | 100000000 |
      | buySideProvider  | 100000000 |
      | pegged1          |   5000000 |
      | pegged2          |   5000000 |
      | pegged3          |   5000000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | pegged1          | BTC   |
      | pegged2          | BTC   |
      | pegged3          | BTC   |
      | sellSideProvider | BTC   |
      | buySideProvider  | BTC   |
# setup pegged orders
    Then traders place pegged orders:
      | trader   | id        | side | volume | reference | offset | price |
      | pegged1  | ETH/DEC19 | sell |     10 | MID       | 10     | 100   |
      | pegged2  | ETH/DEC19 | buy  |      5 | MID       | -15    | 100   |
      | pegged3  | ETH/DEC19 | buy  |      5 | MID       | -10    | 100   |
    Then I see the following order events:
      | trader   | id        | side | volume | reference | offset | price | status        |
      | pegged1  | ETH/DEC19 | sell |     10 | MID       | 10     | 100   | STATUS_PARKED |
      | pegged2  | ETH/DEC19 | buy  |      5 | MID       | -15    | 100   | STATUS_PARKED |
      | pegged3  | ETH/DEC19 | buy  |      5 | MID       | -10    | 100   | STATUS_PARKED |
# keep things simple: remove the events we've just verified
    And clear order events
    Then traders place following orders with references:
      | trader           | id        | type | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell |   1000 |   120 |                0 | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  |   1000 |    80 |                0 | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
    Then I see the following order events:
      | trader            | id        | side | volume | reference | offset | price | status        |
      | sellSideProvider  | ETH/DEC19 | sell |   1000 |           | 0      | 120   | STATUS_ACTIVE |
      | buySideProvider   | ETH/DEC19 | buy  |   1000 |           | 0      | 80    | STATUS_ACTIVE |
# Checked out, remove the order events we've checked, now let's have a look at the pegged order events
    And clear order events by reference:
      | trader            | reference       |
      | sellSideProvider  | sell-provider-1 |
      | buySideProvider   | buy-provider-1  |
# Now check what happened to our pegged orders
    Then I see the following order events:
      | trader   | id        | side | volume | reference | offset | price | status          |
      | pegged1  | ETH/DEC19 | sell |     10 | MID       | 10     | 110   | STATUS_ACTIVE   |
      | pegged2  | ETH/DEC19 | buy  |      5 | MID       | -15    | 85    | STATUS_ACTIVE   |
      | pegged3  | ETH/DEC19 | buy  |      5 | MID       | -10    | 90    | STATUS_ACTIVE   |
