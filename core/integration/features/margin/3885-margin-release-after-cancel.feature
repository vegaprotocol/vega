Feature: Regression test for issue 3885

  Background:

    Given the markets:
      | id        | quote name | asset | auction duration | risk model                    | margin calculator         | fees         | data source config     | price monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | 1                | default-log-normal-risk-model | default-margin-calculator | default-none | default-eth-for-future | default-none     | 1e0                    | 0                         |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @Cancel
  Scenario: Margin should be released after the order was canceled
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | BTC   | 10000  |
      | party2 | BTC   | 10000  |
      | party3 | BTC   | 10000  |
      | party4 | BTC   | 10000  |
      | lpprov | BTC   | 10000  |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 9000              | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 9000              | 0.1 | sell | ASK              | 50         | 100    | submission |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/DEC19 | buy  | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC19 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party2-1  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 14     | 9986    |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | party2-2  |
    Then the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 256    | 9744    |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | party1-3  |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 265    | 9735    |
    And the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-3  |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 265    | 9735    |
    # With a small change to force margin recalculating whennever an order is removed, we can have margins released.
    # But back when we implemented this, we decided not to check margins for parties who still have an open position.
    # The reasoning being that any party with an open position will get their margin released/topped up next MTM cycle.
    # Cancelling an order, even if it changes the potential long/short, will always decrease margin requirements. All this would do is
    # increase the number of transfers between margin and general accounts.
    # | party1 | BTC   | ETH/DEC19 | 24     | 9976    |
    And the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

