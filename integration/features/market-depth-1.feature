Feature: Test market depth events for pegged orders

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Ensure the expected order events for pegged orders are produced when mid price changes
# setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader           | asset | amount    |
      | sellSideProvider | BTC   | 100000000 |
      | buySideProvider  | BTC   | 100000000 |
      | pegged1          | BTC   | 5000000   |
      | pegged2          | BTC   | 5000000   |
      | pegged3          | BTC   | 5000000   |
      | aux              | BTC   | 100000000 |
      | aux2             | BTC   | 100000000 |
# setup pegged orders
    Then the traders place the following pegged orders:
      | trader  | market id | side | volume | reference | offset |
      | pegged1 | ETH/DEC19 | sell | 10     | MID       | 10     |
      | pegged2 | ETH/DEC19 | buy  | 5      | MID       | -15    |
      | pegged3 | ETH/DEC19 | buy  | 5      | MID       | -10    |
    Then the pegged orders should have the following states:
      | trader  | market id | side | volume | reference | offset | price | status        |
      | pegged1 | ETH/DEC19 | sell | 10     | MID       | 10     | 0     | STATUS_PARKED |
      | pegged2 | ETH/DEC19 | buy  | 5      | MID       | -15    | 0     | STATUS_PARKED |
      | pegged3 | ETH/DEC19 | buy  | 5      | MID       | -10    | 0     | STATUS_PARKED |
# keep things simple: remove the events we've just verified
    And clear order events
    When the traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux              | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux2             | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |
    Then the orders should have the following states:
      | trader           | market id | side | volume | price | status        |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | STATUS_ACTIVE |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | STATUS_ACTIVE |
# Checked out, remove the order events we've checked, now let's have a look at the pegged order events
    And clear order events by reference:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
      | buySideProvider  | buy-provider-1  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
# Now check what happened to our pegged orders
    Then the pegged orders should have the following states:
      | trader  | market id | side | volume | reference | offset | price | status        |
      | pegged1 | ETH/DEC19 | sell | 10     | MID       | 10     | 110   | STATUS_ACTIVE |
      | pegged2 | ETH/DEC19 | buy  | 5      | MID       | -15    | 85    | STATUS_ACTIVE |
      | pegged3 | ETH/DEC19 | buy  | 5      | MID       | -10    | 90    | STATUS_ACTIVE |
