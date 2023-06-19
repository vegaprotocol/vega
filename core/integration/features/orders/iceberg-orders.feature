Feature: Iceberg orders

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 1500  |

  @iceberg
  Scenario: Iceberg order submission with valid TIF's (0014-ORDT-007)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak | only |
      | party1 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | 10           | 5            | post |

    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 10             | 10    | STATUS_ACTIVE | 90              |

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | initial peak | minimum peak | only |
      | party2 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTT | 3600       | 8            | 4            | post |

    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party2 | ETH/DEC19 | buy  | 8              | 10    | STATUS_ACTIVE | 92              |


  @iceberg
  Scenario: An iceberg order with either an ordinary or pegged limit price can be submitted (0014-ORDT-008)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | best-bid  |
      | party2 | ETH/DEC19 | sell | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC | best-ask  |
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak | reference        |
      | party1 | ETH/DEC19 | buy  | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC | 3           | 1            | ordinary-iceberg |
    And the parties place the following pegged iceberg orders:
      | party  | market id | side | volume | resulting trades | type       | tif     | initial peak | minimum peak | pegged reference | offset | reference      |
      | party1 | ETH/DEC19 | buy  | 10     | 0                | TYPE_LIMIT | TIF_GTC | 2            | 1            | BID              | 1      | pegged-iceberg |
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 3      |
      | buy  | 9     | 2      |
      | buy  | 10    | 1      |

    # Move best-bid and check pegged iceberg order is re-priced
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     |
      | party2 | best-bid  | 9     | 0          | TIF_GTC |
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 3      |
      | buy  | 8     | 2      |
      | buy  | 9     | 1      |


  @iceberg
  Scenario: Iceberg order margin calculation (0014-ORDT-011)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak | only | reference       |
      | party1 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | 10           | 5            | post | iceberg-order-1 |

    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 10             | 10    | STATUS_ACTIVE | 90              |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 26     | 9974    |

    # And another party places a normal limit order for the same price and quantity, then the same margin should be taken
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference            |
      | party2 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | normal-limit-order-1 |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party2 | BTC   | ETH/DEC19 | 26     | 9974    |

    # Now we cancel the iceberg order
    Then the parties cancel the following orders:
      | party  | reference       |
      | party1 | iceberg-order-1 |

    # And the margin taken for the iceberg order is released
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 0      | 10000   |

    # Now we cancel the normal limit order
    Then the parties cancel the following orders:
      | party  | reference            |
      | party2 | normal-limit-order-1 |

    # And the margin taken for the normal limit order is released
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party2 | BTC   | ETH/DEC19 | 0      | 10000   |


  @iceberg
  Scenario: iceberg basic refresh (0014-ORDT-012)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak |
      | party1 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | 10           | 5            |

    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 10             | 10    | STATUS_ACTIVE | 90              |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 6      | 10    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 10             | 10    | STATUS_ACTIVE | 84              |

  @iceberg
  Scenario: iceberg refreshes leaving auction
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place an iceberg order that will trade when coming out of auction
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak |
      | party1 | ETH/DEC19 | buy  | 100    | 2     | 0                | TYPE_LIMIT | TIF_GTC | 10           | 10           |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 10             | 2     | STATUS_ACTIVE | 89              |


  @iceberg
  @margin
  Scenario: Iceberg increase size success and not losing position in order book (0014-ORDT-023)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | initial peak | minimum peak |
      | party1 | ETH/DEC19 | sell | 50     | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 2            | 1            |
      | party2 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-2 | 2            | 1            |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 12     | 9988    |

    # increasing size
    Then the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party1 | this-order-1 | 2     | 50         | TIF_GTC |

    # the visible is the same and only the reserve amount has increased
    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | sell | 2              | 2     | STATUS_ACTIVE | 98              |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 24     | 9976    |

    # matching the order now
    # this should match with the size 2 order of party1
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 2      | 2     | 1                | TYPE_LIMIT | TIF_GTC | party3    |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party3 | party1 | 2     | 2    |

  @iceberg
  Scenario: Iceberg decrease size success and not losing position in order book (0014-ORDT-024) (0014-ORDT-025)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | initial peak | minimum peak |
      | party1 | ETH/DEC19 | sell | 100    | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 2            | 1            |
      | party2 | ETH/DEC19 | sell | 100    | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-2 | 2            | 1            |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 24     | 9976    |

    # decreasing size
    Then the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party1 | this-order-1 | 2     | -50        | TIF_GTC |

    # the visible is the same and only the reserve amount has decreased
    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | sell | 2              | 2     | STATUS_ACTIVE | 48              |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 12     | 9988    |

    # matching the order now
    # this should match with the size 2 order of party1
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 2      | 2     | 1                | TYPE_LIMIT | TIF_GTC | party3    |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party3 | party1 | 2     | 2    |

  @iceberg
  Scenario: Iceberg amend price reenters aggressively
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | initial peak | minimum peak |
      | party1 | ETH/DEC19 | sell | 16     | 5     | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 5            | 1            |
      | party2 | ETH/DEC19 | buy  | 10     | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-2 | 2            | 1            |

    # amend the buy order so that it will cross with the other iceberg
    Then the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party2 | this-order-2 | 5     | 0          | TIF_GTC |

    # the amended iceberg will trade aggressively and be fully consumed
    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | sell | 5              | 5     | STATUS_ACTIVE | 1               |
      | party2 | ETH/DEC19 | buy  | 0              | 5     | STATUS_FILLED | 0               |


  @iceberg
  Scenario: An aggressive iceberg order crosses an order with volume > iceberg volume (0014-ORDT-027)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 15     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak |
      | party1 | ETH/DEC19 | buy  | 10     | 5     | 1                | TYPE_LIMIT | TIF_GTC | 2            | 1            |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party2 | 5     | 10   |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 0              | 5     | STATUS_FILLED | 0               |


  @iceberg
  Scenario: An aggressive iceberg order crosses an order with volume < iceberg volume (0014-ORDT-028)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak |
      | party1 | ETH/DEC19 | buy  | 15     | 5     | 1                | TYPE_LIMIT | TIF_GTC | 2            | 1            |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party2 | 5     | 10   |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 2              | 5     | STATUS_ACTIVE | 3               |


  @iceberg
  Scenario: A passive iceberg order (the only order at the price level) crosses an order with volume > iceberg volume (0014-ORDT-029)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak |
      | party1 | ETH/DEC19 | buy  | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2            | 1            |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 15     | 5     | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party2 | 5     | 10   |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 0              | 5     | STATUS_FILLED | 0               |


  @iceberg
  Scenario: A passive iceberg order (one of multiple orders at the price level) crosses an order with volume > iceberg volume (0014-ORDT-030)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak |
      | party1 | ETH/DEC19 | buy  | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2            | 1            |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | buy  | 7      | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | buy  | 7      | 5     | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/DEC19 | sell | 15     | 5     | 3                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party4 | 5     | 2    |
      | party2 | party4 | 5     | 7    |
      | party3 | party4 | 5     | 6    |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 2              | 5     | STATUS_ACTIVE | 6               |


  @iceberg
  Scenario: An aggressive iceberg order crosses orders where the cumulative volume > iceberg volume (0014-ORDT-031)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 30     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | sell | 40     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC19 | sell | 50     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak |
      | party1 | ETH/DEC19 | buy  | 100    | 5     | 3                | TYPE_LIMIT | TIF_GTC | 2            | 1            |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party2 | 5     | 30   |
      | party1 | party3 | 5     | 40   |
      | party1 | party4 | 5     | 30   |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 0              | 5     | STATUS_FILLED | 0               |


  @iceberg
  Scenario: An aggressive iceberg order crosses orders where the cumulative volume < iceberg volume (0014-ORDT-032)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 30     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | sell | 40     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak |
      | party1 | ETH/DEC19 | buy  | 100    | 5     | 2                | TYPE_LIMIT | TIF_GTC | 2            | 1            |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party2 | 5     | 30   |
      | party1 | party3 | 5     | 40   |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 2              | 5     | STATUS_ACTIVE | 28              |


  @iceberg
  Scenario: Amended order trades with iceberg order triggering a refresh (0014-ORDT-033)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak |
      | party1 | ETH/DEC19 | sell | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2            | 1            |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | party2 | ETH/DEC19 | buy  | 7      | 4     | 0                | TYPE_LIMIT | TIF_GTC | order-to-amend |
      | party3 | ETH/DEC19 | sell | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC |                |
    When the parties amend the following orders:
      | party  | reference      | price | size delta | tif     |
      | party2 | order-to-amend | 5     | 0          | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller | price | size | 
      | party2 | party1 | 5     | 2    |
      | party2 | party3 | 5     | 5    |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | sell | 2              | 5     | STATUS_ACTIVE | 6               |


  @iceberg
  Scenario: Attempting to wash trade with iceberg orders (0014-ORDT-034)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | sell | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak | reference |
      | party1 | ETH/DEC19 | sell | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2            | 1            | iceberg   |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 5     | 2                | TYPE_LIMIT | TIF_GTC | normal    |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party2 | 5     | 5    |
      | party1 | party3 | 5     | 5    |
    And the orders should have the following states:
      | party  | market id | reference | side | volume | price | status                  |
      | party1 | ETH/DEC19 | normal    | buy  | 20     | 5     | STATUS_PARTIALLY_FILLED |
    And the iceberg orders should have the following states:
      | party  | market id | reference | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | iceberg   | sell | 2              | 5     | STATUS_ACTIVE | 3               |

  @iceberg
  Scenario: An order matches multiple icebergs at the same level where the order volume < cumulative iceberg display volume (0014-ORDT-037)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak | reference |
      | party1 | ETH/DEC19 | sell | 200    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2            | 1            | iceberg   |
      | party2 | ETH/DEC19 | sell | 100    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2            | 1            | iceberg   |
      | party3 | ETH/DEC19 | sell | 100    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2            | 1            | iceberg   |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/DEC19 | buy  | 300    | 5     | 3                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party4 | party1 | 5     | 150  |
      | party4 | party2 | 5     | 75   |
      | party4 | party3 | 5     | 75   |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | sell | 2              | 5     | STATUS_ACTIVE | 48              |
      | party2 | ETH/DEC19 | sell | 2              | 5     | STATUS_ACTIVE | 23              |
      | party3 | ETH/DEC19 | sell | 2              | 5     | STATUS_ACTIVE | 23              |


@iceberg
  Scenario: An order matches multiple icebergs at the same level where the order volume > cumulative iceberg display volume (0014-ORDT-038)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak | reference |
      | party1 | ETH/DEC19 | sell | 200    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2            | 1            | iceberg   |
      | party2 | ETH/DEC19 | sell | 100    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2            | 1            | iceberg   |
      | party3 | ETH/DEC19 | sell | 100    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2            | 1            | iceberg   |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/DEC19 | buy  | 600    | 5     | 3                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party4 | party1 | 5     | 200  |
      | party4 | party2 | 5     | 100  |
      | party4 | party3 | 5     | 100  |

