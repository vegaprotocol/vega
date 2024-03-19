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

  Scenario: When entering an auction, all GFN orders will be cancelled. (0026-AUCT-031)

    # Place some orders that cross so we can leave the auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | sell1     |
    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

    # Place some GFN orders into the orderbook
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 10     | 980   | 0                | TYPE_LIMIT | TIF_GFN | buy2      |
      | party2 | BTC/ETH   | buy  | 10     | 985   | 0                | TYPE_LIMIT | TIF_GFN | buy3      |
      | party4 | BTC/ETH   | sell | 10     | 1015  | 0                | TYPE_LIMIT | TIF_GFN | sell2     |
      | party5 | BTC/ETH   | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GFN | sell3     |
      | party1 | BTC/ETH   | buy  | 1      | 1015  | 0                | TYPE_LIMIT | TIF_GTC | buy4     |
 
    When the network moves ahead "1" blocks

    # Check we are now in monitoring auction
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    # Check that all the GFN orders have been cancelled
  And the orders should have the following states:
    | party  | market id | reference | side | volume | remaining | price | status        |
    | party1 | BTC/ETH   | buy2      | buy  | 10     | 10        | 980   | STATUS_CANCELLED |
    | party2 | BTC/ETH   | buy3      | buy  | 10     | 10        | 985   | STATUS_CANCELLED |
    | party4 | BTC/ETH   | sell2     | sell | 10     | 10        | 1015  | STATUS_CANCELLED |
    | party5 | BTC/ETH   | sell3     | sell | 10     | 10        | 1020  | STATUS_CANCELLED |


  
