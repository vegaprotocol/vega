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
      | 3600    | 0.999       | 10                |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party1 | BTC   | 1000   |
      | party2 | ETH   | 10000  |
      | party2 | BTC   | 1000   |
      | party5 | ETH   | 10000  |
      | party5 | BTC   | 1000   |
    And the average block duration is "1"

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 959       | 1042      | 0            | 0              | 0             |

    Scenario: For Good 'Til Time (GTT) orders: All attempts to self trade are prevented and the aggressive
              side is STOPPED even if partially filled. The passive side is left untouched. (0068-MATC-077)

      # Place some orders and try to self trade
      And the parties place the following orders:
        | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference |
        | party2 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |            | sell-gtc1 |
        | party1 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |            | sell-gtc2 |
        | party2 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 3600       | buy-gtt1  |
        | party1 | BTC/ETH   | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTT | 3600       | buy-gtt2  |

      Then the orders should have the following status:
        | party  | reference  | status                  |
        | party2 | sell-gtc1  | STATUS_FILLED           |
        | party1 | sell-gtc2  | STATUS_ACTIVE           |
        | party2 | buy-gtt1   | STATUS_STOPPED          |
        | party1 | buy-gtt2   | STATUS_PARTIALLY_FILLED |


    Scenario: For Good 'Till Cancelled (GTC) orders: All attempts to self trade are prevented and the aggressive
              side is STOPPED even if partially filled. The passive side is left untouched. (0068-MATC-077)

      # Place some orders and try to self trade
      And the parties place the following orders:
        | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
        | party2 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-gtc1 |
        | party1 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-gtc2 |
        | party2 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-gtc1  |
        | party1 | BTC/ETH   | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | buy-gtc2  |

      Then the orders should have the following status:
        | party  | reference  | status                  |
        | party2 | sell-gtc1  | STATUS_FILLED           |
        | party1 | sell-gtc2  | STATUS_ACTIVE           |
        | party2 | buy-gtc1   | STATUS_STOPPED          |
        | party1 | buy-gtc2   | STATUS_PARTIALLY_FILLED |


    Scenario: For Good For Normal (GFN) orders: All attempts to self trade are prevented and the aggressive
              side is STOPPED even if partially filled. The passive side is left untouched. (0068-MATC-077)

      # Place some orders and try to self trade
      And the parties place the following orders:
        | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
        | party2 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-gtc1 |
        | party1 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-gtc2 |
        | party2 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFN | buy-gfn1  |
        | party1 | BTC/ETH   | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GFN | buy-gfn2  |

      Then the orders should have the following status:
        | party  | reference  | status                  |
        | party2 | sell-gtc1  | STATUS_FILLED           |
        | party1 | sell-gtc2  | STATUS_ACTIVE           |
        | party2 | buy-gfn1   | STATUS_STOPPED          |
        | party1 | buy-gfn2   | STATUS_PARTIALLY_FILLED |
