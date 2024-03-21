Feature: If an instruction causes the market to enter a Price Monitoring Auction the market should enter the auction immediately
         before continuing with later instructions. (0074-BTCH-015)

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
      | 3600    | 0.999       | 10                |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | ETH   | 10000  |
      | party4 | BTC   | 100    |
      | party5 | BTC   | 100    |
    And the average block duration is "1"

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | BTC/ETH   | sell | 2      | 1050  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 959       | 1042      | 0            | 0              | 0             |

  Scenario: Placing an order outside of price bands will force the market to go into auction before the order is traded. 
            All other orders after the out of bounds order will be placed in the auction.

    Then the party "party1" starts a batch instruction
      Then the party "party1" adds the following orders to a batch:
        | market id | side | volume | price | type        | tif     | reference |
        | BTC/ETH   | buy  | 1      | 1050  | TYPE_LIMIT  | TIF_GTC | p1-buy1   |
        | BTC/ETH   | buy  | 1      | 1060  | TYPE_LIMIT  | TIF_GTC | p1-buy2   |
    Then the party "party1" submits their batch instruction
    When the network moves ahead "1" blocks

    # `p1-buy1` forced us into an auction
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    # Both orders rest on the book in the auction instead of crossing
    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status         |
      | party1 | BTC/ETH   | buy  | 1050  | 1         | 1      | p1-buy1   | STATUS_ACTIVE  | 
      | party1 | BTC/ETH   | buy  | 1060  | 1         | 1      | p1-buy2   | STATUS_ACTIVE  | 




