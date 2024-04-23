Feature: Spot trader amends his orders

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
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @AmendBug
  Scenario: 008 Amending expiry time of an active GTT order to a past time whilst also simultaneously amending the price of the order will cause the order to immediately expire with the order details updated to reflect the order details requiring amendment (0004-AMND-048)
    Given time is updated to "2019-11-30T00:00:00Z"
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount |
      | trader1 | BTC   | 10000  |
      | trader2 | BTC   | 10000  |
      | trader3 | BTC   | 10000  |
      | aux     | BTC   | 100000 |
      | aux2    | BTC   | 100000 |
      | aux     | ETH   | 100000 |
      | aux2    | ETH   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When time is updated to "2019-11-30T00:00:04Z"

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference |
      | trader1 | BTC/ETH   | sell | 3      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 3600       | GTT-ref-1 |
    # trader1 amend expiration date and price at the simultaneously
    And the parties amend the following orders:
      | party   | reference | price | size delta | expiration date      | tif     |
      | trader1 | GTT-ref-1 | 1002  | 0          | 2019-11-30T00:00:05Z | TIF_GTT |
    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 1000  | 0      |
      | sell | 1002  | 3      |
      | sell | 10001 | 1      |

    When time is updated to "2020-01-30T00:00:00Z"

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 1000  | 0      |
      | sell | 1002  | 0      |
      | sell | 10001 | 1      |

    When time is updated to "2020-01-30T10:00:00Z"

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference |
      | trader2 | BTC/ETH   | sell | 5      | 1005  | 0                | TYPE_LIMIT | TIF_GTT | 3600       | GTT-ref-2 |
    # trader2 amend expiration date only
    And the parties amend the following orders:
      | party   | reference | price | size delta | expiration date      | tif     |
      | trader2 | GTT-ref-2 | 1005  | 0          | 2020-01-30T10:00:01Z | TIF_GTT |
    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 1000  | 0      |
      | sell | 1005  | 5      |
      | sell | 10001 | 1      |
    When time is updated to "2020-01-30T12:00:01Z"

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 1000  | 0      |
      | sell | 1005  | 0      |
      | sell | 10001 | 1      |

    When time is updated to "2020-01-30T12:01:01Z"

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference |
      | trader3 | BTC/ETH   | sell | 6      | 1006  | 0                | TYPE_LIMIT | TIF_GTT | 3600       | GTT-ref-3 |

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 1006  | 6      |
      | sell | 10001 | 1      |

    When time is updated to "2020-02-01T12:00:01Z"

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 1006  | 0      |
      | sell | 10001 | 1      |

  Scenario: In Spot market amending an order in a way that increases the volume sufficiently leads to holding account balance increasing (0004-AMND-049). In Spot market amending an order in a way that decreases the volume sufficiently leads to holding account balance decreasing (0004-AMND-050). In Spot market, if an order is amended such that holding requirement is increased and user has sufficient balance in the general account to top up their holding account then the amendment is executed successfully (0011-MARA-018).
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount |
      | trader1 | BTC   | 10000  |
      | aux     | BTC   | 100000 |
      | aux2    | BTC   | 100000 |
      | aux     | ETH   | 100000 |
      | aux2    | ETH   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/ETH   | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/ETH   | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When time is updated to "2019-11-30T00:00:04Z"

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | GTC-ref-1 |

    Then "trader1" should have general account balance of "9999" for asset "BTC"
    Then "trader1" should have holding account balance of "1" for asset "BTC"

    # trader1 amend increase volumne should increase holding account balance
    And the parties amend the following orders:
      | party   | reference | price | size delta | tif     |
      | trader1 | GTC-ref-1 | 1000  | 4          | TIF_GTC |

    Then "trader1" should have general account balance of "9995" for asset "BTC"
    Then "trader1" should have holding account balance of "5" for asset "BTC"

    # trader1 amend increase volumne should decrease holding account balance
    And the parties amend the following orders:
      | party   | reference | price | size delta | tif     |
      | trader1 | GTC-ref-1 | 1000  | -2         | TIF_GTC |

    Then "trader1" should have general account balance of "9997" for asset "BTC"
    Then "trader1" should have holding account balance of "3" for asset "BTC"

    # trader1 amend increase volumne has insufficient funds, amend should fail and order should remain unchanged 0011-MARA-019
    And the parties amend the following orders:
      | party   | reference | price | size delta | tif     | error                                                        |
      | trader1 | GTC-ref-1 | 1000  | 9998       | TIF_GTC | party does not have sufficient balance to cover the new size |

    Then "trader1" should have general account balance of "9997" for asset "BTC"
    Then "trader1" should have holding account balance of "3" for asset "BTC"

    And the orders should have the following status:
      | party   | reference | status        |
      | trader1 | GTC-ref-1 | STATUS_ACTIVE |