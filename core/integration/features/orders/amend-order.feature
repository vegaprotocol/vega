Feature: Amend orders

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Amend rejected for non existing order, amend the order to cancel 0004-AMND-058
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount |
      | myboi | BTC   | 10000  |
      | aux   | BTC   | 100000 |
      | aux1  | BTC   | 100000 |
      | aux2  | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |reference |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |      |
      | aux   | ETH/DEC19 | sell | 3      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |aux_s |
      | aux1  | ETH/DEC19 | sell | 3      | 10002 | 0                | TYPE_LIMIT | TIF_GTC |aux_s1|
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |      |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |      |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

    #0004-AMND-058: Size change amends which would result in the remaining part of the order being reduced below zero should instead cancel the order
    Then the parties amend the following orders:
      | party | reference | price | size delta | tif    |  
      | aux   | aux_s     | 10001 | -4        | TIF_GTC | 

    And the orders should have the following status:
      | party  | reference | status           |
      | aux    |aux_s      | STATUS_CANCELLED |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 10001 | 0      |

    #0004-AMND-059:A transaction specifying both a `sizeDelta` and `size` field should be rejected as invalid
    Then the parties amend the following orders:
      | party | reference | price | size delta | tif     | size | 
      | aux1  | aux_s1    | 10002 | -1         | TIF_GTC | 1    |

    # cancel the order, so we cannot edit it.
    And the parties cancel the following orders:
      | party | reference   |
      | myboi | myboi-ref-1 |

    Then the parties amend the following orders:
      | party | reference   | price | size delta | tif     | error                        |
      | myboi | myboi-ref-1 | 2     | 3          | TIF_GTC | OrderError: Invalid Order ID |

  Scenario: Reduce size with delta success and not loosing position in order book (0004-AMND-003, 0004-AMND-057)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | myboi  | BTC   | 10000  |
      | myboi2 | BTC   | 10000  |
      | myboi3 | BTC   | 10000  |
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

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | myboi  | ETH/DEC19 | sell | 10     | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1  |
      | myboi2 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi2-ref-1 |

    # reducing size with delta
    Then the parties amend the following orders:
      | party | reference   | price | size delta | tif     |
      | myboi | myboi-ref-1 | 0     | -2         | TIF_GTC |

    And the orders should have the following states:
      | party | market id | reference   | side | volume | remaining | price | status        |
      | myboi | ETH/DEC19 | myboi-ref-1 | sell | 8      | 8         | 2     | STATUS_ACTIVE |

    # matching the order now
    # this should match with the size 3 order of myboi
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | myboi3 | ETH/DEC19 | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi3-ref-1 |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi  | 2     | 3    |

    And the orders should have the following states:
      | party | market id | reference   | side | volume | remaining | price | status        |
      | myboi | ETH/DEC19 | myboi-ref-1 | sell | 8      | 5         | 2     | STATUS_ACTIVE |

    # reducing size with target
    When the parties amend the following orders:
      | party | reference   | price | size | tif     |
      | myboi | myboi-ref-1 | 0     | 6    | TIF_GTC |

    Then the orders should have the following states:
      | party | market id | reference   | side | volume | remaining | price | status        |
      | myboi | ETH/DEC19 | myboi-ref-1 | sell | 6      | 3         | 2     | STATUS_ACTIVE |

  Scenario: Increase size success and loosing position in order book (0004-AMND-005, 0004-AMND-056)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | myboi  | BTC   | 10000     |
      | myboi2 | BTC   | 10000     |
      | myboi3 | BTC   | 100000000 |
      | aux    | BTC   | 100000    |
      | aux2   | BTC   | 100000    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

    And the parties amend the following orders:
      | party | reference   | price | size delta | tif     |
      | myboi | myboi-ref-1 | 0     | 3          | TIF_GTC |

    Then the orders should have the following states:
      | party | market id | reference   | side | volume | remaining | price | status        |
      | myboi | ETH/DEC19 | myboi-ref-1 | sell | 8      | 8         | 2     | STATUS_ACTIVE |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi2 | 2     | 3    |

    And the orders should have the following states:
      | party  | market id | reference   | side | volume | remaining | price | status        |
      | myboi  | ETH/DEC19 | myboi-ref-1 | sell | 8      | 8         | 2     | STATUS_ACTIVE |
      | myboi2 | ETH/DEC19 | myboi-ref-2 | sell | 5      | 2         | 2     | STATUS_ACTIVE |

    When the parties amend the following orders:
      | party  | reference   | price | size | tif     |
      | myboi2 | myboi-ref-2 | 0     | 10   | TIF_GTC |

    Then the orders should have the following states:
      | party  | market id | reference   | side | volume | remaining | price | status        |
      | myboi  | ETH/DEC19 | myboi-ref-1 | sell | 8      | 8         | 2     | STATUS_ACTIVE |
      | myboi2 | ETH/DEC19 | myboi-ref-2 | sell | 10     | 7         | 2     | STATUS_ACTIVE |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi  | 2     | 3    |
    And the orders should have the following states:
      | party  | market id | reference   | side | volume | remaining | price | status        |
      | myboi  | ETH/DEC19 | myboi-ref-1 | sell | 8      | 5         | 2     | STATUS_ACTIVE |
      | myboi2 | ETH/DEC19 | myboi-ref-2 | sell | 10     | 7         | 2     | STATUS_ACTIVE |


  Scenario: Reduce size success and order cancelled as remaining is less than or equal to 0 (0004-AMND-058)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount |
      | myboi | BTC   | 10000  |
      | aux   | BTC   | 100000 |
      | aux2  | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

    And the parties amend the following orders:
      | party | reference   | price | size delta | tif     |
      | myboi | myboi-ref-1 | 0     | -5         | TIF_GTC |

    Then the orders should have the following status:
      | party | reference   | status           |
      | myboi | myboi-ref-1 | STATUS_CANCELLED |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

    And the parties amend the following orders:
      | party | reference   | price | size | tif     |
      | myboi | myboi-ref-1 | 0     | 0    | TIF_GTC |

    Then the orders should have the following status:
      | party | reference   | status           |
      | myboi | myboi-ref-1 | STATUS_CANCELLED |

  Scenario: Amend to invalid tif is rejected
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount |
      | myboi | BTC   | 10000  |
      | aux   | BTC   | 100000 |
      | aux2  | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |


    # cannot amend TIF to TIF_FOK so this will be rejected
    Then the parties amend the following orders:
      | party | reference   | price | size delta | tif     | error                                      |
      | myboi | myboi-ref-1 | 0     | 0          | TIF_FOK | OrderError: Cannot amend TIF to FOK or IOC |

  Scenario: TIF_GTC to TIF_GTT rejected without expiry
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount |
      | myboi | BTC   | 10000  |
      | aux   | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

    # TIF_GTT rejected because of missing expiration date
    Then the parties amend the following orders:
      | party | reference   | price | size delta | tif     | error                                                           |
      | myboi | myboi-ref-1 | 0     | 0          | TIF_GTT | OrderError: Cannot amend order to GTT without an expiryAt field |

  Scenario: TIF_GTC to TIF_GTT with time in the past
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount |
      | myboi | BTC   | 10000  |
      | aux   | BTC   | 100000 |
      | aux2  | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

    # reducing size, remaining goes from 2 to -1, this will cancel
    Then the parties amend the following orders:
      | party | reference   | price | expiration date      | size delta | tif     |
      | myboi | myboi-ref-1 | 2     | 2030-11-30T00:00:00Z | 0          | TIF_GTT |
