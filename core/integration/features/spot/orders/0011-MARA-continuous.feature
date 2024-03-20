Feature: Spot market

  Background:

    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.value.windowLength               | 1h    |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.01      | 0.03               |

    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 360000  | 0.999       | 1                 |
    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | ETH   | 10000  |
      | party2 | BTC   | 10     |
      | party3 | ETH   | 10000  |
      | party5 | BTC   | 100    |
    And the average block duration is "1"
    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"


  Scenario: In Spot market, holding in holding account is correctly calculated for all order types in continuous trading limit GTT. (0011-MARA-022)
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference |
      | party5 | BTC/ETH   | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 50         | sell1     |

    And "party5" should have holding account balance of "5" for asset "BTC"

  Scenario: In Spot market, holding in holding account is correctly calculated for all order types in continuous trading limit GTC. (0011-MARA-023)
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | BTC/ETH   | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    And "party5" should have holding account balance of "5" for asset "BTC"

  Scenario: In Spot market, holding in holding account is correctly calculated for all order types in continuous trading limit GFN. (0011-MARA-024)
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFN | sell1     |

    And "party5" should have holding account balance of "1" for asset "BTC"

  Scenario: In Spot market, holding in holding account is correctly calculated for all order types in continuous trading pegged GTT. (0011-MARA-025)
    Given the following network parameters are set:
      | name                           | value |
      | limits.markets.maxPeggedOrders | 10    |

    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party2 | BTC/ETH   | sell | 5      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |

    And the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset |
      | party5 | BTC/ETH   | sell | 1      | ASK              | 100    |

    Then "party5" should have holding account balance of "1" for asset "BTC"

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 1000  | 0      |
      
    #0068-MATC-072, Incoming limit GTT orders match if possible, any remaining is placed on the book.
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 2          | buy2      |
      | party5 | BTC/ETH   | sell | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTT | 2          | sell2     |

    Then the orders should have the following status:
      | party  | reference | status        |
      | party1 | buy2      | STATUS_FILLED |
      | party5 | sell2     | STATUS_ACTIVE |
    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | sell | 1000  | 1      |

    When the network moves ahead "2" blocks
    Then the orders should have the following status:
      | party  | reference | status         |
      | party5 | sell2     | STATUS_EXPIRED |

  Scenario: In Spot market, holding in holding account is correctly calculated for all order types in continuous trading pegged GTC. (0011-MARA-026)
    Given the following network parameters are set:
      | name                           | value |
      | limits.markets.maxPeggedOrders | 10    |

    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party2 | BTC/ETH   | sell | 5      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |

    When the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset |
      | party5 | BTC/ETH   | sell | 1      | ASK              | 100    |

    Then "party5" should have holding account balance of "1" for asset "BTC"

  Scenario: In Spot market, holding in holding account is correctly calculated for all order types in continuous trading pegged GFN. (0011-MARA-027)
    Given the following network parameters are set:
      | name                           | value |
      | limits.markets.maxPeggedOrders | 10    |

    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party2 | BTC/ETH   | sell | 5      | 1000000 | 0                | TYPE_LIMIT | TIF_GFN | t2-s-1    |

    When the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset |
      | party5 | BTC/ETH   | sell | 1      | ASK              | 100    |

    Then "party5" should have holding account balance of "1" for asset "BTC"

