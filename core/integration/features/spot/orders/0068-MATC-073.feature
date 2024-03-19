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
      | party1 | BTC/ETH   | buy  | 1      | 998   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | BTC/ETH   | sell | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

    Scenario: For Good 'Til Time (GTT) / Good 'Till Cancelled (GTC) / Good For Normal (GFN) orders: 
              Incoming LIMIT: POST-ONLY TRUE orders will be placed fully on the book if no orders currently cross. (0068-MATC-073)

    And the parties place the following orders:
        | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | reference | only |
        | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTT | 3600       | buy-gtt   | post |
        | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC |            | buy-gtc   | post |
        | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GFN |            | buy-gfn   | post |
        | party5 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTT | 3600       | sell-gtt  | post |
        | party5 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC |            | sell-gtc  | post |
        | party5 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GFN |            | sell-gfn  | post |

    Then the orders should have the following status:
        | party  | reference  | status        |
        | party1 | buy-gtt    | STATUS_ACTIVE |
        | party1 | buy-gtc    | STATUS_ACTIVE |
        | party1 | buy-gfn    | STATUS_ACTIVE |
        | party5 | sell-gtt   | STATUS_ACTIVE |
        | party5 | sell-gtc   | STATUS_ACTIVE |
        | party5 | sell-gfn   | STATUS_ACTIVE |

