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
      | 60      | 0.999       | 1                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | ETH   | 10000  |
      | party4 | BTC   | 1000   |
      | party5 | BTC   | 1000   |
    And the average block duration is "1"

  Scenario: As the Vega network, in auction mode, all orders are placed in the book but never uncross
            until the end of the auction period. (0026-AUCT-024)

    # Place some orders that cross
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA | buy1      |
      | party2 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
      | party4 | BTC/ETH   | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |
      | party5 | BTC/ETH   | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA | sell2     |

    When the network moves ahead "1" blocks

    # Check that nothing has traded
  And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status        |
    | party1 | BTC/ETH   | buy1      | buy  | 10     | 10        | 1000  | STATUS_ACTIVE |
    | party2 | BTC/ETH   | buy2      | buy  | 10     | 10        | 1000  | STATUS_ACTIVE |
    | party4 | BTC/ETH   | sell1     | sell | 10     | 10        | 1000  | STATUS_ACTIVE |
    | party5 | BTC/ETH   | sell2     | sell | 10     | 10        | 1000  | STATUS_ACTIVE |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

  # Now we have moved out of auction, check the trades have taken place
  Then the following trades should be executed:
    | buyer  | seller | price | size |
    | party1 | party4 | 1000  | 10   |
    | party2 | party5 | 1000  | 10   |

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 60      | 995       | 1005      | 0            | 0              | 0             |


  # Now move into a price monitoring auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 10     | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy3      |
      | party5 | BTC/ETH   | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell3     |
      | party2 | BTC/ETH   | buy  | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | buy4      |
 
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

  # Place some crossing orders and make sure they do not trade
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy5      |
      | party5 | BTC/ETH   | sell | 11     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell5     |
  
  Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
  
  And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status        |
    | party1 | BTC/ETH   | buy5      | buy  | 10     | 10        | 1000  | STATUS_ACTIVE |
    | party5 | BTC/ETH   | sell5     | sell | 11     | 11        | 1000  | STATUS_ACTIVE |
    | party1 | BTC/ETH   | buy3      | buy  | 10     | 10        | 990   | STATUS_ACTIVE |
    | party5 | BTC/ETH   | sell3     | sell | 10     | 10        | 1010  | STATUS_ACTIVE |
    | party2 | BTC/ETH   | buy4      | buy  | 1      | 1         | 1010  | STATUS_ACTIVE |

  # Now move forward until we come out of auction and make sure the crossed orders are matched and trade correctly

  When the network moves ahead "5" blocks
  Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

  And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status        |
    | party1 | BTC/ETH   | buy5      | buy  | 10     | 0         | 1000  | STATUS_FILLED |
    | party5 | BTC/ETH   | sell5     | sell | 11     | 0         | 1000  | STATUS_FILLED |


