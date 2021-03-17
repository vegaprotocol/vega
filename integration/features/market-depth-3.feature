Feature: Test market depth events for pegged orders (with BID and ASK price)

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration |  maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 |  BTC        | BTC   |  simple     | 0         | 0         | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 42               | 0                |  0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Ensure the expect order events for pegged orders are produced for all references
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount    |
      | sellSideProvider | BTC   | 100000000 |
      | buySideProvider  | BTC   | 100000000 |
      | pegged1          | BTC   | 5000000   |
      | pegged2          | BTC   | 5000000   |
      | pegged3          | BTC   | 5000000   |
      | pegged4          | BTC   | 5000000   |
# setup pegged orders
    Then traders place pegged orders:
      | trader  | market id | side | volume | reference | offset | price |
      | pegged1 | ETH/DEC19 | sell | 5      | ASK       | 10     | 100   |
      | pegged2 | ETH/DEC19 | sell | 5      | MID       | 15     | 100   |
      | pegged3 | ETH/DEC19 | buy  | 5      | BID       | -10    | 100   |
      | pegged4 | ETH/DEC19 | buy  | 5      | MID       | -10    | 100   |
    Then I see the following order events:
      | trader  | market id | side | volume | reference | offset | price | status        |
      | pegged1 | ETH/DEC19 | sell | 5      | ASK       | 10     | 100   | STATUS_PARKED |
      | pegged2 | ETH/DEC19 | sell | 5      | MID       | 15     | 100   | STATUS_PARKED |
      | pegged3 | ETH/DEC19 | buy  | 5      | BID       | -10    | 100   | STATUS_PARKED |
      | pegged4 | ETH/DEC19 | buy  | 5      | MID       | -10    | 100   | STATUS_PARKED |
# keep things simple: remove the events we've just verified
    And clear order events
    Then traders place following orders with references:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
    Then I see the following order events:
      | trader           | market id | side | volume | reference | offset | price | status        |
      | sellSideProvider | ETH/DEC19 | sell | 1000   |           | 0      | 120   | STATUS_ACTIVE |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   |           | 0      | 80    | STATUS_ACTIVE |
# Checked out, remove the order events we've checked, now let's have a look at the pegged order events
    And clear order events by reference:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
      | buySideProvider  | buy-provider-1  |
# Now check what happened to our pegged orders
    Then I see the following order events:
      | trader  | market id | side | volume | reference | offset | price | status        |
      | pegged1 | ETH/DEC19 | sell | 5      | ASK       | 10     | 130   | STATUS_ACTIVE |
      | pegged1 | ETH/DEC19 | sell | 5      | ASK       | 10     | 130   | STATUS_ACTIVE |
      | pegged1 | ETH/DEC19 | sell | 5      | ASK       | 10     | 130   | STATUS_ACTIVE |
      | pegged2 | ETH/DEC19 | sell | 5      | MID       | 15     | 115   | STATUS_ACTIVE |
      | pegged3 | ETH/DEC19 | buy  | 5      | BID       | -10    | 70    | STATUS_ACTIVE |
      | pegged4 | ETH/DEC19 | buy  | 5      | MID       | -10    | 90    | STATUS_ACTIVE |
