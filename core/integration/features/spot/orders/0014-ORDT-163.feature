Feature: stop order in spot market

  Background:
    Given time is updated to "2024-01-01T00:00:00Z"

    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 1s    |
      | market.value.windowLength               | 1h    |
      | spam.protection.max.stopOrdersPerMarket | 5     |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
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
      | party1 | ETH   | 100    |
      | party1 | BTC   | 11     |
      | party2 | ETH   | 10000  |
      | party2 | BTC   | 10     |
      | party3 | ETH   | 10000  |
      | party3 | BTC   | 1000   |
      | party4 | BTC   | 1000   |
      | party5 | BTC   | 1000   |
    And the average block duration is "1"

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party4 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

  Scenario:0014-ORDT-163, 0014-ORDT-164: A wrapped buy/sell order will be rejected when triggered if the party doesn't have enough of the required quote asset to cover the order.

    # place an order to match with the limit order then check the stop is filled
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | BTC/ETH   | sell | 50     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | p4-sell   |

    # create party1 stop order
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | only | ra price trigger | error | reference |
      | party1 | BTC/ETH   | buy  | 50     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |      | 1005             |       | stop1     |

    # now we trade at 1005, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 1      | 1005  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | BTC/ETH   | sell | 1      | 1005  | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order was triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | BTC/ETH   | STATUS_TRIGGERED | stop1     |

    Then "party1" should have general account balance of "100" for asset "ETH"
    Then "party1" should have general account balance of "11" for asset "BTC"

    And the parties cancel the following orders:
      | party  | reference |
      | party4 | p4-sell   |

    # place an order to match with the limit order then check the stop is filled
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | BTC/ETH   | buy  | 50     | 1015  | 0                | TYPE_LIMIT | TIF_GTC | p4-sell   |

    # create party2 stop order
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | only | ra price trigger | error | reference |
      | party2 | BTC/ETH   | sell | 50     | 1015  | 0                | TYPE_LIMIT | TIF_GTC |      | 1020             |       | stop2     |

    # now we trade at 1005, this will breach the trigger
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | BTC/ETH   | buy  | 1      | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | BTC/ETH   | sell | 1      | 1020  | 1                | TYPE_LIMIT | TIF_GTC |

    # check that the order was triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party2 | BTC/ETH   | STATUS_TRIGGERED | stop2     |

    Then "party2" should have general account balance of "10000" for asset "ETH"
    Then "party2" should have general account balance of "10" for asset "BTC"


