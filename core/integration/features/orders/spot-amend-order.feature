Feature: Amend spot orders

  Background:
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 1                | fees-config-1 | default-none     | default-basic |
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 1000   |
      | party2 | BTC   | 100    |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Reduce size with delta success and not losing position in order book (0004-AMND-0032)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | myboi  | BTC   | 10000  |
      | myboi  | ETH   | 10000  |
      | myboi2 | BTC   | 10000  |
      | myboi2 | ETH   | 10000  |
      | myboi3 | BTC   | 10000  |
      | myboi3 | ETH   | 10000  |
      | aux    | BTC   | 100000 |
      | aux    | ETH   | 100000 |
      | aux2   | BTC   | 100000 |
      | aux2   | ETH   | 100000 |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | myboi  | BTC/ETH   | sell | 10     | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1  |
      | myboi2 | BTC/ETH   | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi2-ref-1 |

    # reducing size with delta
    Then the parties amend the following orders:
      | party | reference   | price | size delta | tif     |
      | myboi | myboi-ref-1 | 0     | -2         | TIF_GTC |

    And the orders should have the following states:
      | party | market id | reference   | side | volume | remaining | price | status        |
      | myboi | BTC/ETH   | myboi-ref-1 | sell | 8      | 8         | 2     | STATUS_ACTIVE |

    # matching the order now
    # this should match with the size 3 order of myboi
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | myboi3 | BTC/ETH   | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi3-ref-1 |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi  | 2     | 3    |

    And the orders should have the following states:
      | party | market id | reference   | side | volume | remaining | price | status        |
      | myboi | BTC/ETH   | myboi-ref-1 | sell | 8      | 5         | 2     | STATUS_ACTIVE |

    # reducing size with target
    When the parties amend the following orders:
      | party | reference   | price | size | tif     |
      | myboi | myboi-ref-1 | 0     | 6    | TIF_GTC |

    Then the orders should have the following states:
      | party | market id | reference   | side | volume | remaining | price | status        |
      | myboi | BTC/ETH   | myboi-ref-1 | sell | 6      | 3         | 2     | STATUS_ACTIVE |

  Scenario: Increase size success and loosing position in order book (0004-AMND-033)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | myboi  | BTC   | 10000     |
      | myboi2 | BTC   | 10000     |
      | myboi3 | BTC   | 100000000 |
      | aux    | BTC   | 100000    |
      | aux2   | BTC   | 100000    |
      | myboi  | ETH   | 10000     |
      | myboi2 | ETH   | 10000     |
      | myboi3 | ETH   | 100000000 |
      | aux    | ETH   | 100000    |
      | aux2   | ETH   | 100000    |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | BTC/ETH   | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | BTC/ETH   | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

    And the parties amend the following orders:
      | party | reference   | price | size delta | tif     |
      | myboi | myboi-ref-1 | 0     | 3          | TIF_GTC |

    Then the orders should have the following states:
      | party | market id | reference   | side | volume | remaining | price | status        |
      | myboi | BTC/ETH   | myboi-ref-1 | sell | 8      | 8         | 2     | STATUS_ACTIVE |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | BTC/ETH   | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi2 | 2     | 3    |

    And the orders should have the following states:
      | party  | market id | reference   | side | volume | remaining | price | status        |
      | myboi  | BTC/ETH   | myboi-ref-1 | sell | 8      | 8         | 2     | STATUS_ACTIVE |
      | myboi2 | BTC/ETH   | myboi-ref-2 | sell | 5      | 2         | 2     | STATUS_ACTIVE |

    When the parties amend the following orders:
      | party  | reference   | price | size | tif     |
      | myboi2 | myboi-ref-2 | 0     | 10   | TIF_GTC |

    Then the orders should have the following states:
      | party  | market id | reference   | side | volume | remaining | price | status        |
      | myboi  | BTC/ETH   | myboi-ref-1 | sell | 8      | 8         | 2     | STATUS_ACTIVE |
      | myboi2 | BTC/ETH   | myboi-ref-2 | sell | 10     | 7         | 2     | STATUS_ACTIVE |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | BTC/ETH   | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi  | 2     | 3    |
    And the orders should have the following states:
      | party  | market id | reference   | side | volume | remaining | price | status        |
      | myboi  | BTC/ETH   | myboi-ref-1 | sell | 8      | 5         | 2     | STATUS_ACTIVE |
      | myboi2 | BTC/ETH   | myboi-ref-2 | sell | 10     | 7         | 2     | STATUS_ACTIVE |

  Scenario: Reduce size success and order cancelled as remaining is less than or equal to 0 (0004-AMND-058)
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount |
      | myboi | BTC   | 10000  |
      | aux   | BTC   | 100000 |
      | aux2  | BTC   | 100000 |
      | myboi | ETH   | 10000  |
      | aux   | ETH   | 100000 |
      | aux2  | ETH   | 100000 |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | BTC/ETH   | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

    And the parties amend the following orders:
      | party | reference   | price | size delta | tif     |
      | myboi | myboi-ref-1 | 0     | -5         | TIF_GTC |

    Then the orders should have the following status:
      | party | reference   | status           |
      | myboi | myboi-ref-1 | STATUS_CANCELLED |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | BTC/ETH   | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

    And the parties amend the following orders:
      | party | reference   | price | size | tif     |
      | myboi | myboi-ref-1 | 0     | 0    | TIF_GTC |

    Then the orders should have the following status:
      | party | reference   | status           |
      | myboi | myboi-ref-1 | STATUS_CANCELLED |

  Scenario: Amend to invalid tif is rejected (0004-AMND-034)
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount |
      | myboi | BTC   | 10000  |
      | aux   | BTC   | 100000 |
      | aux2  | BTC   | 100000 |
      | myboi | ETH   | 10000  |
      | aux   | ETH   | 100000 |
      | aux2  | ETH   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | BTC/ETH   | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |


    # cannot amend TIF to TIF_FOK so this will be rejected
    Then the parties amend the following orders:
      | party | reference   | price | size delta | tif     | error                                      |
      | myboi | myboi-ref-1 | 0     | 0          | TIF_FOK | OrderError: Cannot amend TIF to FOK or IOC |

  Scenario: TIF_GTC to TIF_GTT rejected without expiry (0004-AMND-034)
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount |
      | myboi | BTC   | 10000  |
      | aux   | BTC   | 100000 |
      | myboi | ETH   | 10000  |
      | aux   | ETH   | 100000 |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | BTC/ETH   | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

    # TIF_GTT rejected because of missing expiration date
    Then the parties amend the following orders:
      | party | reference   | price | size delta | tif     | error                                                           |
      | myboi | myboi-ref-1 | 0     | 0          | TIF_GTT | OrderError: Cannot amend order to GTT without an expiryAt field |

  Scenario: TIF_GTC to TIF_GTT with time in the past (0004-AMND-034)
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount |
      | myboi | BTC   | 10000  |
      | aux   | BTC   | 100000 |
      | aux2  | BTC   | 100000 |
      | myboi | ETH   | 10000  |
      | aux   | ETH   | 100000 |
      | aux2  | ETH   | 100000 |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | BTC/ETH   | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

    # reducing size, remaining goes from 2 to -1, this will cancel
    Then the parties amend the following orders:
      | party | reference   | price | expiration date      | size delta | tif     |
      | myboi | myboi-ref-1 | 2     | 2030-11-30T00:00:00Z | 0          | TIF_GTT |

  Scenario: Any attempt to amend to or from the TIF values GFA and GFN will result in a rejected amend (0004-AMND-035)
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount |
      | myboi | BTC   | 10000  |
      | aux   | BTC   | 100000 |
      | aux2  | BTC   | 100000 |
      | myboi | ETH   | 10000  |
      | aux   | ETH   | 100000 |
      | aux2  | ETH   | 100000 |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | BTC/ETH   | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

    Then the parties amend the following orders:
      | party | reference   | price | tif     | error                                      |
      | myboi | myboi-ref-1 | 2     | TIF_GFA | OrderError: Cannot amend TIF to GFA or GFN |
      | myboi | myboi-ref-1 | 2     | TIF_GFN | OrderError: Cannot amend TIF to GFA or GFN |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | BTC/ETH   | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GFN | myboi-ref-2 |

    Then the parties amend the following orders:
      | party | reference   | price | tif     | error                                      |
      | myboi | myboi-ref-2 | 2     | TIF_GTC | OrderError: Cannot amend TIF from GFA or GFN |

