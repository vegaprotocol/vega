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

  Scenario: In Spot market, holding in holding account s correctly calculated for all order types in auction mode limit GTT (0011-MARA-028)
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 50         | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 50         | sell1     |

    Then "party5" should have holding account balance of "1" for asset "BTC"
    And "party1" should have holding account balance of "10" for asset "ETH"

  Scenario: In Spot market, holding in holding account s correctly calculated for all order types in auction mode limit GTC (0011-MARA-029)
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    Then "party5" should have holding account balance of "1" for asset "BTC"
    And "party1" should have holding account balance of "10" for asset "ETH"

  Scenario: In Spot market, holding in holding account s correctly calculated for all order types in auction mode limit GFA (0011-MARA-030)
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | sell1     |

    Then "party5" should have holding account balance of "1" for asset "BTC"
    And "party1" should have holding account balance of "10" for asset "ETH"

  Scenario: In Spot market, holding in holding account s correctly calculated for all order types in auction mode pegged GTT (0011-MARA-031)
    Given the following network parameters are set:
      | name                           | value |
      | limits.markets.maxPeggedOrders | 10    |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 50         | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 50         | sell1     |

    When the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset | reference |
      | party2 | BTC/ETH   | sell | 1      | ASK              | 100    | pegged1   |

    Then "party5" should have holding account balance of "1" for asset "BTC"
    And "party1" should have holding account balance of "10" for asset "ETH"

    And the orders should have the following status:
      | party  | reference | status        |
      | party2 | pegged1   | STATUS_PARKED |

  Scenario: In Spot market, holding in holding account s correctly calculated for all order types in auction mode pegged GTC (0011-MARA-032)
    Given the following network parameters are set:
      | name                           | value |
      | limits.markets.maxPeggedOrders | 10    |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    Then "party5" should have holding account balance of "1" for asset "BTC"
    And "party1" should have holding account balance of "10" for asset "ETH"

    When the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset | reference |
      | party2 | BTC/ETH   | sell | 1      | ASK              | 100    | pegged1   |

    And the orders should have the following status:
      | party  | reference | status        |
      | party2 | pegged1   | STATUS_PARKED |
