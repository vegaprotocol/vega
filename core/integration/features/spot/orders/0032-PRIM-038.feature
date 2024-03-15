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
      | 30      | 0.999       | 10                |
      | 60      | 0.999       | 10                |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party1 | BTC   | 1000   |
      | party2 | ETH   | 10000  |
      | party4 | BTC   | 1000   |
      | party5 | BTC   | 1000   |
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

  Scenario: Same as PRIM-038, but more matching orders get placed during the auction extension. The volume of the trades generated
            by the later orders is larger than that of the original pair which triggered the auction. Hence the auction concludes
            generating the trades from the later orders. The overall auction duration is equal to the sum of the extension periods
            of the two triggers. (0032-PRIM-038)

    # Check that the market price bounds are set 
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 30      | 997       | 1003      | 0            | 0              | 0             |
      | 1000       | TRADING_MODE_CONTINUOUS | 60      | 995       | 1005      | 0            | 0              | 0             |

    # Place 2 persistent orders that are outside both price bounds
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1006  | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1006  | 0                | TYPE_LIMIT | TIF_GTC | sell1     |
    When the network moves ahead "1" blocks

    Then "party1" should have holding account balance of "11" for asset "ETH"
    Then "party5" should have holding account balance of "1" for asset "BTC"

    # Check we have been placed in auction
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    # If we move forward 10 blocks we should still be in auction due to the second extension
    When the network moves ahead "10" blocks
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    # Now place some larger orders while in the auction extension
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | BTC/ETH   | buy  | 5      | 1007  | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
      | party4 | BTC/ETH   | sell | 5      | 1007  | 0                | TYPE_LIMIT | TIF_GTC | sell2     |

    # DEBUGGING
    Then "party2" should have holding account balance of "51" for asset "ETH"
    Then "party4" should have holding account balance of "5" for asset "BTC"

    # If we move forward 10 more blocks we should leave the auction
    When the network moves ahead "10" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # The mark price should show the orders have traded
    And the mark price should be "1007" for the market "BTC/ETH"

   And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status         |
    | party1 | BTC/ETH   | buy1      | buy  | 1      | 1         | 1006  | STATUS_ACTIVE  |
    | party5 | BTC/ETH   | sell1     | sell | 1      | 0         | 1006  | STATUS_FILLED  |
    | party2 | BTC/ETH   | buy2      | buy  | 5      | 0         | 1007  | STATUS_FILLED  |
    | party4 | BTC/ETH   | sell2     | sell | 5      | 1         | 1007  | STATUS_ACTIVE  |

