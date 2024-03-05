Feature: Spot market

  Background:
    Given time is updated to "2024-01-01T00:00:00Z"

    Given the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.value.windowLength                           | 1h    |
    
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
      | party4 | BTC   | 100  |
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
    And the mark price should be "1000" for the market "BTC/ETH"

  Scenario: For an iceberg order sitting on the book, when a batch containing normal orders and other iceberg orders that will trades
            against the existing iceberg sitting on the book is sent, the resting iceberg order refreshes between each order in the
            batch (0014-ORDT-095)

    # Place an iceberg on the book ready to be matched by the later batch
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only | reference |
      | party5 | BTC/ETH   | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | 2         | 2                    | post | iceberg1  |

    # Create a batch with iceberg and normal orders
    Then the party "party1" starts a batch instruction

    Then the party "party1" adds the following orders to a batch:
      | market id | side | volume | price | type        | tif     | reference |
      | BTC/ETH   | buy  | 2      | 1000  | TYPE_LIMIT  | TIF_GTC | p1-buy1   |

    Then the party "party1" adds the following iceberg orders to a batch:
      | market id | side | volume | price | type       | tif     | reference | peak size | minimum visible size |
      | BTC/ETH   | buy  | 5      | 1000  | TYPE_LIMIT | TIF_GTC | iceberg2  | 2         | 1                    |

    Then the party "party1" adds the following orders to a batch:
      | market id | side | volume | price | type        | tif     | reference |
      | BTC/ETH   | buy  | 3      | 1000  | TYPE_LIMIT  | TIF_GTC | p1-buy2   |

    Then the party "party1" submits their batch instruction

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party5 | 1000  | 2    |
      | party1 | party5 | 1000  | 5    |
      | party1 | party5 | 1000  | 3    |

    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status         |
      | party1 | BTC/ETH   | buy  | 1000  | 0         | 2      | p1-buy1   | STATUS_FILLED  | 
      | party1 | BTC/ETH   | buy  | 1000  | 0         | 5      | iceberg2  | STATUS_FILLED  | 
      | party1 | BTC/ETH   | buy  | 1000  | 0         | 3      | p1-buy2   | STATUS_FILLED  | 
      | party5 | BTC/ETH   | sell | 1000  | 0         | 10     | iceberg1  | STATUS_FILLED  | 




