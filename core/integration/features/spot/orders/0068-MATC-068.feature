Feature: Spot market FOK limit and market order

  Background:

    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.value.windowLength               | 1h    |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 0              |
      | BTC | 0              |

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
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 0              | 0                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 40000  |
      | party2 | ETH   | 10000  |
      | party3 | ETH   | 10000  |
      | party4 | BTC   | 900    |
      | party5 | BTC   | 900    |
    And the average block duration is "1"
    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

  Scenario: test FOK limit order and market order (0068-MATC-067, 0068-MATC-068, 0068-MATC-069)

    # Place a buy order
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | reference | error |
      | party1 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | buy1      |       |
      | party5 | BTC/ETH   | sell | 5      | 1000  | 1                | TYPE_MARKET | TIF_FOK | sell1     |       |
      | party5 | BTC/ETH   | sell | 15     | 1000  | 0                | TYPE_MARKET | TIF_FOK | sell2     |       |
      | party4 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT  | TIF_FOK | sell3     |       |
      | party4 | BTC/ETH   | sell | 6      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK | sell4     |       |

    And "party5" should have general account balance of "894" for asset "BTC"
    And "party5" should have holding account balance of "0" for asset "BTC"
    And "party4" should have general account balance of "899" for asset "BTC"

    #0068-MATC-067,Incoming MARKET orders will be matched fully if the volume is available, otherwise the order is cancelled.
    #0068-MATC-068,Incoming FOK limit order will be fully matched if possible to the other side of the book  
    #0068-MATC-069,for incoming FOK limit order, if a complete fill is not possible the order is stopped without trading at all.
    Then the orders should have the following status:
      | party  | reference | status         |
      | party1 | buy1      | STATUS_ACTIVE  |
      | party5 | sell1     | STATUS_FILLED  |
      | party5 | sell2     | STATUS_STOPPED |
      | party4 | sell3     | STATUS_FILLED  |
      | party4 | sell4     | STATUS_STOPPED |

    And "party4" should have general account balance of "899" for asset "BTC"
    And "party5" should have general account balance of "894" for asset "BTC"
