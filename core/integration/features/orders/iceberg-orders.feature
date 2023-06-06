Feature: Amend orders

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @iceberg
  Scenario: iceberg basic refresh
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | BTC   | 10000  |
      | party2 | BTC   | 10000  |
      | aux    | BTC   | 100000 |
      | aux2   | BTC   | 100000 |

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
  Scenario: iceberg refrehes leaving auction
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | BTC   | 10000  |
      | party2 | BTC   | 10000  |
      | aux    | BTC   | 100000 |
      | aux2   | BTC   | 100000 |

    # place an iceberg order that will trade when coming out of auction
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | initial peak | minimum peak |
      | party1 | ETH/DEC19 | buy  | 100    | 2     | 0                | TYPE_LIMIT | TIF_GTC | 10           | 10            |

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
  Scenario: Iceberg increase size success and not losing position in order book
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1  | BTC   | 10000  |
      | party2 | BTC   | 10000  |
      | party3 | BTC   | 10000  |
      | aux    | BTC   | 100000 |
      | aux2   | BTC   | 100000 |

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
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   | initial peak | minimum peak |
      | party1 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 2            | 1            |
      | party2 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-2 | 2            | 1            |

    # reducing size
    Then the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party1 | this-order-1 | 2     | 5          | TIF_GTC |

    # the visible is the same and only the reserve amount has increased
    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status         | reserved volume |
      | party1 | ETH/DEC19 | sell  | 2              | 2     | STATUS_ACTIVE  | 8               |

    # matching the order now
    # this should match with the size 3 order of party1
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party3 | ETH/DEC19 | buy  | 2      | 2     | 1                | TYPE_LIMIT | TIF_GTC | party3 |

    Then the following trades should be executed:
      | buyer  | seller  | price | size |
      | party3 | party1  | 2     | 2    |


  @icebergg
  Scenario: Iceberg amend price reenters aggressively
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | BTC   | 10000  |
      | party2 | BTC   | 10000  |
      | aux    | BTC   | 100000 |
      | aux2   | BTC   | 100000 |

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
      | party  | market id | side | volume  | price | resulting trades | type       | tif     | reference    | initial peak | minimum peak  |
      | party1 | ETH/DEC19 | sell | 16      | 5     | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 5            | 1             |
      | party2 | ETH/DEC19 | buy  | 10      | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-2 | 2            | 1             |

    # amend the buy order so that it will cross with the other iceberg
    Then the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party2 | this-order-2 | 5     | 0          | TIF_GTC |

    # the amended iceberg will trade aggressively and be fully consumed
    Then the iceberg orders should have the following states:
      | party  | market id | side  | visible volume | price | status          | reserved volume |
      | party1 | ETH/DEC19 | sell  | 5              | 5     | STATUS_ACTIVE   | 1               |
      | party2 | ETH/DEC19 | buy   | 0              | 5     | STATUS_FILLED   | 0               |
