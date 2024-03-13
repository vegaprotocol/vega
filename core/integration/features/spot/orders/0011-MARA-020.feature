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
      | party3 | ETH   | 10000  |
      | party5 | BTC   | 100    |
    And the average block duration is "1"
    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GFA | p1-gfa    |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |           |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"


  Scenario: In Spot Market, cancelling an order releases the holding amount back to user's general account. (0011-MARA-020)

    # Place a buy order
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | BTC/ETH   | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |

    And "party5" should have general account balance of "94" for asset "BTC"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | buy1      |

    #0039-MKTD-033 GFA order leaving auction should be cancelled
    Then the orders should have the following status:
      | party  | reference | status           |
      | party1 | buy1      | STATUS_FILLED    |
      | party1 | p1-gfa    | STATUS_CANCELLED |
      | party5 | sell1     | STATUS_ACTIVE    |

    Then the parties cancel the following orders:
      | party  | reference |
      | party5 | sell1     |

    And "party5" should have general account balance of "98" for asset "BTC"
