Feature: Spot market

  Background:
    Given time is updated to "2024-01-01T00:00:00Z"

#    Given the following network parameters are set:
#      | name                                                | value |
#      | network.markPriceUpdateMaximumFrequency             | 0s    |
#      | market.value.windowLength                           | 1h    |
    
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 3              |
      | BTC | 3              |

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
      | party3 | BTC   | 100    |
      | party4 | BTC   | 100    |
      | party5 | BTC   | 100    |
    And the average block duration is "1"

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party5 | BTC/ETH   | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then "party1" should have holding account balance of "100" for asset "ETH"
    Then "party5" should have holding account balance of "100" for asset "BTC"

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

  Scenario: In Spot market, for multiple iceberg orders submitted as a batch of orders with a mix of ordinary limit orders and market orders,
            the iceberg orders are processed atomically and the order book volume and price, holding calculations , order status are all
            correct. (0014-ORDT-094)


    # Party1 has no pending buys so their holding account will be empty
    Then "party1" should have holding account balance of "0" for asset "ETH"
    # Party5 has 9 pending sells so they will have 9 in their holding account
    Then "party5" should have holding account balance of "90" for asset "BTC"


    # Create a batch with iceberg and normal orders
    Then the party "party1" starts a batch instruction

    Then the party "party1" adds the following orders to a batch:
      | market id | side | volume | price | type        | tif     | reference | expires in |
      | BTC/ETH   | buy  | 1      | 0     | TYPE_MARKET | TIF_IOC | p1-buy1   |            |
      | BTC/ETH   | buy  | 1      | 999   | TYPE_LIMIT  | TIF_GTC | p1-buy2   |            |
      | BTC/ETH   | buy  | 1      | 999   | TYPE_LIMIT  | TIF_GFN | p1-buy3   |            |
      | BTC/ETH   | buy  | 1      | 999   | TYPE_LIMIT  | TIF_GTT | p1-buy4   | 3600       |

    Then the party "party1" adds the following iceberg orders to a batch:
      | market id | side | volume | price | type       | tif     | reference | peak size | minimum visible size |
      | BTC/ETH   | buy  | 4      | 999   | TYPE_LIMIT | TIF_GTC | p1-ib-1   | 2         | 1                    |
      | BTC/ETH   | buy  | 4      | 1000  | TYPE_LIMIT | TIF_GTC | p1-ib-2   | 2         | 1                    |

    Then the party "party1" submits their batch instruction

    # 2 trades should occur
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party5 | 1000  | 1    |
      | party1 | party5 | 1000  | 4    |

    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status          |
      | party1 | BTC/ETH   | buy  | 0     | 0         | 1      | p1-buy1   | STATUS_FILLED  | 
      | party1 | BTC/ETH   | buy  | 999   | 1         | 1      | p1-buy2   | STATUS_ACTIVE  | 
      | party1 | BTC/ETH   | buy  | 999   | 1         | 1      | p1-buy3   | STATUS_ACTIVE  | 
      | party1 | BTC/ETH   | buy  | 999   | 1         | 1      | p1-buy4   | STATUS_ACTIVE  | 
      | party1 | BTC/ETH   | buy  | 999   | 2         | 4      | p1-ib-1   | STATUS_ACTIVE  | 
      | party1 | BTC/ETH   | buy  | 1000  | 0         | 4      | p1-ib-2   | STATUS_FILLED  | 

    Then "party1" should have holding account balance of "696" for asset "ETH"
    Then "party5" should have holding account balance of "40" for asset "BTC"



  Scenario: In Spot market, for multiple iceberg orders submitted as a batch of orders with a mix of ordinary limit orders and market orders,
            the iceberg orders are processed atomically and the order book volume and price, holding calculations , order status are all
            correct. (0014-ORDT-094)


    # Party1 has no pending buys so their holding account will be empty
    Then "party1" should have holding account balance of "0" for asset "ETH"
    # Party5 has 9 pending sells so they will have 9 in their holding account
    Then "party5" should have holding account balance of "90" for asset "BTC"

    # Create a batch with only normal orders to make sure we get the same holding account values at the end
    Then the party "party1" starts a batch instruction

    Then the party "party1" adds the following orders to a batch:
      | market id | side | volume | price | type        | tif     | reference | expires in |
      | BTC/ETH   | buy  | 1      | 0     | TYPE_MARKET | TIF_IOC | p1-buy1   |            |
      | BTC/ETH   | buy  | 1      | 999   | TYPE_LIMIT  | TIF_GTC | p1-buy2   |            |
      | BTC/ETH   | buy  | 1      | 999   | TYPE_LIMIT  | TIF_GFN | p1-buy3   |            |
      | BTC/ETH   | buy  | 1      | 999   | TYPE_LIMIT  | TIF_GTT | p1-buy4   | 3600       |
      | BTC/ETH   | buy  | 4      | 999   | TYPE_LIMIT  | TIF_GTC | p1-buy5   |            |
      | BTC/ETH   | buy  | 4      | 1000  | TYPE_LIMIT  | TIF_GTC | p1-buy6   |            |

    Then the party "party1" submits their batch instruction

    # 2 trades should occur
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party5 | 1000  | 1    |
      | party1 | party5 | 1000  | 4    |

    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status         |
      | party1 | BTC/ETH   | buy  | 0     | 0         | 1      | p1-buy1   | STATUS_FILLED  | 
      | party1 | BTC/ETH   | buy  | 999   | 1         | 1      | p1-buy2   | STATUS_ACTIVE  | 
      | party1 | BTC/ETH   | buy  | 999   | 1         | 1      | p1-buy3   | STATUS_ACTIVE  | 
      | party1 | BTC/ETH   | buy  | 999   | 1         | 1      | p1-buy4   | STATUS_ACTIVE  | 
      | party1 | BTC/ETH   | buy  | 999   | 4         | 4      | p1-buy5   | STATUS_ACTIVE  | 
      | party1 | BTC/ETH   | buy  | 1000  | 0         | 4      | p1-buy6   | STATUS_FILLED  | 

    Then "party1" should have holding account balance of "696" for asset "ETH"
    Then "party5" should have holding account balance of "40" for asset "BTC"
