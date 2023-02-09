Feature: Test market depth events for pegged orders (with BID and ASK price)

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | limits.markets.maxPeggedOrders          | 1500  |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Ensure the expect order events for pegged orders are produced for all references
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | sellSideProvider | BTC   | 100000000 |
      | buySideProvider  | BTC   | 100000000 |
      | pegged1          | BTC   | 5000000   |
      | pegged2          | BTC   | 5000000   |
      | pegged3          | BTC   | 5000000   |
      | pegged4          | BTC   | 5000000   |
      | aux              | BTC   | 100000000 |
      | aux2             | BTC   | 100000000 |
    # setup pegged orders
    Then the parties place the following pegged orders:
      | party   | market id | side | volume | pegged reference | offset |
      | pegged1 | ETH/DEC19 | sell | 5      | ASK              | 10     |
      | pegged2 | ETH/DEC19 | sell | 5      | MID              | 15     |
      | pegged3 | ETH/DEC19 | buy  | 5      | BID              | 10     |
      | pegged4 | ETH/DEC19 | buy  | 5      | MID              | 10     |
    Then the pegged orders should have the following states:
      | party   | market id | side | volume | reference | offset | price | status        |
      | pegged1 | ETH/DEC19 | sell | 5      | ASK       | 10     | 0     | STATUS_PARKED |
      | pegged2 | ETH/DEC19 | sell | 5      | MID       | 15     | 0     | STATUS_PARKED |
      | pegged3 | ETH/DEC19 | buy  | 5      | BID       | 10     | 0     | STATUS_PARKED |
      | pegged4 | ETH/DEC19 | buy  | 5      | MID       | 10     | 0     | STATUS_PARKED |
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux              | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux2             | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |
    Then the orders should have the following states:
      | party            | market id | side | volume | price | status        |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | STATUS_ACTIVE |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | STATUS_ACTIVE |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    # Now check what happened to our pegged orders
    Then the pegged orders should have the following states:
      | party   | market id | side | volume | reference | offset | price | status        |
      | pegged1 | ETH/DEC19 | sell | 5      | ASK       | 10     | 130   | STATUS_ACTIVE |
      | pegged1 | ETH/DEC19 | sell | 5      | ASK       | 10     | 130   | STATUS_ACTIVE |
      | pegged1 | ETH/DEC19 | sell | 5      | ASK       | 10     | 130   | STATUS_ACTIVE |
      | pegged2 | ETH/DEC19 | sell | 5      | MID       | 15     | 115   | STATUS_ACTIVE |
      | pegged3 | ETH/DEC19 | buy  | 5      | BID       | 10     | 70    | STATUS_ACTIVE |
      | pegged4 | ETH/DEC19 | buy  | 5      | MID       | 10     | 90    | STATUS_ACTIVE |
