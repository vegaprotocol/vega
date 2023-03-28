Feature: Trader amends his orders

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: 001 Amend rejected for non existing order
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

    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

    # cancel the order, so we cannot edit it.
    And the parties cancel the following orders:
      | party | reference   |
      | myboi | myboi-ref-1 |
    And the parties amend the following orders:
      | party | reference   | price | size delta | tif     | error                        |
      | myboi | myboi-ref-1 | 2     | 3          | TIF_GTC | OrderError: Invalid Order ID |

  Scenario: 002 Reduce size success and not loosing position in order book
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
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

    # reducing size
    When the parties amend the following orders:
      | party | reference   | price | size delta | tif     |
      | myboi | myboi-ref-1 | 0     | -2         | TIF_GTC |

    # matching the order now
    # this should match with the size 3 order of myboi
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi  | 2     | 3    |

  Scenario: 003 Increase size success and loosing position in order book
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
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |
    And the parties amend the following orders:
      | party | reference   | price | size delta | tif     |
      | myboi | myboi-ref-1 | 0     | 3          | TIF_GTC |

    # matching the order now
    # this should match with the size 3 order of myboi
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi2 | 2     | 3    |

  Scenario: 004 Reduce size success and order cancelled as  < to remaining
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
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

    # matching the order now
    # this will reduce the remaining to 2 so it get cancelled later on
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |
    And the parties amend the following orders:
      | party | reference   | price | size delta | tif     |
      | myboi | myboi-ref-1 | 0     | -3         | TIF_GTC |
    Then the orders should have the following status:
      | party | reference   | status           |
      | myboi | myboi-ref-1 | STATUS_CANCELLED |

  Scenario: 005 Amend to invalid tif is rejected
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

    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
    And the parties amend the following orders:
      | party | reference   | price | size delta | tif     | error                                      |
      | myboi | myboi-ref-1 | 0     | 0          | TIF_FOK | OrderError: Cannot amend TIF to FOK or IOC |

  Scenario: 006 TIF_GTC to TIF_GTT rejected without expiry
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount |
      | myboi | BTC   | 10000  |
      | aux   | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
    And the parties amend the following orders:
      | party | reference   | price | size delta | tif     | error                                                           |
      | myboi | myboi-ref-1 | 0     | 0          | TIF_GTT | OrderError: Cannot amend order to GTT without an expiryAt field |


  Scenario: 007 TIF_GTC to TIF_GTT with time in the past
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

    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
    And the parties amend the following orders:
      | party | reference   | price | size delta | expiration date      | tif     | error                                                   |
      | myboi | myboi-ref-1 | 2     | 0          | 2019-11-30T00:00:00Z | TIF_GTT | OrderError: ExpiryAt field must not be before CreatedAt |

@AmendBug
Scenario: 008 Amending expiry time of an active GTT order to a past time whilst also simultaneously amending the price of the order will cause the order to immediately expire with the order details updated to reflect the order details requiring amendment (0004-AMND-029)
    Given time is updated to "2019-11-30T00:00:00Z"
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount |
      | trader1 | BTC   | 10000  |
      | trader2 | BTC   | 10000  |
      | trader3 | BTC   | 10000  |
      | aux     | BTC   | 100000 |
      | aux2    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When time is updated to "2019-11-30T00:00:04Z"

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference   |
      | trader1 | ETH/DEC19 | sell | 3      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 3600       |GTT-ref-1 |
    # trader1 amend expiration date and price at the simultaneously
    And the parties amend the following orders:
      | party   | reference  | price | size delta | expiration date      | tif     | 
      | trader1 | GTT-ref-1  | 1002  | 0          | 2019-11-30T00:00:05Z | TIF_GTT | 
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1000  | 0      |
      | sell | 1002  | 3      |
      | sell | 10001 | 1      |

    When time is updated to "2020-01-30T00:00:00Z"
 
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1000  | 0      |
      | sell | 1002  | 0      |
      | sell | 10001 | 1      |

    When time is updated to "2020-01-30T10:00:00Z"

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference   |
      | trader2 | ETH/DEC19 | sell | 5      | 1005  | 0                | TYPE_LIMIT | TIF_GTT | 3600       |GTT-ref-2 |
    # trader2 amend expiration date only
    And the parties amend the following orders:
      | party   | reference  | price | size delta | expiration date      | tif     | 
      | trader2 | GTT-ref-2  | 1005  | 0          | 2020-01-30T10:00:01Z | TIF_GTT | 
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1000  | 0      |
      | sell | 1005  | 5      |
      | sell | 10001 | 1      |
    When time is updated to "2020-01-30T12:00:01Z"
 
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1000  | 0      |
      | sell | 1005  | 0      |
      | sell | 10001 | 1      |

    When time is updated to "2020-01-30T12:01:01Z"

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference   |
      | trader3 | ETH/DEC19 | sell | 6      | 1006  | 0                | TYPE_LIMIT | TIF_GTT | 3600       |GTT-ref-3 |

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1006  | 6      |
      | sell | 10001 | 1      |

    When time is updated to "2020-02-01T12:00:01Z"

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1006  | 0      |
      | sell | 10001 | 1      |

   
