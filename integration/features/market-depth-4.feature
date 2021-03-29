Feature: Test market depth events for pegged orders (cancelling pegged orders)

  Background:
    Given the insurance pool initial balance for the markets is "0":

    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 1                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Check order events with larger pegged orders, and lower balance
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount    |
      | sellSideProvider | BTC   | 100000000 |
      | buySideProvider  | BTC   | 100000000 |
      | pegged1          | BTC   | 50000     |
      | pegged2          | BTC   | 50000     |
      | pegged3          | BTC   | 50000     |
      | pegged4          | BTC   | 50000     |
      | aux              | BTC   | 100000000 |
      | aux2             | BTC   | 100000000 |
# setup pegged orders
    When traders place pegged orders:
      | trader  | market id | side | volume | reference | offset | price |
      | pegged1 | ETH/DEC19 | sell | 500    | ASK       | 10     | 100   |
      | pegged2 | ETH/DEC19 | sell | 500    | MID       | 15     | 100   |
      | pegged3 | ETH/DEC19 | buy  | 500    | BID       | -10    | 100   |
      | pegged4 | ETH/DEC19 | buy  | 500    | MID       | -10    | 100   |
    Then I see the following order events:
      | trader  | market id | side | volume | reference | offset | price | status        |
      | pegged1 | ETH/DEC19 | sell | 500    | ASK       | 10     | 100   | STATUS_PARKED |
      | pegged2 | ETH/DEC19 | sell | 500    | MID       | 15     | 100   | STATUS_PARKED |
      | pegged3 | ETH/DEC19 | buy  | 500    | BID       | -10    | 100   | STATUS_PARKED |
      | pegged4 | ETH/DEC19 | buy  | 500    | MID       | -10    | 100   | STATUS_PARKED |
# keep things simple: remove the events we've just verified
    Given clear order events
# setup orderbook
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux              | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux2             | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |
    Then I see the following order events:
      | trader           | market id | side | volume | reference | offset | price | status        |
      | sellSideProvider | ETH/DEC19 | sell | 1000   |           | 0      | 120   | STATUS_ACTIVE |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   |           | 0      | 80    | STATUS_ACTIVE |
# Checked out, remove the order events we've checked, now let's have a look at the pegged order events
    When clear order events by reference:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
      | buySideProvider  | buy-provider-1  |
    Then the opening auction period for market "ETH/DEC19" ends
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"
# Now check what happened to our pegged orders
    Then I see the following order events:
      | trader  | market id | side | volume | reference | offset | price | status        |
      | pegged1 | ETH/DEC19 | sell | 500    | ASK       | 10     | 130   | STATUS_ACTIVE |
      | pegged1 | ETH/DEC19 | sell | 500    | ASK       | 10     | 130   | STATUS_ACTIVE |
      | pegged1 | ETH/DEC19 | sell | 500    | ASK       | 10     | 130   | STATUS_ACTIVE |
      | pegged2 | ETH/DEC19 | sell | 500    | MID       | 15     | 115   | STATUS_ACTIVE |
      | pegged3 | ETH/DEC19 | buy  | 500    | BID       | -10    | 70    | STATUS_ACTIVE |
      | pegged4 | ETH/DEC19 | buy  | 500    | MID       | -10    | 90    | STATUS_ACTIVE |
##  Cancel some pegged events, and clear order event buffer so we can ignore the events we checked above
    When traders cancel all their market orders:
      | trader  | market id  |
      | pegged1 | ETH/DEC19 |
      | pegged3 | ETH/DEC19 |
      | pegged2 | ETH/DEC19 |

    Then I see the following order events:
      | trader  | market id | side | volume | reference | offset | price | status           |
      | pegged3 | ETH/DEC19 | buy  | 500    | BID       | -10    | 70    | STATUS_CANCELLED |
      | pegged1 | ETH/DEC19 | sell | 500    | ASK       | 10     | 130   | STATUS_CANCELLED |
      | pegged2 | ETH/DEC19 | sell | 500    | MID       | 15     | 115   | STATUS_CANCELLED |
